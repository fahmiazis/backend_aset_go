package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ============================================================
// STAGE HELPERS
// ============================================================

// recordStage mencatat perpindahan stage ke transaction_stages
func recordStage(tx *gorm.DB, transactionID uint, transactionNumber, fromStage, toStage, action, actorID string, actorName *string, notes *string) error {
	var from *string
	if fromStage != "" {
		f := fromStage
		from = &f
	}

	stage := models.TransactionStage{
		TransactionID:     transactionID,
		TransactionNumber: transactionNumber,
		FromStage:         from,
		ToStage:           toStage,
		Action:            action,
		ActorID:           actorID,
		ActorName:         actorName,
		Notes:             notes,
	}

	return tx.Create(&stage).Error
}

// stageToStatus mapping stage ke status transaksi
var stageToStatus = map[string]string{
	models.StageDraft:          models.TransactionStatusDraft,
	models.StageVerifikasiAset: "PENDING",
	models.StageApproval:       "PENDING",
	models.StageProsesBudget:   "PROCESSING",
	models.StageEksekusiAset:   "PROCESSING",
	models.StageGR:             "PROCESSING",
	models.StageSelesai:        models.TransactionStatusApproved,
	models.StageRejected:       models.TransactionStatusRejected,
}

// updateTransactionStage update current_stage & status di tabel transactions
func updateTransactionStage(tx *gorm.DB, transaction *models.Transaction, toStage string) error {
	status, ok := stageToStatus[toStage]
	if !ok {
		status = models.TransactionStatusDraft // fallback
	}

	return tx.Model(transaction).Updates(map[string]interface{}{
		"current_stage": toStage,
		"status":        status,
	}).Error
}

// validateStageTransition memastikan perpindahan stage valid
func validateStageTransition(currentStage, targetStage string) error {
	validTransitions := map[string][]string{
		models.StageDraft:          {models.StageVerifikasiAset, models.StageRejected},
		models.StageVerifikasiAset: {models.StageApproval, models.StageRejected},
		models.StageApproval:       {models.StageProsesBudget, models.StageRejected},
		models.StageProsesBudget:   {models.StageEksekusiAset, models.StageRejected},
		models.StageEksekusiAset:   {models.StageGR, models.StageRejected},
		models.StageGR:             {models.StageSelesai},
	}

	allowed, ok := validTransitions[currentStage]
	if !ok {
		return fmt.Errorf("invalid current stage: %s", currentStage)
	}

	for _, s := range allowed {
		if s == targetStage {
			return nil
		}
	}

	return fmt.Errorf("cannot transition from %s to %s", currentStage, targetStage)
}

// getProcurementTransaction ambil transaction procurement by transaction_number
func getProcurementTransaction(transactionNumber string) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxProcurement).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("procurement transaction not found")
		}
		return nil, err
	}
	return &transaction, nil
}

// ============================================================
// GENERATE IO NUMBER
// Format: {branch_code}IO{nomor_urut_4digit}
// Contoh: BC000001IO0001
// ============================================================

func GenerateIONumber(tx *gorm.DB, branchCode string) (string, error) {
	var seq models.DocumentNumberSequence

	// Lock row untuk hindari race condition
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("sequence_type = ? AND reference_code = ?", models.SeqTypeIO, branchCode).
		First(&seq).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Buat baru jika belum ada
		seq = models.DocumentNumberSequence{
			SequenceType:  models.SeqTypeIO,
			ReferenceCode: branchCode,
			LastSequence:  0,
		}
		if err := tx.Create(&seq).Error; err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	// Increment
	seq.LastSequence++
	if err := tx.Model(&seq).Update("last_sequence", seq.LastSequence).Error; err != nil {
		return "", err
	}

	// Format: BC000001IO0001
	ioNumber := fmt.Sprintf("%sIO%04d", branchCode, seq.LastSequence)
	return ioNumber, nil
}

// ============================================================
// GENERATE ASSET NUMBER
// Format: {category_code}{ddmmyy}{nomor_urut_4digit}
// Contoh: VHCL0603260001
// ============================================================

func GenerateAssetNumber(tx *gorm.DB, categoryCode string) (string, error) {
	var seq models.DocumentNumberSequence

	// Lock row untuk hindari race condition
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("sequence_type = ? AND reference_code = ?", models.SeqTypeAsset, categoryCode).
		First(&seq).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		seq = models.DocumentNumberSequence{
			SequenceType:  models.SeqTypeAsset,
			ReferenceCode: categoryCode,
			LastSequence:  0,
		}
		if err := tx.Create(&seq).Error; err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	// Increment
	seq.LastSequence++
	if err := tx.Model(&seq).Update("last_sequence", seq.LastSequence).Error; err != nil {
		return "", err
	}

	// Format: VHCL0603260001
	dateStr := time.Now().Format("020106") // ddmmyy
	assetNumber := fmt.Sprintf("%s%s%04d", strings.ToUpper(categoryCode), dateStr, seq.LastSequence)
	return assetNumber, nil
}

// ============================================================
// GET PROCUREMENT DETAIL WITH STAGE
// Dipakai oleh semua stage response
// ============================================================

