package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
	"strings"
)

// ============================================================
// REVISI AGREEMENT
//
// Approver step berjalan mengeluarkan disposal anggota yang bermasalah.
// Disposal itu kembali ke DRAFT miliknya sendiri — persis seperti revisi
// disposal biasa (seluruh aset aktifnya ditandai perlu revisi, approval
// request diulang dari awal saat disubmit lagi). Sisa anggota tetap di
// agreement dan approval agreement berjalan terus tanpa diulang.
//
// Minimal satu anggota harus tersisa; kalau semuanya bermasalah, tolak
// agreement-nya (anggota otomatis layak dikelompokkan lagi).
// ============================================================

// agreementRevisionNote — catatan stage di disposal anggota. Awalannya tetap
// supaya bisa dicari lagi (agreementRevisedRecently di email).
func agreementRevisionNote(agreementNumber, notes string) string {
	return fmt.Sprintf("Dikembalikan dari kesepakatan %s: %s", agreementNumber, notes)
}

func ReviseDisposalAgreement(userID, agreementNumber string, req dto.ReviseDisposalAgreementRequest) (*dto.DisposalAgreementResponse, error) {
	agreement, err := GetDisposalAgreementByNumber(agreementNumber)
	if err != nil {
		return nil, err
	}
	if agreement.CurrentStage != models.StageAgreementApproval {
		return nil, fmt.Errorf("cannot revise agreement in %s stage", agreement.CurrentStage)
	}

	// hanya approver step berjalan — aturan sama dengan ApproveTransaction
	if err := assertCurrentApprover(userID, agreementNumber, TxDisposalAgreement); err != nil {
		return nil, err
	}
	if err := validateAgreementApproverBranches(userID, agreementNumber); err != nil {
		return nil, err
	}

	var items []models.DisposalAgreementItem
	if err := config.DB.Where("agreement_id = ?", agreement.ID).Find(&items).Error; err != nil {
		return nil, err
	}

	members := map[string]models.DisposalAgreementItem{}
	for _, item := range items {
		members[item.TransactionNumber] = item
	}

	targets := make([]models.DisposalAgreementItem, 0, len(req.TransactionNumbers))
	seen := map[string]bool{}
	for _, number := range req.TransactionNumbers {
		if seen[number] {
			continue
		}
		seen[number] = true
		item, ok := members[number]
		if !ok {
			return nil, fmt.Errorf("transaction %s is not a member of this agreement", number)
		}
		targets = append(targets, item)
	}

	if len(targets) >= len(items) {
		return nil, errors.New("at least one member must remain in the agreement — reject the agreement instead to return all members")
	}

	notes := strings.TrimSpace(req.RevisionNotes)

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, item := range targets {
		var transaction models.Transaction
		if err := tx.First(&transaction, item.TransactionID).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		if transaction.CurrentStage != models.StageDisposalApprovalAgreement {
			tx.Rollback()
			return nil, fmt.Errorf("transaction %s is no longer waiting for agreement (%s)",
				transaction.TransactionNumber, transaction.CurrentStage)
		}

		// keluar dari agreement
		if err := tx.Delete(&models.DisposalAgreementItem{}, item.ID).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// Approver agreement menilai per transaksi, bukan per aset, jadi seluruh
		// aset aktifnya yang ditandai. Penanda ini juga yang membuat tombol
		// submit pengaju berlabel "revisi selesai".
		if err := tx.Model(&models.TransactionDisposalAsset{}).
			Where("transaction_id = ?", transaction.ID).
			Updates(map[string]interface{}{"needs_revision": false, "revision_notes": nil}).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		if err := tx.Model(&models.TransactionDisposalAsset{}).
			Where("transaction_id = ? AND status = ?", transaction.ID, models.DisposalAssetStatusPending).
			Updates(map[string]interface{}{"needs_revision": true, "revision_notes": notes}).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// approval request disposal diulang dari awal saat submit ulang
		if err := resetApprovalsForRevision(tx, transaction.TransactionNumber, TxDisposalFlow); err != nil {
			tx.Rollback()
			return nil, err
		}

		if err := tx.Model(&transaction).Updates(map[string]interface{}{
			"approval_request_number":   nil,
			"approval_agreement_number": nil,
		}).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		fromStage := transaction.CurrentStage
		if err := updateTransactionStage(tx, &transaction, models.StageDisposalDraft); err != nil {
			tx.Rollback()
			return nil, err
		}

		stageNotes := agreementRevisionNote(agreementNumber, notes)
		if err := recordStage(tx, transaction.ID, transaction.TransactionNumber,
			fromStage, models.StageDisposalDraft,
			models.ActionRevise, userID, nil, &stageNotes); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalAgreementDetail(agreementNumber)
}
