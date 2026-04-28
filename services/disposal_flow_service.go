package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
)

const TxDisposalFlow = "disposal"

// ============================================================
// HELPERS
// ============================================================

func getDisposalTransaction(transactionNumber string) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxDisposalFlow).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("disposal transaction not found")
		}
		return nil, err
	}
	return &transaction, nil
}

// stagesForDisposalType — urutan stage berdasarkan tipe disposal
func stagesForDisposalType(disposalType string) []string {
	if disposalType == models.DisposalTypeSell {
		return []string{
			models.StageDisposalDraft,
			models.StageDisposalSubmitted,
			models.StageDisposalPurchasing,
			models.StageDisposalApprovalRequest,
			models.StageDisposalApprovalAgreement,
			models.StageDisposalExecute,
			models.StageDisposalFinance,
			models.StageDisposalTax,
			models.StageDisposalAssetDeletion,
			models.StageDisposalFinished,
		}
	}
	// DISPOSE
	return []string{
		models.StageDisposalDraft,
		models.StageDisposalSubmitted,
		models.StageDisposalApprovalRequest,
		models.StageDisposalApprovalAgreement,
		models.StageDisposalExecute,
		models.StageDisposalAssetDeletion,
		models.StageDisposalFinished,
	}
}

// nextStageForDisposal — ambil next stage sesuai disposal type
func nextStageForDisposal(disposalType, currentStage string) (string, error) {
	stages := stagesForDisposalType(disposalType)
	for i, s := range stages {
		if s == currentStage && i+1 < len(stages) {
			return stages[i+1], nil
		}
	}
	return "", fmt.Errorf("no next stage after %s for disposal type %s", currentStage, disposalType)
}

// ============================================================
// CREATE DRAFT DISPOSAL
// ============================================================

func CreateDisposalDraft(userID string, req dto.CreateDisposalDraftRequest) (*dto.DisposalDetailResponse, error) {
	transactionDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		return nil, errors.New("invalid transaction date format, use YYYY-MM-DD")
	}

	transactionNumber, err := GenerateTransactionNumber(userID, TxDisposalFlow)
	if err != nil {
		return nil, err
	}

	disposalType := req.DisposalType

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	transaction := models.Transaction{
		TransactionNumber: transactionNumber,
		TransactionType:   TxDisposalFlow,
		TransactionDate:   transactionDate,
		Status:            models.TransactionStatusDraft,
		CurrentStage:      models.StageDisposalDraft,
		Notes:             req.Notes,
		CreatedBy:         userID,
		DisposalType:      &disposalType,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalDetail(transactionNumber)
}

// ============================================================
// ADD ASSET KE DRAFT
// ============================================================

func AddAssetToDisposal(userID string, transactionNumber string, req dto.AddDisposalAssetRequest) (*dto.DisposalDetailResponse, error) {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageDisposalDraft {
		return nil, errors.New("can only add assets to DRAFT disposals")
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only modify your own disposal draft")
	}

	// Validasi asset exist dan statusnya ACTIVE
	var asset models.Asset
	if err := config.DB.Preload("Category").First(&asset, req.AssetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("asset not found: %d", req.AssetID)
		}
		return nil, err
	}

	if asset.AssetNumber != req.AssetNumber {
		return nil, errors.New("asset number mismatch")
	}

	if asset.AssetStatus != models.AssetStatusAvailable {
		return nil, fmt.Errorf("asset %s is not available for disposal (status: %s)", req.AssetNumber, asset.AssetStatus)
	}

	// Cek asset belum ada di draft ini
	var existingCount int64
	config.DB.Model(&models.TransactionDisposalAsset{}).
		Where("transaction_id = ? AND asset_id = ?", transaction.ID, req.AssetID).
		Count(&existingCount)
	if existingCount > 0 {
		return nil, fmt.Errorf("asset %s already added to this disposal", req.AssetNumber)
	}

	// Cek asset tidak sedang di disposal lain yang aktif
	var otherDisposalCount int64
	config.DB.Model(&models.TransactionDisposalAsset{}).
		Joins("JOIN transactions ON transactions.id = transaction_disposal_assets.transaction_id").
		Where("transaction_disposal_assets.asset_id = ? AND transactions.current_stage NOT IN ? AND transaction_disposal_assets.status = ?",
			req.AssetID,
			[]string{models.StageDisposalFinished, models.StageDisposalRejected},
			models.DisposalAssetStatusPending,
		).
		Count(&otherDisposalCount)
	if otherDisposalCount > 0 {
		return nil, fmt.Errorf("asset %s is already in another active disposal", req.AssetNumber)
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	disposalAsset := models.TransactionDisposalAsset{
		TransactionID:     transaction.ID,
		TransactionNumber: transactionNumber,
		AssetID:           asset.ID,
		AssetNumber:       asset.AssetNumber,
		DisposalType:      *transaction.DisposalType,
		DisposalReason:    req.DisposalReason,
		Notes:             req.Notes,
		Status:            models.DisposalAssetStatusPending,
	}

	if err := tx.Create(&disposalAsset).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Update asset status → IN_DISPOSAL
	if err := tx.Model(&asset).Update("asset_status", models.AssetStatusInDisposal).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalDetail(transactionNumber)
}

