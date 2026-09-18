package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TxStockOpnameFlow adalah transaction_type yang sama dipakai oleh CRUD lama
// (services/stock_opname_service.go) — flow baru ini menggantikannya, bukan
// menambah tipe baru.
const TxStockOpnameFlow = "stock_opname"

// ============================================================
// HELPERS
// ============================================================

func getStockOpnameTransaction(transactionNumber string) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxStockOpnameFlow).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("stock opname transaction not found")
		}
		return nil, err
	}
	return &transaction, nil
}

// ============================================================
// CREATE DRAFT
// ============================================================

func CreateStockOpnameDraft(userID string, req dto.CreateStockOpnameDraftRequest) (*dto.StockOpnameFlowDetailResponse, error) {
	transactionDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		return nil, errors.New("invalid transaction date format, use YYYY-MM-DD")
	}

	transactionNumber, err := GenerateTransactionNumber(userID, TxStockOpnameFlow)
	if err != nil {
		return nil, err
	}

	transaction := models.Transaction{
		TransactionNumber: transactionNumber,
		TransactionType:   TxStockOpnameFlow,
		TransactionDate:   transactionDate,
		Status:            models.TransactionStatusDraft,
		CurrentStage:      models.StageDraft,
		Notes:             req.Notes,
		CreatedBy:         userID,
	}

	if err := config.DB.Create(&transaction).Error; err != nil {
		return nil, err
	}

	return GetStockOpnameFlowDetail(transactionNumber)
}

// ============================================================
// ADD ASSET KE DRAFT
// ============================================================

func AddAssetToStockOpname(userID string, transactionNumber string, req dto.AddStockOpnameAssetRequest) (*dto.StockOpnameFlowDetailResponse, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageDraft {
		return nil, errors.New("can only add assets to DRAFT stock opnames")
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only modify your own stock opname draft")
	}

	var asset models.Asset
	if err := config.DB.First(&asset, req.AssetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("asset not found: %d", req.AssetID)
		}
		return nil, err
	}

	if asset.AssetNumber != req.AssetNumber {
		return nil, errors.New("asset number mismatch")
	}

	if asset.AssetStatus == models.AssetStatusDisposed {
		return nil, fmt.Errorf("asset %s is already disposed, cannot be opnamed", req.AssetNumber)
	}

	// Asset harus di branch yang sama dengan homebase aktif pembuat opname
	homebase, err := GetUserActiveHomebase(userID)
	if err != nil {
		return nil, err
	}
	if asset.BranchCode == nil || *asset.BranchCode != homebase.Branch.BranchCode {
		return nil, fmt.Errorf("asset %s is not in your branch", req.AssetNumber)
	}

	// Cek asset belum ada di draft ini
	var existingCount int64
	config.DB.Model(&models.TransactionStockOpname{}).
		Where("transaction_id = ? AND asset_id = ?", transaction.ID, req.AssetID).
		Count(&existingCount)
	if existingCount > 0 {
		return nil, fmt.Errorf("asset %s already added to this stock opname", req.AssetNumber)
	}

	// Cek asset tidak sedang dihitung di stock opname lain yang masih berjalan
	var otherOpnameCount int64
	config.DB.Model(&models.TransactionStockOpname{}).
		Joins("JOIN transactions ON transactions.id = transaction_stock_opnames.transaction_id").
		Where("transaction_stock_opnames.asset_id = ? AND transactions.transaction_type = ? AND transactions.id != ? AND transactions.current_stage NOT IN ?",
			req.AssetID, TxStockOpnameFlow, transaction.ID,
			[]string{models.StageFinished, models.StageRejected},
		).
		Count(&otherOpnameCount)
	if otherOpnameCount > 0 {
		return nil, fmt.Errorf("asset %s is already in another active stock opname", req.AssetNumber)
	}

	item := models.TransactionStockOpname{
		TransactionID:     transaction.ID,
		TransactionNumber: transactionNumber,
		AssetID:           asset.ID,
		AssetNumber:       asset.AssetNumber,
		AssetStatus:       asset.AssetStatus, // default awal, diedit lewat UpdateStockOpnameFinding
	}

	if err := config.DB.Create(&item).Error; err != nil {
		return nil, err
	}

	return GetStockOpnameFlowDetail(transactionNumber)
}

