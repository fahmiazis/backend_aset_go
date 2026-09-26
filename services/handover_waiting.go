package services

import (
	"backend-go/config"
	"backend-go/models"

	"gorm.io/gorm"
)

// Pemetaan stage serah terima ke pengerjanya:
//   - APPROVAL tidak terdaftar → ditentukan giliran step approval
//   - HANDOVER_RECEIVING:
//     RETURN   → user ber-confirm_receiving di cabang pengaju (owner biasa)
//     HANDOVER → user PENERIMA itu sendiri (extraCondition/extraCheck)
var handoverWaiting = waitingConfig{
	txType:     TxHandover,
	draftStage: models.StageHandoverDraft,
	owners: map[string]stageOwner{
		models.StageHandoverReceiving: {
			routePath:   handoverConfirmRoute,
			permissions: []string{"confirm_receiving"},
			// serah terima ke user dikonfirmasi penerimanya, bukan pemegang
			// permission — hanya pengembalian yang lewat owner ini
			excludeIDs: handoverToUserTransactionIDs,
		},
	},
	extraCondition: func(userID string) *gorm.DB {
		return config.DB.
			Where("current_stage = ?", models.StageHandoverReceiving).
			Where("handover_type = ?", models.HandoverTypeHandover).
			Where("handover_to_user_id = ?", userID)
	},
	extraCheck: func(userID string, t *models.Transaction) bool {
		return t.CurrentStage == models.StageHandoverReceiving &&
			handoverTypeOf(t) == models.HandoverTypeHandover &&
			t.HandoverToUserID != nil && *t.HandoverToUserID == userID
	},
}

func handoverToUserTransactionIDs() *gorm.DB {
	return config.DB.Model(&models.Transaction{}).
		Select("id").
		Where("transaction_type = ? AND handover_type = ?", TxHandover, models.HandoverTypeHandover)
}