// ============================================================
// REMOVE ASSET DARI DRAFT
// ============================================================

func RemoveAssetFromDisposal(userID string, transactionNumber string, req dto.RemoveDisposalAssetRequest) (*dto.DisposalDetailResponse, error) {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageDisposalDraft {
		return nil, errors.New("can only remove assets from DRAFT disposals")
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only modify your own disposal draft")
	}

	var disposalAsset models.TransactionDisposalAsset
	if err := config.DB.
		Where("transaction_id = ? AND asset_id = ?", transaction.ID, req.AssetID).
		First(&disposalAsset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("asset not found in this disposal")
		}
		return nil, err
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Hapus attachments asset ini dulu
	tx.Where("transaction_disposal_asset_id = ?", disposalAsset.ID).
		Delete(&models.TransactionDisposalAttachment{})

	if err := tx.Delete(&disposalAsset).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Kembalikan asset status → ACTIVE
	if err := tx.Model(&models.Asset{}).
		Where("id = ?", req.AssetID).
		Update("asset_status", models.AssetStatusAvailable).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalDetail(transactionNumber)
}

// ============================================================
// SUBMIT
// DRAFT → SUBMITTED
// ============================================================

func SubmitDisposal(userID string, transactionNumber string, req dto.SubmitDisposalRequest) (*dto.DisposalDetailResponse, error) {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only submit your own disposals")
	}

	if transaction.CurrentStage != models.StageDisposalDraft {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageDisposalDraft)
	}

	// Pastikan ada assets
	var assetCount int64
	config.DB.Model(&models.TransactionDisposalAsset{}).
		Where("transaction_id = ? AND status = ?", transaction.ID, models.DisposalAssetStatusPending).
		Count(&assetCount)
	if assetCount == 0 {
		return nil, errors.New("cannot submit disposal with no assets")
	}

	nextStage, err := nextStageForDisposal(*transaction.DisposalType, transaction.CurrentStage)
	if err != nil {
		return nil, err
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, nextStage); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, nextStage,
		models.ActionSubmit, userID, nil, req.Notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalDetail(transactionNumber)
}

// ============================================================
// PURCHASING — set sale_value per asset (SELL only)
// SUBMITTED → APPROVAL_REQUEST (setelah purchasing confirm)
// ============================================================

func SetDisposalSaleValues(userID string, transactionNumber string, req dto.SetDisposalSaleValueRequest) (*dto.DisposalDetailResponse, error) {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.DisposalType == nil || *transaction.DisposalType != models.DisposalTypeSell {
		return nil, errors.New("sale value can only be set for SELL disposals")
	}

	if transaction.CurrentStage != models.StageDisposalPurchasing {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageDisposalPurchasing)
	}

	// Cek semua attachment purchasing sudah diupload
	allOK, err := checkAllDisposalAttachments(transactionNumber, transaction.ID, models.StageDisposalPurchasing)
	if err != nil {
		return nil, err
	}
	if !allOK {
		return nil, errors.New("not all required purchasing documents are approved for all assets")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update sale_value per asset
	for _, item := range req.Assets {
		if err := tx.Model(&models.TransactionDisposalAsset{}).
			Where("id = ? AND transaction_id = ?", item.DisposalAssetID, transaction.ID).
			Update("sale_value", item.SaleValue).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to set sale value for asset %d: %w", item.DisposalAssetID, err)
		}
	}

	fromStage := transaction.CurrentStage
	nextStage := models.StageDisposalApprovalRequest

	if err := updateTransactionStage(tx, transaction, nextStage); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, nextStage,
		models.ActionSubmit, userID, nil, req.Notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalDetail(transactionNumber)
}