// ============================================================
// INPUT / UPDATE TEMUAN FISIK (masih DRAFT)
// ============================================================

func UpdateStockOpnameFinding(userID string, transactionNumber string, req dto.UpdateStockOpnameFindingRequest) (*dto.StockOpnameFlowDetailResponse, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageDraft {
		return nil, errors.New("can only update findings on DRAFT stock opnames")
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only modify your own stock opname draft")
	}

	var item models.TransactionStockOpname
	if err := config.DB.
		Where("transaction_id = ? AND asset_id = ?", transaction.ID, req.AssetID).
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("asset not found in this stock opname")
		}
		return nil, err
	}

	assetStatus := item.AssetStatus
	if req.AssetStatus != nil {
		assetStatus = *req.AssetStatus
	}

	if err := config.DB.Model(&item).Updates(map[string]interface{}{
		"physical_status": req.PhysicalStatus,
		"condition":       req.Condition,
		"asset_status":    assetStatus,
		"notes":           req.Notes,
	}).Error; err != nil {
		return nil, err
	}

	return GetStockOpnameFlowDetail(transactionNumber)
}

// ============================================================
// REMOVE ASSET DARI DRAFT
// ============================================================

func RemoveAssetFromStockOpname(userID string, transactionNumber string, req dto.RemoveStockOpnameAssetRequest) (*dto.StockOpnameFlowDetailResponse, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageDraft {
		return nil, errors.New("can only remove assets from DRAFT stock opnames")
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only modify your own stock opname draft")
	}

	result := config.DB.
		Where("transaction_id = ? AND asset_id = ?", transaction.ID, req.AssetID).
		Delete(&models.TransactionStockOpname{})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("asset not found in this stock opname")
	}

	return GetStockOpnameFlowDetail(transactionNumber)
}

// ============================================================
// SUBMIT
// DRAFT → APPROVAL
// ============================================================

func SubmitStockOpname(userID string, transactionNumber string, req dto.SubmitStockOpnameRequest) (*dto.StockOpnameFlowDetailResponse, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only submit your own stock opname")
	}

	if transaction.CurrentStage != models.StageDraft {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageDraft)
	}

	var items []models.TransactionStockOpname
	if err := config.DB.Where("transaction_id = ?", transaction.ID).Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("cannot submit stock opname with no assets")
	}
	for _, item := range items {
		if item.PhysicalStatus == "" || item.Condition == "" {
			return nil, fmt.Errorf("finding not yet filled for asset %s", item.AssetNumber)
		}
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

	return GetStockOpnameFlowDetail(transactionNumber)
}

// ============================================================
// INITIATE APPROVAL
// ============================================================

func InitiateStockOpnameApproval(userID string, transactionNumber string, req dto.InitiateApprovalRequest) error {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return err
	}

	if transaction.CurrentStage != models.StageApproval {
		return fmt.Errorf("transaction is not in %s stage", models.StageApproval)
	}

	creatorHomebase, homebaseErr := GetUserActiveHomebase(transaction.CreatedBy)
	branchCode := "ALL"
	if homebaseErr == nil {
		branchCode = creatorHomebase.Branch.BranchCode
	}

	flow, err := GetApprovalFlowByCodeAndBranch(models.FlowStockOpnameApproval, branchCode)
	if err != nil {
		return fmt.Errorf("approval flow %s not found for branch %s or ALL", models.FlowStockOpnameApproval, branchCode)
	}

	if !flow.IsActive {
		return fmt.Errorf("approval flow %s is inactive", models.FlowStockOpnameApproval)
	}

	approvalReq := dto.CreateTransactionApprovalRequest{
		FlowID:            flow.ID,
		TransactionNumber: transactionNumber,
		TransactionType:   TxStockOpnameFlow,
		Metadata:          req.Metadata,
	}

	return InitiateTransactionApproval(approvalReq)
}

