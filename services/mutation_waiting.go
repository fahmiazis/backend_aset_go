package services

import (
	"backend-go/models"

	"gorm.io/gorm"
)

// Pemetaan stage mutasi ke endpoint pengerjanya. Stage APPROVAL tidak terdaftar
// karena ditentukan dari giliran step approval.
var mutationWaiting = waitingConfig{
	txType:     TxMutationFlow,
	draftStage: models.StageMutationDraft,
	owners: map[string]stageOwner{
		// Penerimaan dikonfirmasi cabang TUJUAN, bukan cabang pengaju —
		// karena itu penilaiannya lewat mutation_to_branch_code.
		models.StageMutationReceiving: {
			routePath:           "/transactions/mutation/confirm-receiving",
			permissions:         []string{"confirm_receiving", "create_transaction"},
			byDestinationBranch: true,
		},
		models.StageMutationExecute: {
			routePath: "/transactions/mutation/execute", permissions: []string{"execute_mutation"},
		},
	},
}

func applyMutationWaitingFilter(query *gorm.DB, userID string) *gorm.DB {
	return applyWaitingFilter(query, userID, mutationWaiting)
}

// IsMutationWaitingForUser dipakai halaman detail mutasi.
func IsMutationWaitingForUser(userID, transactionNumber string) bool {
	transaction, err := getMutationTransaction(transactionNumber)
	if err != nil {
		return false
	}
	return isWaitingForUser(userID, transaction, mutationWaiting)
}
