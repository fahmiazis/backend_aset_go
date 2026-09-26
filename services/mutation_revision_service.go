package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
)

// ReturnMutationForRevision mengembalikan pengajuan mutasi ke DRAFT supaya
// pembuatnya bisa memperbaiki aset yang ditandai lalu submit ulang.
func ReturnMutationForRevision(
	userID string,
	transactionNumber string,
	req dto.ReturnForRevisionRequest,
) (*dto.MutationDetailResponse, error) {
	transaction, err := getMutationTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	// Hanya dari stage approval. Setelah barang mulai diserahterimakan dan
	// dieksekusi, perpindahannya tidak bisa dibatalkan hanya dengan mengubah
	// stage.
	if transaction.CurrentStage != models.StageMutationApproval {
		return nil, fmt.Errorf("cannot revise transaction in %s stage", transaction.CurrentStage)
	}

	if err := assertCurrentApprover(userID, transactionNumber, TxMutationFlow); err != nil {
		return nil, err
	}

	// Aset yang ditandai harus benar-benar milik transaksi ini dan masih aktif
	var targets []models.TransactionMutationAsset
	if err := config.DB.
		Where("transaction_id = ? AND id IN ? AND status = ?",
			transaction.ID, req.RowIDs, models.MutationAssetStatusPending).
		Find(&targets).Error; err != nil {
		return nil, err
	}
	if len(targets) != len(req.RowIDs) {
		return nil, errors.New("some selected assets do not belong to this transaction or are no longer active")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := markRowsForRevision(tx, &models.TransactionMutationAsset{},
		transaction.ID, req.RowIDs, req.RevisionNotes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := resetApprovalsForRevision(tx, transactionNumber, TxMutationFlow); err != nil {
		tx.Rollback()
		return nil, err
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageMutationDraft); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageMutationDraft,
		models.ActionRevise, userID, nil, &req.RevisionNotes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetMutationDetail(transactionNumber)
}

// CancelMutation membatalkan pengajuan atas permintaan PEMBUATNYA sendiri.
func CancelMutation(
	userID string,
	transactionNumber string,
	req dto.CancelTransactionRequest,
) (*dto.MutationDetailResponse, error) {
	transaction, err := getMutationTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("only the creator can cancel this transaction")
	}

	// Mulai serah terima sudah ada dokumen dan perpindahan fisik yang tidak
	// bisa dibatalkan hanya dengan memindahkan stage.
	cancelableStages := map[string]bool{
		models.StageMutationDraft:    true,
		models.StageMutationApproval: true,
	}
	if !cancelableStages[transaction.CurrentStage] {
		return nil, fmt.Errorf(
			"cannot cancel transaction in %s stage, it is already being executed",
			transaction.CurrentStage)
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("transaction_number = ? AND transaction_type = ?",
		transactionNumber, TxMutationFlow).
		Delete(&models.TransactionApproval{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Aset dilepas kembali supaya tidak tertahan di mutasi yang batal.
	// Status asetnya ikut dipulihkan ke AVAILABLE (sama seperti RejectMutation)
	// — dulu hanya barisnya yang ditandai CANCELLED, sehingga asetnya tertahan
	// IN_MUTATION selamanya dan tidak bisa dipilih di transaksi mana pun.
	if err := tx.Model(&models.Asset{}).
		Where("id IN (?)", tx.Model(&models.TransactionMutationAsset{}).
			Select("asset_id").
			Where("transaction_id = ? AND status = ?", transaction.ID, models.MutationAssetStatusPending)).
		Where("asset_status = ?", models.AssetStatusInMutation).
		Update("asset_status", models.AssetStatusAvailable).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Model(&models.TransactionMutationAsset{}).
		Where("transaction_id = ? AND status = ?",
			transaction.ID, models.MutationAssetStatusPending).
		Update("status", models.MutationAssetStatusCancelled).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageMutationCancelled); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageMutationCancelled,
		models.ActionCancel, userID, nil, &req.Reason); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	MarkTransactionAsExpired(transactionNumber)

	return GetMutationDetail(transactionNumber)
}