// ============================================================
// AUTO-COMPLETE APPROVAL
// Dipanggil dari services/approval_service.go:ApproveTransaction setelah
// setiap approve, sama seperti autoCompleteMutationApproval.
// Semua step approved → APPROVAL → EXECUTE_STOCK_OPNAME (eksekusi manual
// oleh role yang berwenang, misal PIC Asset).
// ============================================================

func autoCompleteStockOpnameApproval(userID, transactionNumber, transactionType string) error {
	if transactionType != TxStockOpnameFlow {
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

	transaction, err := getStockOpnameTransaction(transactionNumber)
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
	if err := updateTransactionStage(tx, transaction, models.StageStockOpnameExecute); err != nil {
		tx.Rollback()
		return err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageStockOpnameExecute,
		models.ActionApprove, userID, nil, nil); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// ============================================================
// EKSEKUSI
// EXECUTE_STOCK_OPNAME → FINISHED
// Tulis temuan ke asset_values (snapshot baru) + assets.asset_status
// (jika berubah) + asset_histories (audit before/after).
// ============================================================

func ExecuteStockOpname(userID string, transactionNumber string, req dto.ExecuteStockOpnameRequest) (*dto.StockOpnameFlowDetailResponse, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageStockOpnameExecute {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageStockOpnameExecute)
	}

	var items []models.TransactionStockOpname
	if err := config.DB.Where("transaction_id = ?", transaction.ID).Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no assets found in this stock opname")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, item := range items {
		var asset models.Asset
		if err := tx.First(&asset, item.AssetID).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to load asset %s: %w", item.AssetNumber, err)
		}

		foundAssetStatus := item.AssetStatus
		if foundAssetStatus == "" {
			foundAssetStatus = asset.AssetStatus
		}

		var currentValue models.AssetValue
		hasCurrentValue := true
		if err := tx.Where("asset_id = ? AND is_active = ?", asset.ID, true).First(&currentValue).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				tx.Rollback()
				return nil, fmt.Errorf("failed to load current asset value for %s: %w", item.AssetNumber, err)
			}
			hasCurrentValue = false
		}

		before := map[string]interface{}{"asset_status": asset.AssetStatus}
		if hasCurrentValue {
			before["condition"] = currentValue.Condition
			before["physical_status"] = currentValue.PhysicalStatus
			if err := tx.Model(&currentValue).Update("is_active", false).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to deactivate current asset value for %s: %w", item.AssetNumber, err)
			}
		}

		newValue := models.AssetValue{
			AssetID:                 asset.ID,
			EffectiveDate:           time.Now(),
			BookValue:               currentValue.BookValue,
			AcquisitionValue:        currentValue.AcquisitionValue,
			AccumulatedDepreciation: currentValue.AccumulatedDepreciation,
			Condition:               &item.Condition,
			PhysicalStatus:          &item.PhysicalStatus,
			AssetStatus:             &foundAssetStatus,
			IsActive:                true,
		}
		if err := tx.Create(&newValue).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create new asset value for %s: %w", item.AssetNumber, err)
		}

		if foundAssetStatus != asset.AssetStatus {
			if err := tx.Model(&asset).Update("asset_status", foundAssetStatus).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to update asset status for %s: %w", item.AssetNumber, err)
			}
		}

		after := map[string]interface{}{
			"asset_status":    foundAssetStatus,
			"condition":       item.Condition,
			"physical_status": item.PhysicalStatus,
		}

		beforeJSON, _ := json.Marshal(before)
		afterJSON, _ := json.Marshal(after)
		beforeStr := string(beforeJSON)
		afterStr := string(afterJSON)

		if err := CreateAssetHistory(tx, models.AssetHistory{
			AssetID:         asset.ID,
			TransactionType: TxStockOpnameFlow,
			TransactionID:   &transaction.ID,
			DocumentNumber:  &transactionNumber,
			TransactionDate: &transaction.TransactionDate,
			BeforeData:      &beforeStr,
			AfterData:       &afterStr,
			ChangedBy:       &userID,
		}); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to record asset history for %s: %w", item.AssetNumber, err)
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

	return GetStockOpnameFlowDetail(transactionNumber)
}

