package services

import (
	"backend-go/config"
	"backend-go/models"
	"errors"
)

// ============================================================
// SCOPE CABANG UNTUK DETAIL TRANSAKSI
//
// Aturannya sama dengan halaman daftar (listBranchScope /
// applyMutationBranchScope), ditambah pembuat selalu boleh melihat
// transaksinya sendiri walau homebase-nya sudah pindah.
//
// Approver tidak perlu pengecualian: validateApproverBranch sudah
// mensyaratkan approver punya cabang pembuat di user_branchs, jadi mereka
// otomatis masuk scope.
// ============================================================

var ErrTransactionForbidden = errors.New("you do not have access to this transaction")

// CanViewTransaction — nil kalau boleh, ErrTransactionForbidden kalau tidak.
// Transaksi yang tidak ditemukan dikembalikan nil supaya endpoint detail
// tetap menjawab 404 seperti biasa.
func CanViewTransaction(viewer AssetViewer, transactionNumber, txType string) error {
	scope := listBranchScope(viewer)
	if scope.all {
		return nil
	}

	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, txType).
		First(&transaction).Error; err != nil {
		return nil
	}

	if transaction.CreatedBy == viewer.UserID {
		return nil
	}
	if containsString(scope.codes, transactionBranchCode(transaction.TransactionNumber)) {
		return nil
	}
	if len(scope.codes) == 0 {
		return ErrTransactionForbidden
	}

	if txType == TxMutation {
		if transaction.MutationToBranchCode != nil && containsString(scope.codes, *transaction.MutationToBranchCode) {
			return nil
		}
		// Cabang asal/tujuan per aset — satu sumber aturan dengan daftar
		var count int64
		applyMutationBranchScope(
			config.DB.Model(&models.Transaction{}).Where("transactions.id = ?", transaction.ID),
			scope,
		).Count(&count)
		if count > 0 {
			return nil
		}
	}

	return ErrTransactionForbidden
}
