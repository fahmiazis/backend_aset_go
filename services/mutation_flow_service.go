package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
)

const TxMutationFlow = "mutation"

// ============================================================
// HELPERS
// ============================================================

func getMutationTransaction(transactionNumber string) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxMutationFlow).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("mutation transaction not found")
		}
		return nil, err
	}
	return &transaction, nil
}

// ============================================================
// CREATE DRAFT MUTASI
// ============================================================

func CreateMutationDraft(userID string, req dto.CreateMutationDraftRequest) (*dto.MutationDetailResponse, error) {
	transactionDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		return nil, errors.New("invalid transaction date format, use YYYY-MM-DD")
	}

	// Validasi category exist
	var category models.AssetCategory
	if err := config.DB.First(&category, req.CategoryID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("category not found: %d", req.CategoryID)
		}
		return nil, err
	}

	// Validasi branch tujuan exist
	if err := validateBranchExists(req.ToBranchCode); err != nil {
		return nil, err
	}

	// Validasi branch tujuan != homebase user (tidak boleh mutasi ke branch sendiri)
	homebase, err := GetUserActiveHomebase(userID)
	if err != nil {
		return nil, err
	}
	if homebase.Branch.BranchCode == req.ToBranchCode {
		return nil, errors.New("cannot mutate assets to your own branch")
	}

	// Generate transaction number
	transactionNumber, err := GenerateTransactionNumber(userID, TxMutationFlow)
	if err != nil {
		return nil, err
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	transaction := models.Transaction{
		TransactionNumber:    transactionNumber,
		TransactionType:      TxMutationFlow,
		TransactionDate:      transactionDate,
		Status:               models.TransactionStatusDraft,
		CurrentStage:         models.StageDraft,
		Notes:                req.Notes,
		CreatedBy:            userID,
		MutationCategoryID:   &req.CategoryID,
		MutationToBranchCode: &req.ToBranchCode,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetMutationDetail(transactionNumber)
}

// ============================================================
// ADD ASSET KE DRAFT
// ============================================================

func AddAssetToMutation(userID string, transactionNumber string, req dto.AddMutationAssetRequest) (*dto.MutationDetailResponse, error) {
	transaction, err := getMutationTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageDraft {
		return nil, errors.New("can only add assets to DRAFT mutations")
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only modify your own mutation draft")
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
		return nil, fmt.Errorf("asset %s is not available for mutation (status: %s)", req.AssetNumber, asset.AssetStatus)
	}

	// Validasi category harus sama dengan draft
	if asset.CategoryID == nil || *asset.CategoryID != *transaction.MutationCategoryID {
		return nil, fmt.Errorf("asset %s category does not match mutation category", req.AssetNumber)
	}

	// Validasi branch asal harus sama dengan homebase user
	if asset.BranchCode == nil || *asset.BranchCode != *transaction.MutationToBranchCode {
		// branch asal asset harus bukan branch tujuan
	}
	if asset.BranchCode == nil {
		return nil, fmt.Errorf("asset %s has no branch assigned", req.AssetNumber)
	}

	// Cek asset belum ada di draft ini
	var existingCount int64
	config.DB.Model(&models.TransactionMutationAsset{}).
		Where("transaction_id = ? AND asset_id = ?", transaction.ID, req.AssetID).
		Count(&existingCount)
	if existingCount > 0 {
		return nil, fmt.Errorf("asset %s already added to this mutation", req.AssetNumber)
	}

	// Cek asset tidak sedang di draft mutasi lain
	var otherMutationCount int64
	config.DB.Model(&models.TransactionMutationAsset{}).
		Joins("JOIN transactions ON transactions.id = transaction_mutation_assets.transaction_id").
		Where("transaction_mutation_assets.asset_id = ? AND transactions.current_stage NOT IN ? AND transaction_mutation_assets.status = ?",
			req.AssetID,
			[]string{models.StageMutationFinished, models.StageMutationRejected},
			models.MutationAssetStatusPending,
		).
		Count(&otherMutationCount)
	if otherMutationCount > 0 {
		return nil, fmt.Errorf("asset %s is already in another active mutation", req.AssetNumber)
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Tambah asset ke draft
	mutationAsset := models.TransactionMutationAsset{
		TransactionID:     transaction.ID,
		TransactionNumber: transactionNumber,
		AssetID:           asset.ID,
		AssetNumber:       asset.AssetNumber,
		FromBranchCode:    *asset.BranchCode,
		ToBranchCode:      *transaction.MutationToBranchCode,
		FromLocation:      req.FromLocation,
		ToLocation:        req.ToLocation,
		Notes:             req.Notes,
		Status:            models.MutationAssetStatusPending,
	}

	if err := tx.Create(&mutationAsset).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Update asset status → IN_MUTATION
	if err := tx.Model(&asset).Update("asset_status", models.AssetStatusInMutation).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetMutationDetail(transactionNumber)
}

// ============================================================
// REMOVE ASSET DARI DRAFT
// ============================================================

func RemoveAssetFromMutation(userID string, transactionNumber string, req dto.RemoveMutationAssetRequest) (*dto.MutationDetailResponse, error) {
	transaction, err := getMutationTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageDraft {
		return nil, errors.New("can only remove assets from DRAFT mutations")
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only modify your own mutation draft")
	}

	// Cari mutation asset record
	var mutationAsset models.TransactionMutationAsset
	if err := config.DB.
		Where("transaction_id = ? AND asset_id = ?", transaction.ID, req.AssetID).
		First(&mutationAsset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("asset not found in this mutation")
		}
		return nil, err
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Hapus attachment asset ini dulu
	tx.Where("transaction_mutation_asset_id = ?", mutationAsset.ID).
		Delete(&models.TransactionMutationAttachment{})

	// Hapus mutation asset record
	if err := tx.Delete(&mutationAsset).Error; err != nil {
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

	return GetMutationDetail(transactionNumber)
}

// ============================================================
// SUBMIT
// DRAFT → APPROVAL
// ============================================================

func SubmitMutation(userID string, transactionNumber string, req dto.SubmitMutationRequest) (*dto.MutationDetailResponse, error) {
	transaction, err := getMutationTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only submit your own mutations")
	}

	if transaction.CurrentStage != models.StageDraft {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageDraft)
	}

	// Pastikan ada assets
	var assetCount int64
	config.DB.Model(&models.TransactionMutationAsset{}).
		Where("transaction_id = ? AND status = ?", transaction.ID, models.MutationAssetStatusPending).
		Count(&assetCount)
	if assetCount == 0 {
		return nil, errors.New("cannot submit mutation with no assets")
	}

	// Cek attachment semua asset sebelum submit
	allAttachmentOK, err := checkAllMutationAttachments(transactionNumber, transaction.ID, models.StageDraft)
	if err != nil {
		return nil, err
	}
	if !allAttachmentOK {
		return nil, errors.New("not all required attachments are approved for all assets")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageApproval); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageApproval,
		models.ActionSubmit, userID, nil, req.Notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetMutationDetail(transactionNumber)
}

// ============================================================
// INITIATE APPROVAL
// ============================================================

func InitiateMutationApproval(userID string, transactionNumber string, req dto.InitiateApprovalRequest) error {
	transaction, err := getMutationTransaction(transactionNumber)
	if err != nil {
		return err
	}

	if transaction.CurrentStage != models.StageApproval {
		return fmt.Errorf("transaction is not in %s stage", models.StageApproval)
	}

	// Auto-lookup MUTATION_APPROVAL flow by branch creator
	creatorHomebase, homebaseErr := GetUserActiveHomebase(transaction.CreatedBy)
	branchCode := "ALL"
	if homebaseErr == nil {
		branchCode = creatorHomebase.Branch.BranchCode
	}

	flow, err := GetApprovalFlowByCodeAndBranch("MUTATION_APPROVAL", branchCode)
	if err != nil {
		return fmt.Errorf("approval flow MUTATION_APPROVAL not found for branch %s or ALL", branchCode)
	}

	if !flow.IsActive {
		return errors.New("approval flow MUTATION_APPROVAL is inactive")
	}

	approvalReq := dto.CreateTransactionApprovalRequest{
		FlowID:            flow.ID,
		TransactionNumber: transactionNumber,
		TransactionType:   TxMutationFlow,
		Metadata:          req.Metadata,
	}

	return InitiateTransactionApproval(approvalReq)
}

// ============================================================
// EKSEKUSI MUTASI
// APPROVAL → FINISHED
// Update branch_code asset + generate document number
// ============================================================

func ExecuteMutation(userID string, transactionNumber string, req dto.ExecuteMutationRequest) (*dto.MutationDetailResponse, error) {
	transaction, err := getMutationTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	// Hanya bisa dieksekusi kalau sudah di stage EXECUTE_MUTATION
	// (otomatis masuk sini setelah semua approval step approved)
	if transaction.CurrentStage != models.StageMutationExecute {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageMutationExecute)
	}

	// Ambil semua asset di mutasi ini
	var mutationAssets []models.TransactionMutationAsset
	if err := config.DB.
		Where("transaction_id = ? AND status = ?", transaction.ID, models.MutationAssetStatusPending).
		Find(&mutationAssets).Error; err != nil {
		return nil, err
	}

	if len(mutationAssets) == 0 {
		return nil, errors.New("no pending assets found in this mutation")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, ma := range mutationAssets {
		// Generate document number per asset
		docNumber, err := GenerateDocumentNumber(tx)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to generate document number: %w", err)
		}

		// Update mutation asset — document number + status EXECUTED
		if err := tx.Model(&ma).Updates(map[string]interface{}{
			"document_number": docNumber,
			"status":          models.MutationAssetStatusExecuted,
		}).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// Update asset branch_code → branch tujuan
		if err := tx.Model(&models.Asset{}).
			Where("id = ?", ma.AssetID).
			Updates(map[string]interface{}{
				"branch_code":  ma.ToBranchCode,
				"asset_status": models.AssetStatusAvailable, // kembali ACTIVE di branch baru
			}).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update asset branch: %w", err)
		}
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageFinished); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageFinished,
		models.ActionExecute, userID, nil, req.Notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetMutationDetail(transactionNumber)
}

