package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
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

// CreateStockOpnameDraft bikin draft baru dan langsung mengisi semua asset
// yang ada di branch homebase aktif si pembuat (1 user = 1 homebase aktif).
// Tidak ada lagi add/remove asset manual — daftar asset adalah snapshot
// kondisi branch pada saat draft dibuat.
func CreateStockOpnameDraft(userID string, req dto.CreateStockOpnameDraftRequest) (*dto.StockOpnameFlowDetailResponse, error) {
	now := time.Now()
	transactionDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	branchCode, err := GetUserActiveBranchCode(userID)
	if err != nil {
		return nil, errors.New("anda belum memiliki homebase aktif, silakan set homebase terlebih dahulu")
	}

	var branchAssets []models.Asset
	if err := config.DB.
		Where("branch_code = ? AND deleted_at IS NULL AND asset_status != ?", branchCode, models.AssetStatusDisposed).
		Find(&branchAssets).Error; err != nil {
		return nil, err
	}

	if len(branchAssets) == 0 {
		return nil, fmt.Errorf("branch %s belum memiliki asset terdaftar, tidak bisa membuat stock opname", branchCode)
	}

	// Exclude asset yang lagi kepake di stock opname lain yang masih berjalan
	assetIDs := make([]uint, len(branchAssets))
	for i, a := range branchAssets {
		assetIDs[i] = a.ID
	}

	var busyRows []struct {
		AssetID           uint
		TransactionNumber string
	}
	config.DB.Table("transaction_stock_opnames").
		Select("transaction_stock_opnames.asset_id, transactions.transaction_number").
		Joins("JOIN transactions ON transactions.id = transaction_stock_opnames.transaction_id").
		Where("transaction_stock_opnames.asset_id IN ? AND transactions.transaction_type = ? AND transactions.current_stage NOT IN ?",
			assetIDs, TxStockOpnameFlow,
			[]string{models.StageFinished, models.StageRejected},
		).
		Scan(&busyRows)

	busyAssetIDs := map[uint]bool{}
	blockingTxSeen := map[string]bool{}
	var blockingTxNumbers []string
	for _, row := range busyRows {
		busyAssetIDs[row.AssetID] = true
		if !blockingTxSeen[row.TransactionNumber] {
			blockingTxSeen[row.TransactionNumber] = true
			blockingTxNumbers = append(blockingTxNumbers, row.TransactionNumber)
		}
	}

	// Kalau semua asset di branch ini udah kepake stock opname lain yang
	// masih berjalan, gak ada gunanya bikin draft kosong — tolak dari awal
	// dengan pesan yang jelas biar user gak bingung liat draft tanpa asset.
	if len(busyAssetIDs) >= len(branchAssets) {
		blockingList := strings.Join(blockingTxNumbers, ", ")
		if len(blockingTxNumbers) > 3 {
			blockingList = strings.Join(blockingTxNumbers[:3], ", ") + fmt.Sprintf(" (+%d lainnya)", len(blockingTxNumbers)-3)
		}
		return nil, fmt.Errorf(
			"semua asset di branch %s sedang berada di stock opname lain yang masih berjalan: %s. Selesaikan (eksekusi) atau tolak stock opname tersebut dulu sebelum membuat draft baru",
			branchCode, blockingList,
		)
	}

	transactionNumber, err := GenerateTransactionNumber(userID, TxStockOpnameFlow)
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
		TransactionNumber: transactionNumber,
		TransactionType:   TxStockOpnameFlow,
		TransactionDate:   transactionDate,
		Status:            models.TransactionStatusDraft,
		CurrentStage:      models.StageDraft,
		Notes:             req.Notes,
		CreatedBy:         userID,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, asset := range branchAssets {
		if busyAssetIDs[asset.ID] {
			continue
		}
		item := models.TransactionStockOpname{
			TransactionID:     transaction.ID,
			TransactionNumber: transactionNumber,
			AssetID:           asset.ID,
			AssetNumber:       asset.AssetNumber,
			AssetStatus:       asset.AssetStatus, // default awal, diedit lewat UpdateStockOpnameFinding
		}
		if err := tx.Create(&item).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to add asset %s to draft: %w", asset.AssetNumber, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetStockOpnameFlowDetail(transactionNumber)
}

// ============================================================
// INPUT / UPDATE TEMUAN FISIK (masih DRAFT)
// ============================================================

// validateFindingPhysicalConditionPair mencegah kombinasi yang gak mungkin
// terjadi: kalau fisik aset "Tidak Ada" (MISSING), kondisinya wajib
// "Tidak Ada" (NOT_APPLICABLE) — dan sebaliknya, kalau fisiknya "Ada"
// (EXISTS), kondisinya gak boleh NOT_APPLICABLE karena harus dinilai.
func validateFindingPhysicalConditionPair(physicalStatus, condition string) error {
	if physicalStatus == "MISSING" && condition != "NOT_APPLICABLE" {
		return errors.New("condition must be NOT_APPLICABLE when physical_status is MISSING")
	}
	if physicalStatus == "EXISTS" && condition == "NOT_APPLICABLE" {
		return errors.New("condition cannot be NOT_APPLICABLE when physical_status is EXISTS")
	}
	return nil
}

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

	if err := validateFindingPhysicalConditionPair(req.PhysicalStatus, req.Condition); err != nil {
		return nil, err
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

// ============================================================
// EXCEL TEMPLATE — label <-> enum mapping
// Dipakai bersama oleh download (enum -> label) dan upload (label -> enum).
// ============================================================

var stockOpnamePhysicalStatusLabel = map[string]string{
	"EXISTS":  "Ada",
	"MISSING": "Tidak Ada",
}

var stockOpnamePhysicalStatusValue = map[string]string{
	"ada":       "EXISTS",
	"tidak ada": "MISSING",
}

var stockOpnameConditionLabel = map[string]string{
	"GOOD":           "Baik",
	"FAIR":           "Cukup",
	"POOR":           "Kurang",
	"BROKEN":         "Rusak Berat",
	"NOT_APPLICABLE": "Tidak Ada",
}

var stockOpnameConditionValue = map[string]string{
	"baik":        "GOOD",
	"cukup":       "FAIR",
	"kurang":      "POOR",
	"rusak berat": "BROKEN",
	"tidak ada":   "NOT_APPLICABLE",
}

var stockOpnameAssetStatusLabel = map[string]string{
	"ACTIVE":      "Aktif",
	"INACTIVE":    "Tidak Aktif",
	"MAINTENANCE": "Maintenance",
	"RETIRED":     "Retired",
}

var stockOpnameAssetStatusValue = map[string]string{
	"aktif":       "ACTIVE",
	"tidak aktif": "INACTIVE",
	"maintenance": "MAINTENANCE",
	"retired":     "RETIRED",
}

const stockOpnameTemplateSheet = "Stock Opname"

// ============================================================
// EXCEL TEMPLATE — DOWNLOAD
// Snapshot semua asset di draft + temuan yang udah keisi (kalau ada),
// dipakai user sebagai template buat diisi/diedit lalu diupload balik.
// ============================================================

func GenerateStockOpnameTemplateExcel(userID, transactionNumber string) (*excelize.File, string, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, "", err
	}
	if transaction.CreatedBy != userID {
		return nil, "", errors.New("you can only download the template for your own stock opname draft")
	}

	var items []models.TransactionStockOpname
	if err := config.DB.
		Preload("Asset.Category").
		Where("transaction_id = ?", transaction.ID).
		Order("asset_number ASC").
		Find(&items).Error; err != nil {
		return nil, "", err
	}

	f := excelize.NewFile()
	f.SetSheetName("Sheet1", stockOpnameTemplateSheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	headers := []string{
		"No", "Nomor Asset", "Nama Asset", "Kategori",
		"Status Fisik (Ada/Tidak Ada)",
		"Kondisi (Baik/Cukup/Kurang/Rusak Berat/Tidak Ada)",
		"Status Aset (opsional, kosongkan jika tidak berubah)",
		"Catatan",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(stockOpnameTemplateSheet, cell, h)
		f.SetCellStyle(stockOpnameTemplateSheet, cell, cell, headerStyle)
	}

	for i, item := range items {
		row := i + 2
		assetName, categoryName := "", ""
		if item.Asset != nil {
			assetName = item.Asset.AssetName
			if item.Asset.Category != nil {
				categoryName = item.Asset.Category.CategoryName
			}
		}
		notes := ""
		if item.Notes != nil {
			notes = *item.Notes
		}

		values := []interface{}{
			i + 1,
			item.AssetNumber,
			assetName,
			categoryName,
			stockOpnamePhysicalStatusLabel[item.PhysicalStatus],
			stockOpnameConditionLabel[item.Condition],
			stockOpnameAssetStatusLabel[item.AssetStatus],
			notes,
		}
		for col, v := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			f.SetCellValue(stockOpnameTemplateSheet, cell, v)
		}
	}

	colWidths := map[string]float64{"A": 6, "B": 18, "C": 28, "D": 18, "E": 26, "F": 36, "G": 40, "H": 40}
	for col, w := range colWidths {
		f.SetColWidth(stockOpnameTemplateSheet, col, col, w)
	}

	safeName := strings.NewReplacer("/", "-", " ", "_").Replace(transactionNumber)
	filename := fmt.Sprintf("stock_opname_template_%s.xlsx", safeName)
	return f, filename, nil
}

// ============================================================
// EXCEL TEMPLATE — UPLOAD
// Bulk update temuan berdasarkan file yang diisi user. Partial update:
// baris valid langsung disimpan, baris invalid dikumpulkan sebagai error
// report (gak menggagalkan keseluruhan proses).
// ============================================================

func cellAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func ProcessStockOpnameTemplateUpload(userID, transactionNumber string, file io.Reader) (*dto.StockOpnameTemplateUploadResponse, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}
	if transaction.CurrentStage != models.StageDraft {
		return nil, errors.New("can only upload findings for DRAFT stock opnames")
	}
	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only modify your own stock opname draft")
	}

	xf, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read excel file: %w", err)
	}
	defer xf.Close()

	sheet := xf.GetSheetName(0)
	rows, err := xf.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to read excel rows: %w", err)
	}

	var items []models.TransactionStockOpname
	if err := config.DB.Where("transaction_id = ?", transaction.ID).Find(&items).Error; err != nil {
		return nil, err
	}
	itemByAssetNumber := make(map[string]models.TransactionStockOpname, len(items))
	for _, item := range items {
		itemByAssetNumber[item.AssetNumber] = item
	}

	response := &dto.StockOpnameTemplateUploadResponse{Errors: []dto.StockOpnameTemplateRowError{}}

	for i, row := range rows {
		rowNum := i + 1
		if rowNum == 1 {
			continue // header
		}

		assetNumber := cellAt(row, 1)
		if assetNumber == "" {
			continue // baris kosong, skip diam-diam
		}

		item, ok := itemByAssetNumber[assetNumber]
		if !ok {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: assetNumber,
				Message: "asset tidak ditemukan di stock opname ini",
			})
			continue
		}

		physicalLabel := strings.ToLower(cellAt(row, 4))
		conditionLabel := strings.ToLower(cellAt(row, 5))
		assetStatusLabel := strings.ToLower(cellAt(row, 6))
		notesCell := cellAt(row, 7)

		physicalStatus, ok := stockOpnamePhysicalStatusValue[physicalLabel]
		if !ok {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: assetNumber,
				Message: fmt.Sprintf("status fisik tidak valid: %q (harus Ada / Tidak Ada)", cellAt(row, 4)),
			})
			continue
		}

		condition, ok := stockOpnameConditionValue[conditionLabel]
		if !ok {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: assetNumber,
				Message: fmt.Sprintf("kondisi tidak valid: %q", cellAt(row, 5)),
			})
			continue
		}

		if err := validateFindingPhysicalConditionPair(physicalStatus, condition); err != nil {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: assetNumber,
				Message: err.Error(),
			})
			continue
		}

		assetStatus := item.AssetStatus
		if assetStatusLabel != "" {
			mapped, ok := stockOpnameAssetStatusValue[assetStatusLabel]
			if !ok {
				response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
					Row: rowNum, AssetNumber: assetNumber,
					Message: fmt.Sprintf("status aset tidak valid: %q", cellAt(row, 6)),
				})
				continue
			}
			assetStatus = mapped
		}

		updates := map[string]interface{}{
			"physical_status": physicalStatus,
			"condition":       condition,
			"asset_status":    assetStatus,
			"notes":           notesCell,
		}

		if err := config.DB.Model(&models.TransactionStockOpname{}).
			Where("id = ?", item.ID).
			Updates(updates).Error; err != nil {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: assetNumber,
				Message: "gagal menyimpan: " + err.Error(),
			})
			continue
		}

		response.UpdatedCount++
	}

	response.FailedCount = len(response.Errors)

	detail, err := GetStockOpnameFlowDetail(transactionNumber)
	if err != nil {
		return nil, err
	}
	response.Detail = detail

	return response, nil
}