// ============================================================
// INITIATE APPROVAL REQUEST
// SUBMITTED → APPROVAL_REQUEST  (DISPOSE)
// PURCHASING done → APPROVAL_REQUEST (SELL — sudah dihandle di SetDisposalSaleValues)
// ============================================================

func InitiateDisposalApprovalRequest(userID string, transactionNumber string, req dto.InitiateDisposalApprovalRequest) error {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return err
	}

	if transaction.CurrentStage != models.StageDisposalApprovalRequest {
		return fmt.Errorf("transaction is not in %s stage", models.StageDisposalApprovalRequest)
	}

	creatorHomebase, homebaseErr := GetUserActiveHomebase(transaction.CreatedBy)
	branchCode := "ALL"
	if homebaseErr == nil {
		branchCode = creatorHomebase.Branch.BranchCode
	}

	flow, err := GetApprovalFlowByCodeAndBranch(models.FlowDisposalApprovalRequest, branchCode)
	if err != nil {
		return fmt.Errorf("approval flow %s not found for branch %s or ALL", models.FlowDisposalApprovalRequest, branchCode)
	}

	if !flow.IsActive {
		return fmt.Errorf("approval flow %s is inactive", models.FlowDisposalApprovalRequest)
	}

	var metadataStr *string
	if req.Metadata != nil {
		b, err := json.Marshal(req.Metadata)
		if err == nil {
			s := string(b)
			metadataStr = &s
		}
	}

	approvalReq := dto.CreateTransactionApprovalRequest{
		FlowID:            flow.ID,
		TransactionNumber: transactionNumber,
		TransactionType:   TxDisposalFlow,
		Metadata:          metadataStr,
	}

	if err := InitiateTransactionApproval(approvalReq); err != nil {
		return err
	}

	// Simpan approval_request_number = transaction number dari approval flow
	config.DB.Model(&models.Transaction{}).
		Where("transaction_number = ?", transactionNumber).
		Update("approval_request_number", transactionNumber)

	return nil
}

// ============================================================
// AUTO-COMPLETE APPROVAL REQUEST
// Dipanggil setelah semua step di APPROVAL_REQUEST approved
// → pindah ke APPROVAL_AGREEMENT
// ============================================================

func autoCompleteDisposalApprovalRequest(userID, transactionNumber string) error {
	var total, approved int64
	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ? AND approval_flow_code = ?",
			transactionNumber, TxDisposalFlow, models.FlowDisposalApprovalRequest).
		Count(&total)

	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ? AND approval_flow_code = ? AND status = ?",
			transactionNumber, TxDisposalFlow, models.FlowDisposalApprovalRequest, "approved").
		Count(&approved)

	if total == 0 || approved < total {
		return nil
	}

	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return err
	}

	if transaction.CurrentStage != models.StageDisposalApprovalRequest {
		return nil
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	fromStage := transaction.CurrentStage
	nextStage := models.StageDisposalApprovalAgreement

	if err := updateTransactionStage(tx, transaction, nextStage); err != nil {
		tx.Rollback()
		return err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, nextStage,
		models.ActionApprove, userID, nil, nil); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// ============================================================
// INITIATE APPROVAL AGREEMENT
// APPROVAL_REQUEST (done) → APPROVAL_AGREEMENT
// ============================================================

func InitiateDisposalApprovalAgreement(userID string, transactionNumber string, req dto.InitiateDisposalApprovalRequest) error {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return err
	}

	if transaction.CurrentStage != models.StageDisposalApprovalAgreement {
		return fmt.Errorf("transaction is not in %s stage", models.StageDisposalApprovalAgreement)
	}

	creatorHomebase, homebaseErr := GetUserActiveHomebase(transaction.CreatedBy)
	branchCode := "ALL"
	if homebaseErr == nil {
		branchCode = creatorHomebase.Branch.BranchCode
	}

	flow, err := GetApprovalFlowByCodeAndBranch(models.FlowDisposalApprovalAgreement, branchCode)
	if err != nil {
		return fmt.Errorf("approval flow %s not found for branch %s or ALL", models.FlowDisposalApprovalAgreement, branchCode)
	}

	if !flow.IsActive {
		return fmt.Errorf("approval flow %s is inactive", models.FlowDisposalApprovalAgreement)
	}

	var metadataStr *string
	if req.Metadata != nil {
		b, err := json.Marshal(req.Metadata)
		if err == nil {
			s := string(b)
			metadataStr = &s
		}
	}

	approvalReq := dto.CreateTransactionApprovalRequest{
		FlowID:            flow.ID,
		TransactionNumber: transactionNumber,
		TransactionType:   TxDisposalFlow,
		Metadata:          metadataStr,
	}

	if err := InitiateTransactionApproval(approvalReq); err != nil {
		return err
	}

	// Simpan approval_agreement_number = transaction number dari approval flow
	config.DB.Model(&models.Transaction{}).
		Where("transaction_number = ?", transactionNumber).
		Update("approval_agreement_number", transactionNumber)

	return nil
}