// ============================================================
// REJECT
// ============================================================

func RejectMutation(userID string, transactionNumber string, req dto.RejectMutationRequest) (*dto.MutationDetailResponse, error) {
	transaction, err := getMutationTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage == models.StageDraft ||
		transaction.CurrentStage == models.StageFinished ||
		transaction.CurrentStage == models.StageRejected {
		return nil, fmt.Errorf("cannot reject transaction in %s stage", transaction.CurrentStage)
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Kembalikan semua asset status → ACTIVE
	var mutationAssets []models.TransactionMutationAsset
	config.DB.Where("transaction_id = ? AND status = ?", transaction.ID, models.MutationAssetStatusPending).
		Find(&mutationAssets)

	for _, ma := range mutationAssets {
		if err := tx.Model(&models.Asset{}).
			Where("id = ?", ma.AssetID).
			Update("asset_status", models.AssetStatusAvailable).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// Update mutation asset status → CANCELLED
		if err := tx.Model(&ma).Update("status", models.MutationAssetStatusCancelled).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageRejected); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageRejected,
		models.ActionReject, userID, nil, &req.Reason); err != nil {
		tx.Rollback()
		return nil, err
	}

	MarkTransactionAsExpired(transactionNumber)

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetMutationDetail(transactionNumber)
}