// ============================================================
// REJECT
// ============================================================

func RejectStockOpname(userID string, transactionNumber string, req dto.RejectStockOpnameRequest) (*dto.StockOpnameFlowDetailResponse, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
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

	return GetStockOpnameFlowDetail(transactionNumber)
}

// ============================================================
// GET DETAIL
// ============================================================

func GetStockOpnameFlowDetail(transactionNumber string) (*dto.StockOpnameFlowDetailResponse, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	var items []models.TransactionStockOpname
	config.DB.
		Preload("Asset.Category").
		Where("transaction_id = ?", transaction.ID).
		Find(&items)

	var stages []models.TransactionStage
	config.DB.
		Where("transaction_id = ?", transaction.ID).
		Order("created_at ASC").
		Find(&stages)

	itemResponses := make([]dto.StockOpnameFlowItemResponse, len(items))
	for i, item := range items {
		itemResponses[i] = buildStockOpnameItemResponse(item)
	}

	return &dto.StockOpnameFlowDetailResponse{
		Transaction: mapTransactionHeaderToResponse(*transaction),
		Items:       itemResponses,
		Stages:      mapTransactionStagesToResponse(stages),
	}, nil
}

// buildStockOpnameItemResponse menggabungkan temuan auditor (item) dengan
// data sistem aktif saat ini (asset + active asset_value) untuk perbandingan.
func buildStockOpnameItemResponse(item models.TransactionStockOpname) dto.StockOpnameFlowItemResponse {
	resp := dto.StockOpnameFlowItemResponse{
		ID:                item.ID,
		TransactionID:     item.TransactionID,
		TransactionNumber: item.TransactionNumber,
		AssetID:           item.AssetID,
		AssetNumber:       item.AssetNumber,
		Notes:             item.Notes,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}

	if item.PhysicalStatus != "" {
		resp.FoundPhysicalStatus = &item.PhysicalStatus
	}
	if item.Condition != "" {
		resp.FoundCondition = &item.Condition
	}
	if item.AssetStatus != "" {
		resp.FoundAssetStatus = &item.AssetStatus
	}

	if item.Asset != nil {
		resp.AssetName = &item.Asset.AssetName
		resp.BranchCode = item.Asset.BranchCode
		resp.SystemAssetStatus = item.Asset.AssetStatus
		if item.Asset.Category != nil {
			resp.CategoryName = &item.Asset.Category.CategoryName
		}
	}

	var currentValue models.AssetValue
	if err := config.DB.
		Where("asset_id = ? AND is_active = ?", item.AssetID, true).
		First(&currentValue).Error; err == nil {
		resp.SystemCondition = currentValue.Condition
		resp.SystemPhysicalStatus = currentValue.PhysicalStatus
	}

	return resp
}

// ============================================================
// LIST
// ============================================================

type StockOpnameListFilter struct {
	Status       *string `form:"status"`
	CurrentStage *string `form:"current_stage"`
	CreatedBy    *string `form:"created_by"`
	StartDate    *string `form:"start_date"`
	EndDate      *string `form:"end_date"`
	Page         int     `form:"page"`
	Limit        int     `form:"limit"`
}

func GetAllStockOpnameDrafts(filter StockOpnameListFilter) ([]dto.StockOpnameFlowDetailResponse, int64, error) {
	query := config.DB.Model(&models.Transaction{}).
		Where("transaction_type = ?", TxStockOpnameFlow)

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

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit

	var transactions []models.Transaction
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(filter.Limit).
		Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	responses := make([]dto.StockOpnameFlowDetailResponse, len(transactions))
	for i, t := range transactions {
		detail, err := GetStockOpnameFlowDetail(t.TransactionNumber)
		if err != nil {
			return nil, 0, err
		}
		responses[i] = *detail
	}

	return responses, total, nil
}