// ============================================================
// AUTO-COMPLETE APPROVAL AGREEMENT
// → pindah ke EXECUTE
// ============================================================

func autoCompleteDisposalApprovalAgreement(userID, transactionNumber string) error {
	var total, approved int64
	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ? AND approval_flow_code = ?",
			transactionNumber, TxDisposalFlow, models.FlowDisposalApprovalAgreement).
		Count(&total)

	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ? AND approval_flow_code = ? AND status = ?",
			transactionNumber, TxDisposalFlow, models.FlowDisposalApprovalAgreement, "approved").
		Count(&approved)

	if total == 0 || approved < total {
		return nil
	}

	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return err
	}

	if transaction.CurrentStage != models.StageDisposalApprovalAgreement {
		return nil
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	fromStage := transaction.CurrentStage
	nextStage := models.StageDisposalExecute

	if err := updateTransactionStage(tx, transaction, nextStage); err != nil {
		tx.Rollback()
		return err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, nextStage,
		models.ActionApprove, userID, nil, nil); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// ============================================================
// EXECUTE
// APPROVAL_AGREEMENT (done) → EXECUTE
// Creator upload dok penghapusan / hasil jual
// Setelah upload + approve → pindah ke stage berikutnya
// ============================================================

func ExecuteDisposal(userID string, transactionNumber string, req dto.ExecuteDisposalRequest) (*dto.DisposalDetailResponse, error) {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("only the creator can execute disposal")
	}

	if transaction.CurrentStage != models.StageDisposalExecute {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageDisposalExecute)
	}

	// Cek semua attachment execute sudah approved
	allOK, err := checkAllDisposalAttachments(transactionNumber, transaction.ID, models.StageDisposalExecute)
	if err != nil {
		return nil, err
	}
	if !allOK {
		return nil, errors.New("not all required execute documents are approved for all assets")
	}

	nextStage, err := nextStageForDisposal(*transaction.DisposalType, transaction.CurrentStage)
	if err != nil {
		return nil, err
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, nextStage); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, nextStage,
		models.ActionExecute, userID, nil, req.Notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalDetail(transactionNumber)
}

// ============================================================
// FINANCE (SELL only)
// EXECUTE → FINANCE → TAX
// ============================================================

func ConfirmDisposalFinance(userID string, transactionNumber string, req dto.ConfirmDisposalFinanceRequest) (*dto.DisposalDetailResponse, error) {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.DisposalType == nil || *transaction.DisposalType != models.DisposalTypeSell {
		return nil, errors.New("finance stage is only for SELL disposals")
	}

	if transaction.CurrentStage != models.StageDisposalFinance {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageDisposalFinance)
	}

	// Cek semua attachment finance sudah approved
	allOK, err := checkAllDisposalAttachments(transactionNumber, transaction.ID, models.StageDisposalFinance)
	if err != nil {
		return nil, err
	}
	if !allOK {
		return nil, errors.New("not all required finance documents are approved for all assets")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	fromStage := transaction.CurrentStage
	nextStage := models.StageDisposalTax

	if err := updateTransactionStage(tx, transaction, nextStage); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, nextStage,
		models.ActionSubmit, userID, nil, req.Notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalDetail(transactionNumber)
}