// ============================================================
// GET MUTATION DETAIL
// ============================================================

func GetMutationDetail(transactionNumber string) (*dto.MutationDetailResponse, error) {
	transaction, err := getMutationTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	// Get mutation assets
	var mutationAssets []models.TransactionMutationAsset
	config.DB.
		Preload("Asset.Category").
		Preload("Attachments.AttachmentConfig").
		Where("transaction_id = ?", transaction.ID).
		Find(&mutationAssets)

	// Get stages
	var stages []models.TransactionStage
	config.DB.
		Where("transaction_id = ?", transaction.ID).
		Order("created_at ASC").
		Find(&stages)

	// Build response
	assetResponses := make([]dto.MutationAssetResponse, len(mutationAssets))
	for i, ma := range mutationAssets {
		assetResp := dto.MutationAssetResponse{
			ID:                ma.ID,
			TransactionID:     ma.TransactionID,
			TransactionNumber: ma.TransactionNumber,
			AssetID:           ma.AssetID,
			AssetNumber:       ma.AssetNumber,
			FromBranchCode:    ma.FromBranchCode,
			ToBranchCode:      ma.ToBranchCode,
			FromLocation:      ma.FromLocation,
			ToLocation:        ma.ToLocation,
			DocumentNumber:    ma.DocumentNumber,
			Notes:             ma.Notes,
			Status:            ma.Status,
			CreatedAt:         ma.CreatedAt,
			UpdatedAt:         ma.UpdatedAt,
		}

		if ma.Asset != nil {
			assetResp.AssetName = &ma.Asset.AssetName
			assetResp.CategoryID = ma.Asset.CategoryID
			if ma.Asset.Category != nil {
				assetResp.CategoryName = &ma.Asset.Category.CategoryName
			}
		}

		// Map attachments
		attachments := make([]dto.MutationAttachmentResponse, len(ma.Attachments))
		for j, att := range ma.Attachments {
			attResp := dto.MutationAttachmentResponse{
				ID:                         att.ID,
				TransactionMutationAssetID: att.TransactionMutationAssetID,
				AssetID:                    att.AssetID,
				AssetNumber:                att.AssetNumber,
				AttachmentConfigID:         att.AttachmentConfigID,
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
				attResp.AttachmentType = &att.AttachmentConfig.AttachmentType
				attResp.IsRequired = &att.AttachmentConfig.IsRequired
			}
			attachments[j] = attResp
		}
		assetResp.Attachments = attachments

		assetResponses[i] = assetResp
	}

	// Build transaction response
	var categoryName *string
	if transaction.MutationCategoryID != nil {
		var cat models.AssetCategory
		if err := config.DB.First(&cat, transaction.MutationCategoryID).Error; err == nil {
			categoryName = &cat.CategoryName
		}
	}

	return &dto.MutationDetailResponse{
		Transaction: dto.MutationTransactionResponse{
			ID:                transaction.ID,
			TransactionNumber: transaction.TransactionNumber,
			TransactionType:   transaction.TransactionType,
			TransactionDate:   transaction.TransactionDate,
			Status:            transaction.Status,
			CurrentStage:      transaction.CurrentStage,
			CategoryID:        transaction.MutationCategoryID,
			CategoryName:      categoryName,
			ToBranchCode:      transaction.MutationToBranchCode,
			Notes:             transaction.Notes,
			CreatedBy:         transaction.CreatedBy,
			CreatedAt:         transaction.CreatedAt,
			UpdatedAt:         transaction.UpdatedAt,
		},
		Assets: assetResponses,
		Stages: mapTransactionStagesToResponse(stages),
	}, nil
}

