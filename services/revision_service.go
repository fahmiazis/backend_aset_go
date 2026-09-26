package services

import (
	"backend-go/config"
	"backend-go/models"
	"errors"

	"gorm.io/gorm"
)

// ============================================================
// Revisi & pembatalan — bagian yang sama untuk semua flow.
//
// Revisi ≠ tolak: tolak bersifat terminal, sedangkan revisi mengembalikan
// transaksi ke DRAFT dengan isinya tetap utuh supaya pengaju bisa memperbaiki
// baris yang ditandai lalu submit ulang.
//
// Batal ≠ tolak juga: batal dilakukan PENGAJU atas pengajuannya sendiri,
// sedangkan tolak dilakukan orang yang sedang memegang stage itu.
// ============================================================

// currentPendingApprovalFor mengambil step approval yang sedang berjalan, yaitu
// baris pending dengan step_order terkecil. Approver di urutan belakang belum
// gilirannya walau barisnya sudah pending.
func currentPendingApprovalFor(transactionNumber, transactionType string) (*models.TransactionApproval, error) {
	var approvals []models.TransactionApproval
	if err := config.DB.
		Preload("ApprovalFlowStep").
		Where("transaction_number = ? AND transaction_type = ? AND status = ?",
			transactionNumber, transactionType, "pending").
		Find(&approvals).Error; err != nil {
		return nil, err
	}

	if len(approvals) == 0 {
		return nil, errors.New("no pending approval found for this transaction")
	}

	// step_order ada di tabel lain, jadi pengurutannya dilakukan setelah fetch
	// (sama seperti GetTransactionApprovalStatus)
	current := approvals[0]
	for _, item := range approvals[1:] {
		if item.ApprovalFlowStep == nil || current.ApprovalFlowStep == nil {
			continue
		}
		if item.ApprovalFlowStep.StepOrder < current.ApprovalFlowStep.StepOrder {
			current = item
		}
	}

	return &current, nil
}

// assertCurrentApprover memastikan user memang approver step yang sedang
// berjalan, dengan aturan yang sama dengan ApproveTransaction.
func assertCurrentApprover(userID, transactionNumber, transactionType string) error {
	pending, err := currentPendingApprovalFor(transactionNumber, transactionType)
	if err != nil {
		return err
	}

	authorized := false
	if pending.ApproverUserID != nil && *pending.ApproverUserID == userID {
		authorized = true
	}
	if !authorized && pending.ApproverRoleID != nil {
		authorized = userHasRole(userID, *pending.ApproverRoleID)
	}
	if !authorized {
		return errors.New("you are not the approver of the current step")
	}

	return validateApproverBranch(userID, transactionNumber, transactionType)
}

// resetApprovalsForRevision membuang baris approval dan menonaktifkan tanda
// tangan lama.
//
// Barisnya dihapus, bukan ditandai, karena InitiateTransactionApproval menolak
// dengan "approval already initiated" selama masih ada baris untuk flow itu —
// submit ulang tidak akan pernah membentuk approval baru.
//
// Tanda tangan sengaja TIDAK dihapus (jejak audit), hanya tidak lagi dianggap
// yang terbaru.
func resetApprovalsForRevision(tx *gorm.DB, transactionNumber, transactionType string) error {
	if err := tx.Where("transaction_number = ? AND transaction_type = ?",
		transactionNumber, transactionType).
		Delete(&models.TransactionApproval{}).Error; err != nil {
		return err
	}

	return tx.Model(&models.ApprovalSignature{}).
		Where("transaction_number = ? AND transaction_type = ?",
			transactionNumber, transactionType).
		Update("is_recent", false).Error
}

// markRowsForRevision menandai baris yang perlu diperbaiki dan membersihkan
// penanda lama, supaya sisa revisi sebelumnya tidak menumpuk.
//
// `model` adalah tabel baris transaksi (item procurement / aset mutasi /
// aset disposal), `rowIDs` baris yang ditandai approver.
func markRowsForRevision(
	tx *gorm.DB,
	model interface{},
	transactionID uint,
	rowIDs []uint,
	notes string,
) error {
	if err := tx.Model(model).
		Where("transaction_id = ?", transactionID).
		Updates(map[string]interface{}{
			"needs_revision": false,
			"revision_notes": nil,
		}).Error; err != nil {
		return err
	}

	if len(rowIDs) == 0 {
		return nil
	}

	return tx.Model(model).
		Where("transaction_id = ? AND id IN ?", transactionID, rowIDs).
		Updates(map[string]interface{}{
			"needs_revision": true,
			"revision_notes": notes,
		}).Error
}