// ============================================================
// TAX (SELL only)
// FINANCE → TAX → ASSET_DELETION
// ============================================================

func ConfirmDisposalTax(userID string, transactionNumber string, req dto.ConfirmDisposalTaxRequest) (*dto.DisposalDetailResponse, error) {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.DisposalType == nil || *transaction.DisposalType != models.DisposalTypeSell {
		return nil, errors.New("tax stage is only for SELL disposals")
	}

	if transaction.CurrentStage != models.StageDisposalTax {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageDisposalTax)
	}

	// Cek semua attachment tax sudah approved
	allOK, err := checkAllDisposalAttachments(transactionNumber, transaction.ID, models.StageDisposalTax)
	if err != nil {
		return nil, err
	}
	if !allOK {
		return nil, errors.New("not all required tax documents are approved for all assets")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	fromStage := transaction.CurrentStage
	nextStage := models.StageDisposalAssetDeletion

	if err := updateTransactionStage(tx, transaction, nextStage); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, nextStage,
		models.ActionSubmit, userID, nil, req.Notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalDetail(transactionNumber)
}

// ============================================================
// ASSET DELETION — tim asset
// EXECUTE (DISPOSE) / TAX (SELL) → ASSET_DELETION → FINISHED
// Asset status → DISPOSED, asset_value di-zero-kan
// Generate document_number per asset
// ============================================================

func ConfirmDisposalAssetDeletion(userID string, transactionNumber string, req dto.ConfirmDisposalAssetDeletionRequest) (*dto.DisposalDetailResponse, error) {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageDisposalAssetDeletion {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageDisposalAssetDeletion)
	}

	// Ambil semua disposal asset
	var disposalAssets []models.TransactionDisposalAsset
	if err := config.DB.
		Where("transaction_id = ? AND status = ?", transaction.ID, models.DisposalAssetStatusPending).
		Find(&disposalAssets).Error; err != nil {
		return nil, err
	}

	if len(disposalAssets) == 0 {
		return nil, errors.New("no pending assets found in this disposal")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, da := range disposalAssets {
		// Generate document number per asset
		docNumber, err := GenerateDocumentNumber(tx)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to generate document number: %w", err)
		}

		// Update disposal asset → document_number + status DELETED
		if err := tx.Model(&da).Updates(map[string]interface{}{
			"document_number": docNumber,
			"status":          models.DisposalAssetStatusDeleted,
		}).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// Asset status → DISPOSED, asset_value → 0
		if err := tx.Model(&models.Asset{}).
			Where("id = ?", da.AssetID).
			Updates(map[string]interface{}{
				"asset_status": models.AssetStatusDisposed,
				"asset_value":  0,
			}).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update asset status: %w", err)
		}
	}

	fromStage := transaction.CurrentStage
	nextStage := models.StageDisposalFinished

	if err := updateTransactionStage(tx, transaction, nextStage); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, nextStage,
		models.ActionExecute, userID, nil, req.Notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalDetail(transactionNumber)
}

// ============================================================
// REJECT
// ============================================================

func RejectDisposal(userID string, transactionNumber string, req dto.RejectDisposalRequest) (*dto.DisposalDetailResponse, error) {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage == models.StageDisposalDraft ||
		transaction.CurrentStage == models.StageDisposalFinished ||
		transaction.CurrentStage == models.StageDisposalRejected {
		return nil, fmt.Errorf("cannot reject transaction in %s stage", transaction.CurrentStage)
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Kembalikan semua asset status → ACTIVE
	var disposalAssets []models.TransactionDisposalAsset
	config.DB.Where("transaction_id = ? AND status = ?", transaction.ID, models.DisposalAssetStatusPending).
		Find(&disposalAssets)

	for _, da := range disposalAssets {
		if err := tx.Model(&models.Asset{}).
			Where("id = ?", da.AssetID).
			Update("asset_status", models.AssetStatusAvailable).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		if err := tx.Model(&da).Update("status", models.DisposalAssetStatusCancelled).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageDisposalRejected); err != nil {
		tx.Rollback()
		return nil, err
	}

	reason := req.Reason
	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageDisposalRejected,
		models.ActionReject, userID, nil, &reason); err != nil {
		tx.Rollback()
		return nil, err
	}

	MarkTransactionAsExpired(transactionNumber)

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalDetail(transactionNumber)
}

