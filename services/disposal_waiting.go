package services

import (
	"backend-go/models"

	"gorm.io/gorm"
)

// Pemetaan stage disposal ke endpoint pengerjanya. Stage yang tidak terdaftar
// di sini (APPROVAL_REQUEST) dianggap tahap approval dan ditentukan dari
// giliran step, bukan dari permission endpoint.
var disposalWaiting = waitingConfig{
	txType:     TxDisposalFlow,
	draftStage: models.StageDisposalDraft,
	owners: map[string]stageOwner{
		models.StageDisposalPurchasing: {
			routePath: "/transactions/disposal/purchasing", permissions: []string{"manage_purchasing"},
		},
		models.StageDisposalExecute: {
			routePath: "/transactions/disposal/execute", permissions: []string{"execute_disposal"},
		},
		models.StageDisposalFinance: {
			routePath: "/transactions/disposal/finance", permissions: []string{"manage_finance"},
		},
		models.StageDisposalTax: {
			routePath: "/transactions/disposal/tax", permissions: []string{"manage_tax"},
		},
		models.StageDisposalAssetDeletion: {
			routePath: "/transactions/disposal/asset-deletion", permissions: []string{"execute_asset_deletion"},
		},
		// Tahap kesepakatan dikerjakan lewat pengelompokan agreement, bukan
		// endpoint per transaksi. Begitu sudah masuk agreement aktif, bolanya
		// pindah ke approver agreement — stage transaksinya sendiri baru
		// berubah setelah agreement disetujui.
		models.StageDisposalApprovalAgreement: {
			routePath: "/transactions/disposal-agreements", permissions: []string{"manage_disposal_agreement"},
			excludeIDs: activeAgreementTransactionIDs,
		},
	},
}

func applyDisposalWaitingFilter(query *gorm.DB, userID string) *gorm.DB {
	return applyWaitingFilter(query, userID, disposalWaiting)
}

// IsDisposalWaitingForUser dipakai halaman detail disposal.
func IsDisposalWaitingForUser(userID, transactionNumber string) bool {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return false
	}
	return isWaitingForUser(userID, transaction, disposalWaiting)
}
