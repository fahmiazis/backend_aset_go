package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
)

// ReturnProcurementForRevision mengembalikan pengajuan ke DRAFT supaya
// pembuatnya bisa memperbaiki item yang ditandai lalu submit ulang.
//
// Berbeda dengan ReviseProcurement yang sudah ada — itu perubahan item oleh
// PENGAJU tanpa memindahkan stage. Yang ini permintaan dari APPROVER yang
// sedang memegang step berjalan.
func ReturnProcurementForRevision(
	userID string,
	transactionNumber string,
	req dto.ReturnForRevisionRequest,
) (*dto.ProcurementDetailWithStageResponse, error) {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	// Hanya dari stage approval. Setelah itu sudah ada efek samping (anggaran
	// diproses, aset dieksekusi, barang diterima) yang tidak bisa dibatalkan
	// hanya dengan memindahkan stage.
	if transaction.CurrentStage != models.StageApproval {
		return nil, fmt.Errorf("cannot revise transaction in %s stage", transaction.CurrentStage)
	}

	if err := assertCurrentApprover(userID, transactionNumber, TxProcurement); err != nil {
		return nil, err
	}

	// Item yang ditandai harus benar-benar milik transaksi ini
	var targets []models.TransactionProcurement
	if err := config.DB.
		Where("transaction_id = ? AND id IN ?", transaction.ID, req.RowIDs).
		Find(&targets).Error; err != nil {
		return nil, err
	}
	if len(targets) != len(req.RowIDs) {
		return nil, errors.New("some selected items do not belong to this transaction")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := markRowsForRevision(tx, &models.TransactionProcurement{},
		transaction.ID, req.RowIDs, req.RevisionNotes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := resetApprovalsForRevision(tx, transactionNumber, TxProcurement); err != nil {
		tx.Rollback()
		return nil, err
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageDraft); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageDraft,
		models.ActionRevise, userID, nil, &req.RevisionNotes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetProcurementDetailWithStage(transactionNumber)
}

// CancelProcurement membatalkan pengajuan atas permintaan PEMBUATNYA sendiri.
// Beda aktor dan beda stage akhir dengan RejectProcurement, yang dilakukan
// orang yang sedang memegang stage berjalan.
func CancelProcurement(
	userID string,
	transactionNumber string,
	req dto.CancelTransactionRequest,
) (*dto.ProcurementDetailWithStageResponse, error) {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("only the creator can cancel this transaction")
	}

	// Mulai PROCESS_BUDGET sudah ada efek samping (anggaran, aset dibuat,
	// barang diterima) yang tidak bisa dibatalkan hanya dengan memindahkan
	// stage.
	cancelableStages := map[string]bool{
		models.StageDraft:             true,
		models.StageAssetVerification: true,
		models.StageApproval:          true,
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

	// Approval yang sudah terbentuk ikut dibuang supaya tidak menggantung
	if err := tx.Where("transaction_number = ? AND transaction_type = ?",
		transactionNumber, TxProcurement).
		Delete(&models.TransactionApproval{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageCancelled); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageCancelled,
		models.ActionCancel, userID, nil, &req.Reason); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Nomor transaksi dilepas supaya tidak menahan reservoir
	MarkTransactionAsExpired(transactionNumber)

	return GetProcurementDetailWithStage(transactionNumber)
}