// ============================================================
// GET DISPOSAL DETAIL
// ============================================================

func GetDisposalDetail(transactionNumber string) (*dto.DisposalDetailResponse, error) {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	var disposalAssets []models.TransactionDisposalAsset
	config.DB.
		Preload("Asset.Category").
		Preload("Attachments.AttachmentConfig").
		Where("transaction_id = ?", transaction.ID).
		Find(&disposalAssets)

	var stages []models.TransactionStage
	config.DB.
		Where("transaction_id = ?", transaction.ID).
		Order("created_at ASC").
		Find(&stages)

	assetResponses := make([]dto.DisposalAssetResponse, len(disposalAssets))
	for i, da := range disposalAssets {
		assetResp := dto.DisposalAssetResponse{
			ID:                da.ID,
			TransactionID:     da.TransactionID,
			TransactionNumber: da.TransactionNumber,
			AssetID:           da.AssetID,
			AssetNumber:       da.AssetNumber,
			DisposalType:      da.DisposalType,
			DisposalReason:    da.DisposalReason,
			SaleValue:         da.SaleValue,
			DocumentNumber:    da.DocumentNumber,
			Notes:             da.Notes,
			Status:            da.Status,
			CreatedAt:         da.CreatedAt,
			UpdatedAt:         da.UpdatedAt,
		}

		if da.Asset != nil {
			assetResp.AssetName = &da.Asset.AssetName
			assetResp.CategoryID = da.Asset.CategoryID
			assetResp.BranchCode = da.Asset.BranchCode
			if da.Asset.Category != nil {
				assetResp.CategoryName = &da.Asset.Category.CategoryName
			}
		}

		attachments := make([]dto.DisposalAttachmentResponse, len(da.Attachments))
		for j, att := range da.Attachments {
			attachments[j] = mapDisposalAttachmentToResponse(att)
		}
		assetResp.Attachments = attachments
		assetResponses[i] = assetResp
	}

	return &dto.DisposalDetailResponse{
		Transaction: dto.DisposalTransactionResponse{
			ID:                      transaction.ID,
			TransactionNumber:       transaction.TransactionNumber,
			TransactionType:         transaction.TransactionType,
			TransactionDate:         transaction.TransactionDate,
			Status:                  transaction.Status,
			CurrentStage:            transaction.CurrentStage,
			DisposalType:            transaction.DisposalType,
			SaleValue:               transaction.SaleValue,
			ApprovalRequestNumber:   transaction.ApprovalRequestNumber,
			ApprovalAgreementNumber: transaction.ApprovalAgreementNumber,
			Notes:                   transaction.Notes,
			CreatedBy:               transaction.CreatedBy,
			CreatedAt:               transaction.CreatedAt,
			UpdatedAt:               transaction.UpdatedAt,
		},
		Assets: assetResponses,
		Stages: mapTransactionStagesToResponse(stages),
	}, nil
}

// ============================================================
// GET ALL DISPOSALS
// ============================================================

