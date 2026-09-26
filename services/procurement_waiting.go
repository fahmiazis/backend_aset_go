package services

import (
	"backend-go/models"

	"gorm.io/gorm"
)

// Pemetaan stage procurement ke endpoint pengerjanya. Stage APPROVAL tidak
// terdaftar karena ditentukan dari giliran step approval.
var procurementWaiting = waitingConfig{
	txType:     TxProcurement,
	draftStage: models.StageDraft,
	owners: map[string]stageOwner{
		models.StageAssetVerification: {
			routePath: "/transactions/procurement/verify", permissions: []string{"verify_asset"},
		},
		models.StageProcessBudget: {
			routePath: "/transactions/procurement/process-budget", permissions: []string{"process_budget"},
		},
		models.StageExecuteAsset: {
			routePath: "/transactions/procurement/execute", permissions: []string{"execute_asset"},
		},
		// Endpoint GR meloloskan salah satu dari dua permission ini
		models.StageGR: {
			routePath: "/transactions/procurement/gr", permissions: []string{"gr", "create_transaction"},
		},
	},
}

func applyProcurementWaitingFilter(query *gorm.DB, userID string) *gorm.DB {
	return applyWaitingFilter(query, userID, procurementWaiting)
}

// IsProcurementWaitingForUser dipakai halaman detail procurement.
func IsProcurementWaitingForUser(userID, transactionNumber string) bool {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return false
	}
	return isWaitingForUser(userID, transaction, procurementWaiting)
}