func GetAllMutationDrafts(userID string, page, limit int) ([]dto.MutationDetailResponse, int64, error) {
	query := config.DB.Model(&models.Transaction{}).
		Where("transaction_type = ? AND created_by = ?", TxMutationFlow, userID)

	var total int64
	query.Count(&total)

	var transactions []models.Transaction
	query.Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&transactions)

	responses := make([]dto.MutationDetailResponse, 0, len(transactions))
	for _, t := range transactions {
		detail, err := GetMutationDetail(t.TransactionNumber)
		if err == nil {
			responses = append(responses, *detail)
		}
	}

	return responses, total, nil
}

// ============================================================
// ATTACHMENT PER ASSET
// ============================================================

func UploadMutationAttachment(
	userID string,
	transactionNumber string,
	mutationAssetID uint,
	configID uint,
	file multipart.File,
	fileHeader *multipart.FileHeader,
) (*dto.MutationAttachmentResponse, error) {

	// Validasi mutation asset exist dan milik transaksi ini
	var mutationAsset models.TransactionMutationAsset
	if err := config.DB.
		Where("id = ? AND transaction_number = ?", mutationAssetID, transactionNumber).
		First(&mutationAsset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("mutation asset not found")
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

	// Cek apakah sudah ada attachment untuk config ini di asset ini
	var existingCount int64
	config.DB.Model(&models.TransactionMutationAttachment{}).
		Where("transaction_mutation_asset_id = ? AND attachment_config_id = ? AND status IN ?",
			mutationAssetID, configID,
			[]string{models.AttachmentStatusPending, models.AttachmentStatusApproved}).
		Count(&existingCount)
	if existingCount > 0 {
		return nil, errors.New("attachment already uploaded for this config on this asset")
	}

	// Buat direktori
	dirPath := filepath.Join(
		AttachmentStoragePath,
		"mutation",
		sanitizePathSegment(transactionNumber),
		mutationAsset.AssetNumber,
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

	attachment := models.TransactionMutationAttachment{
		TransactionID:              mutationAsset.TransactionID,
		TransactionNumber:          transactionNumber,
		TransactionMutationAssetID: mutationAssetID,
		AssetID:                    mutationAsset.AssetID,
		AssetNumber:                mutationAsset.AssetNumber,
		AttachmentConfigID:         configID,
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
	response := mapMutationAttachmentToResponse(attachment)
	return &response, nil
}

func ReviewMutationAttachment(reviewerID string, attachmentID uint, req dto.ReviewMutationAttachmentRequest) (*dto.MutationAttachmentResponse, error) {
	if req.Status == models.AttachmentStatusRejected && (req.RejectionReason == nil || *req.RejectionReason == "") {
		return nil, errors.New("rejection_reason is required when rejecting")
	}

	var attachment models.TransactionMutationAttachment
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
	response := mapMutationAttachmentToResponse(attachment)
	return &response, nil
}

func GetMutationAttachmentStatus(transactionNumber string, transactionID uint) (*dto.MutationAllAttachmentStatus, error) {
	var mutationAssets []models.TransactionMutationAsset
	config.DB.
		Where("transaction_id = ? AND status = ?", transactionID, models.MutationAssetStatusPending).
		Find(&mutationAssets)

	allCanProceed := true
	assetStatuses := make([]dto.MutationAttachmentStatusSummary, 0, len(mutationAssets))

	for _, ma := range mutationAssets {
		// Get required configs untuk mutation
		requiredConfigs, err := getRequiredConfigs(TxMutationFlow, models.StageDraft, ma.FromBranchCode)
		if err != nil {
			return nil, err
		}

		// Get attachments untuk asset ini
		var attachments []models.TransactionMutationAttachment
		config.DB.Preload("AttachmentConfig").
			Where("transaction_mutation_asset_id = ?", ma.ID).
			Find(&attachments)

		attachmentByConfig := make(map[uint]models.TransactionMutationAttachment)
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

		attResponses := make([]dto.MutationAttachmentResponse, len(attachments))
		for i, att := range attachments {
			attResponses[i] = mapMutationAttachmentToResponse(att)
		}

		assetStatuses = append(assetStatuses, dto.MutationAttachmentStatusSummary{
			TransactionNumber: transactionNumber,
			AssetID:           ma.AssetID,
			AssetNumber:       ma.AssetNumber,
			CanProceed:        canProceed,
			TotalRequired:     totalRequired,
			TotalApproved:     totalApproved,
			TotalPending:      totalPending,
			TotalRejected:     totalRejected,
			Attachments:       attResponses,
		})
	}

	return &dto.MutationAllAttachmentStatus{
		TransactionNumber: transactionNumber,
		AllCanProceed:     allCanProceed,
		Assets:            assetStatuses,
	}, nil
}

// ============================================================
// HELPERS
// ============================================================

func checkAllMutationAttachments(transactionNumber string, transactionID uint, stage string) (bool, error) {
	status, err := GetMutationAttachmentStatus(transactionNumber, transactionID)
	if err != nil {
		return false, err
	}

	isDraft := stage == models.StageDraft

	if isDraft {
		// Di DRAFT: cukup semua required sudah diupload (PENDING atau APPROVED)
		// REJECTED tidak boleh
		for _, asset := range status.Assets {
			if asset.TotalRejected > 0 {
				return false, nil
			}
			// Cek missing = total required - (approved + pending)
			uploaded := asset.TotalApproved + asset.TotalPending
			if uploaded < asset.TotalRequired {
				return false, nil
			}
		}
		return true, nil
	}

	// Stage lain: semua wajib APPROVED
	return status.AllCanProceed, nil
}

func mapMutationAttachmentToResponse(att models.TransactionMutationAttachment) dto.MutationAttachmentResponse {
	resp := dto.MutationAttachmentResponse{
		ID:                         att.ID,
		TransactionMutationAssetID: att.TransactionMutationAssetID,
		AssetID:                    att.AssetID,
		AssetNumber:                att.AssetNumber,
		AttachmentConfigID:         att.AttachmentConfigID,
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

// ============================================================
// MUTATION RECEIVING
// APPROVAL → MUTATION_RECEIVING → EXECUTE_MUTATION
// Dilakukan oleh user homebase branch tujuan
// Upload dokumen serah terima per asset
// ============================================================

func ConfirmMutationReceiving(userID string, transactionNumber string, notes *string) (*dto.MutationDetailResponse, error) {
	transaction, err := getMutationTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageMutationReceiving {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageMutationReceiving)
	}

	// Validasi user homebase harus di branch tujuan
	homebase, err := GetUserActiveHomebase(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user homebase: %w", err)
	}

	if transaction.MutationToBranchCode == nil || homebase.Branch.BranchCode != *transaction.MutationToBranchCode {
		return nil, fmt.Errorf("only users from branch %s can confirm receiving", *transaction.MutationToBranchCode)
	}

	// Cek semua attachment serah terima sudah diupload dan approved
	allAttachmentOK, err := checkAllMutationAttachments(transactionNumber, transaction.ID, models.StageMutationReceiving)
	if err != nil {
		return nil, err
	}
	if !allAttachmentOK {
		return nil, errors.New("not all required receiving documents are approved for all assets")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageMutationExecute); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageMutationExecute,
		models.ActionGR, userID, nil, notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetMutationDetail(transactionNumber)
}

// autoCompleteMutationApproval — dipanggil otomatis setelah approve step
// Kalau semua step approved → pindah stage ke EXECUTE_MUTATION
// Eksekusi dilakukan manual oleh PIC Asset yang berwenang
func autoCompleteMutationApproval(userID, transactionNumber, transactionType string) error {
	if transactionType != TxMutationFlow {
		return nil
	}

	var total, approved int64
	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, transactionType).
		Count(&total)

	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ? AND status = ?", transactionNumber, transactionType, "approved").
		Count(&approved)

	if total == 0 || approved < total {
		return nil
	}

	transaction, err := getMutationTransaction(transactionNumber)
	if err != nil {
		return err
	}

	if transaction.CurrentStage != models.StageApproval {
		return nil
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	fromStage := transaction.CurrentStage
	// Setelah approval → MUTATION_RECEIVING (branch tujuan upload dok serah terima)
	if err := updateTransactionStage(tx, transaction, models.StageMutationReceiving); err != nil {
		tx.Rollback()
		return err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageMutationReceiving,
		models.ActionApprove, userID, nil, nil); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