func GetAllDisposals(filter dto.DisposalListFilter) ([]dto.DisposalDetailResponse, int64, error) {
	query := config.DB.Model(&models.Transaction{}).
		Where("transaction_type = ?", TxDisposalFlow)

	if filter.DisposalType != nil {
		query = query.Where("disposal_type = ?", *filter.DisposalType)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.CurrentStage != nil {
		query = query.Where("current_stage = ?", *filter.CurrentStage)
	}
	if filter.CreatedBy != nil {
		query = query.Where("created_by = ?", *filter.CreatedBy)
	}
	if filter.StartDate != nil {
		query = query.Where("transaction_date >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("transaction_date <= ?", *filter.EndDate)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 10
	}

	var total int64
	query.Count(&total)

	var transactions []models.Transaction
	query.Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&transactions)

	responses := make([]dto.DisposalDetailResponse, 0, len(transactions))
	for _, t := range transactions {
		detail, err := GetDisposalDetail(t.TransactionNumber)
		if err == nil {
			responses = append(responses, *detail)
		}
	}

	return responses, total, nil
}

// ============================================================
// ATTACHMENT PER ASSET PER STAGE
// ============================================================

func UploadDisposalAttachment(
	userID string,
	transactionNumber string,
	disposalAssetID uint,
	configID uint,
	stage string,
	file multipart.File,
	fileHeader *multipart.FileHeader,
) (*dto.DisposalAttachmentResponse, error) {

	// Validasi disposal asset exist dan milik transaksi ini
	var disposalAsset models.TransactionDisposalAsset
	if err := config.DB.
		Where("id = ? AND transaction_number = ?", disposalAssetID, transactionNumber).
		First(&disposalAsset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("disposal asset not found")
		}
		return nil, err
	}

	// Validasi attachment config
	var attachmentConfig models.AttachmentConfig
	if err := config.DB.First(&attachmentConfig, configID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("attachment config not found")
		}
		return nil, err
	}

	// Cek apakah sudah ada attachment untuk config + stage ini di asset ini
	var existingCount int64
	config.DB.Model(&models.TransactionDisposalAttachment{}).
		Where("transaction_disposal_asset_id = ? AND attachment_config_id = ? AND stage = ? AND status IN ?",
			disposalAssetID, configID, stage,
			[]string{models.AttachmentStatusPending, models.AttachmentStatusApproved}).
		Count(&existingCount)
	if existingCount > 0 {
		return nil, errors.New("attachment already uploaded for this config on this asset at this stage")
	}

	// Buat direktori
	dirPath := filepath.Join(
		AttachmentStoragePath,
		"disposal",
		sanitizePathSegment(transactionNumber),
		disposalAsset.AssetNumber,
		stage,
	)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	fileName := fmt.Sprintf("%s_%s", timestamp, fileHeader.Filename)
	filePath := filepath.Join(dirPath, fileName)

	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	fileSize, err := io.Copy(dst, file)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	mimeType := detectMimeType(fileHeader.Filename)
	now := time.Now()

	attachment := models.TransactionDisposalAttachment{
		TransactionID:              disposalAsset.TransactionID,
		TransactionNumber:          transactionNumber,
		TransactionDisposalAssetID: disposalAssetID,
		AssetID:                    disposalAsset.AssetID,
		AssetNumber:                disposalAsset.AssetNumber,
		AttachmentConfigID:         configID,
		Stage:                      stage,
		FileName:                   fileHeader.Filename,
		FilePath:                   filePath,
		FileSize:                   &fileSize,
		MimeType:                   &mimeType,
		Status:                     models.AttachmentStatusPending,
		UploadedBy:                 userID,
		UploadedAt:                 now,
	}

	if err := config.DB.Create(&attachment).Error; err != nil {
		os.Remove(filePath)
		return nil, err
	}

	attachment.AttachmentConfig = &attachmentConfig
	response := mapDisposalAttachmentToResponse(attachment)
	return &response, nil
}

func ReviewDisposalAttachment(reviewerID string, attachmentID uint, req dto.ReviewDisposalAttachmentRequest) (*dto.DisposalAttachmentResponse, error) {
	if req.Status == models.AttachmentStatusRejected && (req.RejectionReason == nil || *req.RejectionReason == "") {
		return nil, errors.New("rejection_reason is required when rejecting")
	}

	var attachment models.TransactionDisposalAttachment
	if err := config.DB.
		Preload("AttachmentConfig").
		First(&attachment, attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("attachment not found")
		}
		return nil, err
	}

	if attachment.Status != models.AttachmentStatusPending {
		return nil, errors.New("only PENDING attachments can be reviewed")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":      req.Status,
		"reviewed_by": reviewerID,
		"reviewed_at": now,
	}
	if req.RejectionReason != nil {
		updates["rejection_reason"] = req.RejectionReason
	}

	if err := config.DB.Model(&attachment).Updates(updates).Error; err != nil {
		return nil, err
	}

	config.DB.Preload("AttachmentConfig").First(&attachment, attachmentID)
	response := mapDisposalAttachmentToResponse(attachment)
	return &response, nil
}