func GetProcurementDetailWithStage(transactionNumber string) (*dto.ProcurementDetailWithStageResponse, error) {
	// Get transaction
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	// Get procurement items
	var procurements []models.TransactionProcurement
	if err := config.DB.
		Preload("Category").
		Preload("TransactionProcurementDetails").
		Where("transaction_id = ?", transaction.ID).
		Find(&procurements).Error; err != nil {
		return nil, err
	}

	// Get verifications
	var verifications []models.TransactionItemVerification
	config.DB.Where("transaction_id = ?", transaction.ID).Find(&verifications)
	verifMap := make(map[uint]models.TransactionItemVerification)
	for _, v := range verifications {
		verifMap[v.TransactionProcurementID] = v
	}

	// Get stage history
	var stages []models.TransactionStage
	config.DB.
		Where("transaction_id = ?", transaction.ID).
		Order("created_at ASC").
		Find(&stages)

	// Get GR status
	var grs []models.AssetGR
	config.DB.Where("transaction_id = ?", transaction.ID).Find(&grs)
	grMap := make(map[uint]models.AssetGR)
	for _, gr := range grs {
		grMap[gr.AssetID] = gr
	}

	// Get assets yang sudah digenerate (jika sudah EKSEKUSI_ASET)
	var assetAcquisitions []models.AssetAcquisition
	config.DB.
		Preload("Asset").
		Where("transaction_id = ?", transaction.ID).
		Find(&assetAcquisitions)
	acquisitionMap := make(map[uint]*models.Asset)
	for i := range assetAcquisitions {
		if assetAcquisitions[i].Asset != nil && assetAcquisitions[i].AssetID != nil {
			acquisitionMap[*assetAcquisitions[i].AssetID] = assetAcquisitions[i].Asset
		}
	}

	// Build items response
	items := make([]dto.ProcurementItemWithVerificationResponse, len(procurements))
	for i, p := range procurements {
		item := dto.ProcurementItemWithVerificationResponse{
			ProcurementItemResponse: mapProcurementItemToResponse(p),
		}

		// Attach verification
		if v, ok := verifMap[p.ID]; ok {
			item.Verification = &dto.TransactionItemVerificationResponse{
				ID:                       v.ID,
				TransactionID:            v.TransactionID,
				TransactionProcurementID: v.TransactionProcurementID,
				ItemName:                 &p.ItemName,
				ItemType:                 v.ItemType,
				IsActive:                 v.IsActive,
				VerifiedBy:               v.VerifiedBy,
				VerifiedAt:               v.VerifiedAt,
				Notes:                    v.Notes,
			}
		}

		items[i] = item
	}

	// Build GR response
	grResponses := make([]dto.AssetGRResponse, len(grs))
	for i, gr := range grs {
		grResponses[i] = mapAssetGRToResponse(gr)
	}

	return &dto.ProcurementDetailWithStageResponse{
		Transaction: dto.ProcurementTransactionResponse{
			ID:                transaction.ID,
			TransactionNumber: transaction.TransactionNumber,
			TransactionType:   transaction.TransactionType,
			TransactionDate:   transaction.TransactionDate,
			Status:            transaction.Status,
			CurrentStage:      transaction.CurrentStage,
			IONumber:          transaction.IONumber,
			Notes:             transaction.Notes,
			CreatedBy:         transaction.CreatedBy,
			ApprovedBy:        transaction.ApprovedBy,
			ApprovedAt:        transaction.ApprovedAt,
			CreatedAt:         transaction.CreatedAt,
			UpdatedAt:         transaction.UpdatedAt,
		},
		Items:    items,
		Stages:   mapTransactionStagesToResponse(stages),
		GRStatus: grResponses,
	}, nil
}

// ============================================================
// MAPPERS
// ============================================================

func mapTransactionStageToResponse(s models.TransactionStage) dto.TransactionStageResponse {
	return dto.TransactionStageResponse{
		ID:                s.ID,
		TransactionID:     s.TransactionID,
		TransactionNumber: s.TransactionNumber,
		FromStage:         s.FromStage,
		ToStage:           s.ToStage,
		Action:            s.Action,
		ActorID:           s.ActorID,
		ActorName:         s.ActorName,
		Notes:             s.Notes,
		CreatedAt:         s.CreatedAt,
	}
}

func mapTransactionStagesToResponse(stages []models.TransactionStage) []dto.TransactionStageResponse {
	result := make([]dto.TransactionStageResponse, len(stages))
	for i, s := range stages {
		result[i] = mapTransactionStageToResponse(s)
	}
	return result
}

func mapAssetGRToResponse(gr models.AssetGR) dto.AssetGRResponse {
	return dto.AssetGRResponse{
		ID:                gr.ID,
		TransactionID:     gr.TransactionID,
		TransactionNumber: gr.TransactionNumber,
		AssetID:           gr.AssetID,
		AssetNumber:       gr.AssetNumber,
		BranchCode:        gr.BranchCode,
		GRDate:            gr.GRDate,
		GRBy:              gr.GRBy,
		GRAt:              gr.GRAt,
		Notes:             gr.Notes,
		CreatedAt:         gr.CreatedAt,
	}
}