func GetDisposalAttachmentStatus(transactionNumber string, transactionID uint, stage string) (*dto.DisposalAllAttachmentStatus, error) {
	var disposalAssets []models.TransactionDisposalAsset
	config.DB.
		Where("transaction_id = ? AND status = ?", transactionID, models.DisposalAssetStatusPending).
		Find(&disposalAssets)

	allCanProceed := true
	assetStatuses := make([]dto.DisposalAttachmentStatusSummary, 0, len(disposalAssets))

	for _, da := range disposalAssets {
		// Get required configs untuk disposal di stage ini
		requiredConfigs, err := getRequiredConfigs(TxDisposalFlow, stage, "")
		if err != nil {
			return nil, err
		}

		var attachments []models.TransactionDisposalAttachment
		config.DB.Preload("AttachmentConfig").
			Where("transaction_disposal_asset_id = ? AND stage = ?", da.ID, stage).
			Find(&attachments)

		attachmentByConfig := make(map[uint]models.TransactionDisposalAttachment)
		for _, att := range attachments {
			attachmentByConfig[att.AttachmentConfigID] = att
		}

		totalRequired := len(requiredConfigs)
		totalApproved, totalPending, totalRejected := 0, 0, 0

		for _, cfg := range requiredConfigs {
			att, exists := attachmentByConfig[cfg.ID]
			if !exists {
				continue
			}
			switch att.Status {
			case models.AttachmentStatusApproved:
				totalApproved++
			case models.AttachmentStatusPending:
				totalPending++
			case models.AttachmentStatusRejected:
				totalRejected++
			}
		}

		canProceed := totalApproved == totalRequired && totalRejected == 0
		if !canProceed {
			allCanProceed = false
		}

		attResponses := make([]dto.DisposalAttachmentResponse, len(attachments))
		for i, att := range attachments {
			attResponses[i] = mapDisposalAttachmentToResponse(att)
		}

		assetStatuses = append(assetStatuses, dto.DisposalAttachmentStatusSummary{
			AssetID:       da.AssetID,
			AssetNumber:   da.AssetNumber,
			Stage:         stage,
			CanProceed:    canProceed,
			TotalRequired: totalRequired,
			TotalApproved: totalApproved,
			TotalPending:  totalPending,
			TotalRejected: totalRejected,
			Attachments:   attResponses,
		})
	}

	return &dto.DisposalAllAttachmentStatus{
		TransactionNumber: transactionNumber,
		Stage:             stage,
		AllCanProceed:     allCanProceed,
		Assets:            assetStatuses,
	}, nil
}

// ============================================================
// HELPERS INTERNAL
// ============================================================

func checkAllDisposalAttachments(transactionNumber string, transactionID uint, stage string) (bool, error) {
	status, err := GetDisposalAttachmentStatus(transactionNumber, transactionID, stage)
	if err != nil {
		return false, err
	}
	return status.AllCanProceed, nil
}

func mapDisposalAttachmentToResponse(att models.TransactionDisposalAttachment) dto.DisposalAttachmentResponse {
	resp := dto.DisposalAttachmentResponse{
		ID:                         att.ID,
		TransactionDisposalAssetID: att.TransactionDisposalAssetID,
		AssetID:                    att.AssetID,
		AssetNumber:                att.AssetNumber,
		AttachmentConfigID:         att.AttachmentConfigID,
		Stage:                      att.Stage,
		FileName:                   att.FileName,
		FilePath:                   att.FilePath,
		FileSize:                   att.FileSize,
		MimeType:                   att.MimeType,
		Status:                     att.Status,
		UploadedBy:                 att.UploadedBy,
		UploadedAt:                 att.UploadedAt,
		ReviewedBy:                 att.ReviewedBy,
		ReviewedAt:                 att.ReviewedAt,
		RejectionReason:            att.RejectionReason,
		CreatedAt:                  att.CreatedAt,
	}
	if att.AttachmentConfig != nil {
		resp.AttachmentType = &att.AttachmentConfig.AttachmentType
		resp.IsRequired = &att.AttachmentConfig.IsRequired
	}
	return resp
}
