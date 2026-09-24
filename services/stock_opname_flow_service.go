package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // registrasi decoder image/jpeg — dipakai excelize.AddPicture buat baca dimensi foto JPEG
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	_ "golang.org/x/image/webp" // registrasi decoder image/webp — dipakai resolveStockOpnamePicturePath
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

	// Gak ada lagi pengecualian "asset lagi kepake stock opname lain yang
	// masih berjalan" — user yang sama boleh bikin stock opname baru kapan
	// aja, termasuk hari yang sama & overlap asset dengan stock opname lain
	// yang belum selesai. Tiap stock opname independen satu sama lain.
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
		item := models.StockOpnameDraftItem{
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

// validateFindingPhysicalConditionPair memvalidasi bahwa physicalStatus &
// condition beneran ada di master data (bukan hardcode lagi, lihat
// services/stock_opname_status_master_service.go) DAN mencegah kombinasi
// yang gak mungkin terjadi: kalau physicalStatus.RequiresNotApplicableCondition
// true (dulu hardcode "MISSING"/"BORROWED"), condition WAJIB salah satu yang
// IsNotApplicableValue — dan sebaliknya kalau false (dulu "EXISTS"), condition
// JUSTRU DILARANG pakai nilai IsNotApplicableValue karena harus benar-benar
// dinilai.
func validateFindingPhysicalConditionPair(
	physicalMap map[string]models.StockOpnamePhysicalStatusMaster,
	conditionMap map[string]models.StockOpnameConditionMaster,
	physicalStatus, condition string,
) error {
	pm, ok := physicalMap[physicalStatus]
	if !ok {
		return fmt.Errorf("physical_status tidak valid: %q", physicalStatus)
	}
	cm, ok := conditionMap[condition]
	if !ok {
		return fmt.Errorf("condition tidak valid: %q", condition)
	}
	if pm.RequiresNotApplicableCondition && !cm.IsNotApplicableValue {
		return fmt.Errorf("condition must be a not-applicable value when physical_status is %s", physicalStatus)
	}
	if !pm.RequiresNotApplicableCondition && cm.IsNotApplicableValue {
		return fmt.Errorf("condition cannot be a not-applicable value when physical_status is %s", physicalStatus)
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

	physicalMap, err := loadStockOpnamePhysicalStatusMap()
	if err != nil {
		return nil, err
	}
	conditionMap, err := loadStockOpnameConditionMap()
	if err != nil {
		return nil, err
	}
	if err := validateFindingPhysicalConditionPair(physicalMap, conditionMap, req.PhysicalStatus, req.Condition); err != nil {
		return nil, err
	}

	var item models.StockOpnameDraftItem
	if err := config.DB.
		Where("transaction_id = ? AND asset_id = ?", transaction.ID, req.AssetID).
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("asset not found in this stock opname")
		}
		return nil, err
	}

	if err := ensureStockOpnameAssetEditable(transaction.ID, req.AssetID); err != nil {
		return nil, err
	}

	if physicalMap[req.PhysicalStatus].RequiresBorrowDocument {
		soCfg, err := getOrCreateStockOpnameConfig()
		if err != nil {
			return nil, err
		}
		if soCfg.BorrowDocIsRequired {
			var docCount int64
			config.DB.Model(&models.StockOpnameBorrowDocument{}).
				Where("transaction_id = ? AND asset_id = ?", transaction.ID, req.AssetID).
				Count(&docCount)
			if docCount == 0 {
				return nil, errors.New("dokumen peminjaman wajib diupload dulu sebelum menyimpan status Dipinjam")
			}
		}
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

// BulkUpdateStockOpnameFinding dipakai autosave grid "Lengkapi Data" —
// menyimpan banyak asset sekaligus dalam satu request (biasanya dipanggil
// tiap beberapa detik dari FE, bukan tiap keystroke). Partial update per
// item, sama semantiknya kayak upload excel: baris valid tetap kesimpen,
// baris invalid dikumpulkan sebagai error report tanpa gagalin baris lain.
func BulkUpdateStockOpnameFinding(userID string, transactionNumber string, req dto.BulkUpdateStockOpnameFindingRequest) (*dto.StockOpnameTemplateUploadResponse, error) {
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

	var existing []models.StockOpnameDraftItem
	if err := config.DB.Where("transaction_id = ?", transaction.ID).Find(&existing).Error; err != nil {
		return nil, err
	}
	byAssetID := make(map[uint]models.StockOpnameDraftItem, len(existing))
	for _, item := range existing {
		byAssetID[item.AssetID] = item
	}

	revisionScope, revisionActive, err := loadStockOpnameRevisionScope(transaction.ID)
	if err != nil {
		return nil, err
	}

	var borrowDocAssetIDs []uint
	config.DB.Model(&models.StockOpnameBorrowDocument{}).
		Where("transaction_id = ?", transaction.ID).
		Pluck("asset_id", &borrowDocAssetIDs)
	hasBorrowDoc := make(map[uint]bool, len(borrowDocAssetIDs))
	for _, id := range borrowDocAssetIDs {
		hasBorrowDoc[id] = true
	}

	soCfg, err := getOrCreateStockOpnameConfig()
	if err != nil {
		return nil, err
	}

	physicalMap, err := loadStockOpnamePhysicalStatusMap()
	if err != nil {
		return nil, err
	}
	conditionMap, err := loadStockOpnameConditionMap()
	if err != nil {
		return nil, err
	}

	response := &dto.StockOpnameTemplateUploadResponse{Errors: []dto.StockOpnameTemplateRowError{}}

	for i, reqItem := range req.Items {
		rowNum := i + 1

		current, ok := byAssetID[reqItem.AssetID]
		if !ok {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: fmt.Sprintf("asset_id=%d", reqItem.AssetID),
				Message: "asset not found in this stock opname",
			})
			continue
		}

		if revisionActive && !revisionScope[reqItem.AssetID] {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: current.AssetNumber, Message: errStockOpnameAssetNotInRevision,
			})
			continue
		}

		// Value non-empty tapi gak dikenal enum-nya ditolak; string kosong
		// selalu diloloskan di sini (representasi "field sengaja dikosongkan
		// lagi" saat user ganti pilihan) — lihat komentar di dto.
		if reqItem.PhysicalStatus != nil && *reqItem.PhysicalStatus != "" {
			if _, ok := physicalMap[*reqItem.PhysicalStatus]; !ok {
				response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
					Row: rowNum, AssetNumber: current.AssetNumber,
					Message: fmt.Sprintf("status fisik tidak valid: %q", *reqItem.PhysicalStatus),
				})
				continue
			}
		}
		if reqItem.Condition != nil && *reqItem.Condition != "" {
			if _, ok := conditionMap[*reqItem.Condition]; !ok {
				response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
					Row: rowNum, AssetNumber: current.AssetNumber,
					Message: fmt.Sprintf("kondisi tidak valid: %q", *reqItem.Condition),
				})
				continue
			}
		}
		if reqItem.AssetStatus != nil && *reqItem.AssetStatus != "" {
			if _, ok := stockOpnameAssetStatusLabel[*reqItem.AssetStatus]; !ok {
				response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
					Row: rowNum, AssetNumber: current.AssetNumber,
					Message: fmt.Sprintf("status aset tidak valid: %q", *reqItem.AssetStatus),
				})
				continue
			}
		}

		effectivePhysical := current.PhysicalStatus
		if reqItem.PhysicalStatus != nil {
			effectivePhysical = *reqItem.PhysicalStatus
		}
		effectiveCondition := current.Condition
		if reqItem.Condition != nil {
			effectiveCondition = *reqItem.Condition
		}

		// Cuma divalidasi kalau dua-duanya udah keisi — kalau salah satu
		// masih kosong berarti user masih di tengah proses ngisi cell lain,
		// belum saatnya divalidasi silang.
		if effectivePhysical != "" && effectiveCondition != "" {
			if err := validateFindingPhysicalConditionPair(physicalMap, conditionMap, effectivePhysical, effectiveCondition); err != nil {
				response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
					Row: rowNum, AssetNumber: current.AssetNumber, Message: err.Error(),
				})
				continue
			}
		}

		if physicalMap[effectivePhysical].RequiresBorrowDocument && soCfg.BorrowDocIsRequired && !hasBorrowDoc[current.AssetID] {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: current.AssetNumber,
				Message: "dokumen peminjaman wajib diupload dulu sebelum menyimpan status Dipinjam",
			})
			continue
		}

		updates := map[string]interface{}{}
		if reqItem.PhysicalStatus != nil {
			updates["physical_status"] = *reqItem.PhysicalStatus
		}
		if reqItem.Condition != nil {
			updates["condition"] = *reqItem.Condition
		}
		if reqItem.AssetStatus != nil {
			updates["asset_status"] = *reqItem.AssetStatus
		}
		if reqItem.Notes != nil {
			updates["notes"] = *reqItem.Notes
		}

		if len(updates) == 0 {
			continue
		}

		if err := config.DB.Model(&models.StockOpnameDraftItem{}).
			Where("id = ?", current.ID).
			Updates(updates).Error; err != nil {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: current.AssetNumber, Message: "gagal menyimpan: " + err.Error(),
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

	var items []models.StockOpnameDraftItem
	if err := config.DB.Where("transaction_id = ?", transaction.ID).Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("cannot submit stock opname with no assets")
	}

	var photos []models.StockOpnameAssetPhoto
	if err := config.DB.Where("transaction_id = ?", transaction.ID).Find(&photos).Error; err != nil {
		return nil, err
	}
	photoByAsset := make(map[uint]models.StockOpnameAssetPhoto, len(photos))
	for _, p := range photos {
		photoByAsset[p.AssetID] = p
	}

	var borrowDocAssetIDs []uint
	config.DB.Model(&models.StockOpnameBorrowDocument{}).
		Where("transaction_id = ?", transaction.ID).
		Pluck("asset_id", &borrowDocAssetIDs)
	hasBorrowDoc := make(map[uint]bool, len(borrowDocAssetIDs))
	for _, id := range borrowDocAssetIDs {
		hasBorrowDoc[id] = true
	}

	soCfg, err := getOrCreateStockOpnameConfig()
	if err != nil {
		return nil, err
	}

	physicalMap, err := loadStockOpnamePhysicalStatusMap()
	if err != nil {
		return nil, err
	}

	revisionScope, revisionActive, err := loadStockOpnameRevisionScope(transaction.ID)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	for _, item := range items {
		if item.PhysicalStatus == "" || item.Condition == "" {
			return nil, fmt.Errorf("finding not yet filled for asset %s", item.AssetNumber)
		}
		photo, hasPhoto := photoByAsset[item.AssetID]
		if !hasPhoto {
			return nil, fmt.Errorf("foto bukti fisik belum dilampirkan untuk asset %s", item.AssetNumber)
		}
		// Validasi upload-time (lihat UploadStockOpnameAssetPhoto) cuma cek
		// tanggal modifikasi file relatif ke SAAT UPLOAD — draft yang dibiarkan
		// lama sebelum di-submit bisa lolos padahal fotonya udah basi. Di sini
		// dicek ulang relatif ke SAAT SUBMIT, pakai created_at (kapan foto
		// itu benar-benar sampai ke server), bukan captured_at.
		// Asset yang dikunci selama revisi di-skip: fotonya udah lolos di
		// submit sebelumnya dan creator gak bisa upload ulang.
		lockedByRevision := revisionActive && !revisionScope[item.AssetID]
		if !lockedByRevision && now.Sub(photo.CreatedAt) > stockOpnamePhotoMaxAge {
			return nil, fmt.Errorf(
				"foto bukti fisik untuk asset %s sudah diupload lebih dari 10 hari sebelum submit (upload: %s) — upload ulang foto yang lebih baru",
				item.AssetNumber, photo.CreatedAt.Format("02 Jan 2006"),
			)
		}
		if physicalMap[item.PhysicalStatus].RequiresBorrowDocument && soCfg.BorrowDocIsRequired && !hasBorrowDoc[item.AssetID] {
			return nil, fmt.Errorf("dokumen peminjaman belum dilampirkan untuk asset %s", item.AssetNumber)
		}
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Pindahkan item dari "draft" ke "aktif" (transaction_stock_opnames) —
	// insert-select (tanpa kolom id, biar dapet id baru di tabel tujuan)
	// lalu delete dari tabel asal, dua-duanya dalam tx yang sama biar atomik.
	if err := tx.Exec(`
		INSERT INTO transaction_stock_opnames
			(transaction_id, transaction_number, asset_id, asset_number, physical_status, `+"`condition`"+`, asset_status, notes, created_at, updated_at)
		SELECT transaction_id, transaction_number, asset_id, asset_number, physical_status, `+"`condition`"+`, asset_status, notes, created_at, updated_at
		FROM stock_opname_draft_items
		WHERE transaction_id = ?
	`, transaction.ID).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to move draft items to active table: %w", err)
	}
	if err := tx.Exec(`DELETE FROM stock_opname_draft_items WHERE transaction_id = ?`, transaction.ID).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to clear moved draft items: %w", err)
	}

	// Putaran revisi selesai begitu di-submit ulang — kunci asset dilepas.
	if err := tx.Where("transaction_id = ?", transaction.ID).
		Delete(&models.StockOpnameRevisionAsset{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// IsSubmissive cuma penanda kepatuhan jadwal (dinilai dari tanggal
	// SUBMIT, bukan tanggal draft dibuat) — gak pernah menghalangi submit
	// itu sendiri, submit di luar jendela tetap jalan seperti biasa.
	isSubmissive := isWithinSubmissionWindow(now, soCfg.SubmissionStartDay, soCfg.SubmissionEndDay)
	if err := tx.Model(transaction).Update("is_submissive", isSubmissive).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

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

	// Pindahkan item dari "aktif" (transaction_stock_opnames) ke "history" —
	// sama pola insert-select + delete kayak di SubmitStockOpname, dalam tx
	// yang sama biar atomik dengan penulisan asset_values/asset_histories
	// di atas dan perubahan stage di bawah.
	if err := tx.Exec(`
		INSERT INTO stock_opname_item_history
			(transaction_id, transaction_number, asset_id, asset_number, physical_status, `+"`condition`"+`, asset_status, notes, created_at, updated_at)
		SELECT transaction_id, transaction_number, asset_id, asset_number, physical_status, `+"`condition`"+`, asset_status, notes, created_at, updated_at
		FROM transaction_stock_opnames
		WHERE transaction_id = ?
	`, transaction.ID).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to move items to history table: %w", err)
	}
	if err := tx.Exec(`DELETE FROM transaction_stock_opnames WHERE transaction_id = ?`, transaction.ID).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to clear moved active items: %w", err)
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
// AUTO-REJECT APPROVAL
// Dipanggil dari services/approval_service.go:RejectTransaction setelah
// approver me-reject salah satu step — pasangan autoCompleteStockOpnameApproval.
// Tanpa ini transaksi nyangkut di APPROVAL dengan approval berstatus rejected.
// APPROVAL → REJECTED (final, gak bisa direvisi lagi).
// ============================================================

func autoRejectStockOpnameApproval(userID, transactionNumber, transactionType, notes string) error {
	if transactionType != TxStockOpnameFlow {
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
	if err := updateTransactionStage(tx, transaction, models.StageRejected); err != nil {
		tx.Rollback()
		return err
	}

	reason := "Rejected by approver"
	if notes != "" {
		reason = notes
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageRejected,
		models.ActionReject, userID, nil, &reason); err != nil {
		tx.Rollback()
		return err
	}

	MarkTransactionAsExpired(transactionNumber)

	return tx.Commit().Error
}

// ============================================================
// REVISI
// APPROVAL / EXECUTE_STOCK_OPNAME → DRAFT
// Ditrigger approver (step yang lagi pending) atau eksekutor, bukan creator.
// Reviser mencentang asset mana yang perlu dibenerin (atau "revisi semua");
// selama DRAFT revisi cuma asset itu yang boleh diubah creator, sisanya
// dikunci (lihat ensureStockOpnameAssetEditable). Beda dengan REJECTED yang
// final.
// ============================================================

// ReviseStockOpnameByApprover: revisi dari stage APPROVAL, otorisasinya sama
// persis dengan approve/reject (approver user/role + branch).
func ReviseStockOpnameByApprover(userID string, transactionNumber string, req dto.ReviseStockOpnameByApproverRequest) (*dto.StockOpnameFlowDetailResponse, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}
	if transaction.CurrentStage != models.StageApproval {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageApproval)
	}

	var approval models.TransactionApproval
	if err := config.DB.
		Where("id = ? AND transaction_number = ? AND transaction_type = ?", req.TransactionApprovalID, transactionNumber, TxStockOpnameFlow).
		First(&approval).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("transaction approval not found for this stock opname")
		}
		return nil, err
	}
	if approval.Status != "pending" {
		return nil, errors.New("approval already processed")
	}
	if approval.ApproverUserID != nil && *approval.ApproverUserID != userID {
		return nil, errors.New("you are not authorized to revise this transaction")
	}
	if approval.ApproverRoleID != nil {
		var userRole models.UserRole
		if err := config.DB.
			Where("user_id = ? AND role_id = ?", userID, *approval.ApproverRoleID).
			First(&userRole).Error; err != nil {
			return nil, errors.New("you do not have the required role to revise this transaction")
		}
	}
	if err := validateApproverBranch(userID, transactionNumber, TxStockOpnameFlow); err != nil {
		return nil, err
	}

	return reviseStockOpnameToDraft(userID, transaction, req.AssetIDs, req.ReviseAll, req.RevisionNotes)
}

// ReviseStockOpnameByExecutor: revisi dari stage EXECUTE_STOCK_OPNAME,
// otorisasinya lewat permission execute_stock_opname di route.
func ReviseStockOpnameByExecutor(userID string, transactionNumber string, req dto.ReviseStockOpnameRequest) (*dto.StockOpnameFlowDetailResponse, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}
	if transaction.CurrentStage != models.StageStockOpnameExecute {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageStockOpnameExecute)
	}

	return reviseStockOpnameToDraft(userID, transaction, req.AssetIDs, req.ReviseAll, req.RevisionNotes)
}

// reviseStockOpnameToDraft: item dibalikin dari transaction_stock_opnames ke
// stock_opname_draft_items (kebalikan SubmitStockOpname), asset yang
// dichecklist dicatat di stock_opname_revision_assets, approval lama
// di-soft-delete biar di-initiate ulang dari step pertama setelah submit
// berikutnya. Foto & dokumen peminjaman gak ikut dipindah — dikunci lewat
// (transaction_id, asset_id), jadi otomatis kebawa lagi ke draft.
func reviseStockOpnameToDraft(userID string, transaction *models.Transaction, assetIDs []uint, reviseAll bool, notes string) (*dto.StockOpnameFlowDetailResponse, error) {
	var itemAssetIDs []uint
	if err := config.DB.Model(&models.TransactionStockOpname{}).
		Where("transaction_id = ?", transaction.ID).
		Pluck("asset_id", &itemAssetIDs).Error; err != nil {
		return nil, err
	}
	inTransaction := make(map[uint]bool, len(itemAssetIDs))
	for _, id := range itemAssetIDs {
		inTransaction[id] = true
	}

	revisionAssetIDs := itemAssetIDs
	if !reviseAll {
		seen := make(map[uint]bool, len(assetIDs))
		revisionAssetIDs = make([]uint, 0, len(assetIDs))
		for _, id := range assetIDs {
			if !inTransaction[id] {
				return nil, fmt.Errorf("asset_id %d not found in this stock opname", id)
			}
			if !seen[id] {
				seen[id] = true
				revisionAssetIDs = append(revisionAssetIDs, id)
			}
		}
	}
	if len(revisionAssetIDs) == 0 {
		return nil, errors.New("pilih minimal 1 asset yang perlu direvisi, atau pakai revisi semua")
	}

	transactionNumber := transaction.TransactionNumber

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Exec(`
		INSERT INTO stock_opname_draft_items
			(transaction_id, transaction_number, asset_id, asset_number, physical_status, `+"`condition`"+`, asset_status, notes, created_at, updated_at)
		SELECT transaction_id, transaction_number, asset_id, asset_number, physical_status, `+"`condition`"+`, asset_status, notes, created_at, updated_at
		FROM transaction_stock_opnames
		WHERE transaction_id = ?
	`, transaction.ID).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to move active items back to draft table: %w", err)
	}
	if err := tx.Exec(`DELETE FROM transaction_stock_opnames WHERE transaction_id = ?`, transaction.ID).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to clear moved active items: %w", err)
	}

	// Sisa putaran revisi sebelumnya (harusnya udah kehapus pas submit)
	if err := tx.Where("transaction_id = ?", transaction.ID).
		Delete(&models.StockOpnameRevisionAsset{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	revisionRows := make([]models.StockOpnameRevisionAsset, len(revisionAssetIDs))
	for i, id := range revisionAssetIDs {
		revisionRows[i] = models.StockOpnameRevisionAsset{TransactionID: transaction.ID, AssetID: id, CreatedBy: userID}
	}
	if err := tx.CreateInBatches(&revisionRows, 500).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// InitiateTransactionApproval nolak kalau masih ada approval (non-deleted)
	// untuk transaksi ini, jadi approval putaran lama harus dibuang.
	if err := tx.Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxStockOpnameFlow).
		Delete(&models.TransactionApproval{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Model(&models.ApprovalSignature{}).
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxStockOpnameFlow).
		Update("is_recent", false).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageDraft); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageDraft,
		models.ActionRevise, userID, nil, &notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetStockOpnameFlowDetail(transactionNumber)
}

// loadStockOpnameRevisionScope: asset yang boleh diubah selama DRAFT revisi.
// active=false berarti draft biasa (belum pernah direvisi / udah di-submit
// ulang) — semua asset boleh diubah.
func loadStockOpnameRevisionScope(transactionID uint) (scope map[uint]bool, active bool, err error) {
	var ids []uint
	if err := config.DB.Model(&models.StockOpnameRevisionAsset{}).
		Where("transaction_id = ?", transactionID).
		Pluck("asset_id", &ids).Error; err != nil {
		return nil, false, err
	}
	scope = make(map[uint]bool, len(ids))
	for _, id := range ids {
		scope[id] = true
	}
	return scope, len(ids) > 0, nil
}

const errStockOpnameAssetNotInRevision = "asset ini tidak termasuk yang diminta revisi — cuma asset yang dichecklist yang boleh diubah"

func ensureStockOpnameAssetEditable(transactionID, assetID uint) error {
	scope, active, err := loadStockOpnameRevisionScope(transactionID)
	if err != nil {
		return err
	}
	if active && !scope[assetID] {
		return errors.New(errStockOpnameAssetNotInRevision)
	}
	return nil
}

// ============================================================
// GET DETAIL
// ============================================================

// stockOpnameItemRow adalah representasi umum 1 baris temuan stock opname,
// dipakai supaya GetStockOpnameFlowDetail bisa nampilin item dari salah satu
// dari 3 tabel (draft/aktif/history — tergantung stage transaksi) lewat 1
// kode path yang sama, tanpa peduli lagi row-nya sekarang fisik ada di tabel
// mana.
type stockOpnameItemRow struct {
	ID                uint
	TransactionID     uint
	TransactionNumber string
	AssetID           uint
	AssetNumber       string
	PhysicalStatus    string
	Condition         string
	AssetStatus       string
	Notes             *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Asset             *models.Asset
}

func draftItemToRow(i models.StockOpnameDraftItem) stockOpnameItemRow {
	return stockOpnameItemRow{
		ID: i.ID, TransactionID: i.TransactionID, TransactionNumber: i.TransactionNumber,
		AssetID: i.AssetID, AssetNumber: i.AssetNumber, PhysicalStatus: i.PhysicalStatus,
		Condition: i.Condition, AssetStatus: i.AssetStatus, Notes: i.Notes,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt, Asset: i.Asset,
	}
}

func activeItemToRow(i models.TransactionStockOpname) stockOpnameItemRow {
	return stockOpnameItemRow{
		ID: i.ID, TransactionID: i.TransactionID, TransactionNumber: i.TransactionNumber,
		AssetID: i.AssetID, AssetNumber: i.AssetNumber, PhysicalStatus: i.PhysicalStatus,
		Condition: i.Condition, AssetStatus: i.AssetStatus, Notes: i.Notes,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt, Asset: i.Asset,
	}
}

func historyItemToRow(i models.StockOpnameItemHistory) stockOpnameItemRow {
	return stockOpnameItemRow{
		ID: i.ID, TransactionID: i.TransactionID, TransactionNumber: i.TransactionNumber,
		AssetID: i.AssetID, AssetNumber: i.AssetNumber, PhysicalStatus: i.PhysicalStatus,
		Condition: i.Condition, AssetStatus: i.AssetStatus, Notes: i.Notes,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt, Asset: i.Asset,
	}
}

// getStockOpnameItemRows membaca item stock opname dari tabel yang tepat
// sesuai stage transaksinya (lihat komentar stockOpnameItemRow) — dipakai
// bareng oleh GetStockOpnameFlowDetail dan GenerateStockOpnameDocumentationExcel
// biar logic "baca dari tabel yang mana" gak keduplikasi.
func getStockOpnameItemRows(transaction *models.Transaction) []stockOpnameItemRow {
	var itemRows []stockOpnameItemRow
	switch transaction.CurrentStage {
	case models.StageDraft:
		var items []models.StockOpnameDraftItem
		config.DB.Preload("Asset.Category").Where("transaction_id = ?", transaction.ID).Order("asset_number ASC").Find(&items)
		for _, it := range items {
			itemRows = append(itemRows, draftItemToRow(it))
		}
	case models.StageFinished:
		var items []models.StockOpnameItemHistory
		config.DB.Preload("Asset.Category").Where("transaction_id = ?", transaction.ID).Order("asset_number ASC").Find(&items)
		for _, it := range items {
			itemRows = append(itemRows, historyItemToRow(it))
		}
	default:
		var items []models.TransactionStockOpname
		config.DB.Preload("Asset.Category").Where("transaction_id = ?", transaction.ID).Order("asset_number ASC").Find(&items)
		for _, it := range items {
			itemRows = append(itemRows, activeItemToRow(it))
		}
	}
	return itemRows
}

func GetStockOpnameFlowDetail(transactionNumber string) (*dto.StockOpnameFlowDetailResponse, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	// Item stock opname sekarang tersebar di 3 tabel tergantung stage:
	// DRAFT -> stock_opname_draft_items, FINISHED -> stock_opname_item_history,
	// selain itu (APPROVAL/EXECUTE_STOCK_OPNAME/REJECTED) -> transaction_stock_opnames.
	itemRows := getStockOpnameItemRows(transaction)

	var stages []models.TransactionStage
	config.DB.
		Where("transaction_id = ?", transaction.ID).
		Order("created_at ASC").
		Find(&stages)

	var photos []models.StockOpnameAssetPhoto
	config.DB.Where("transaction_id = ?", transaction.ID).Find(&photos)
	photoByAssetID := make(map[uint]models.StockOpnameAssetPhoto, len(photos))
	for _, p := range photos {
		photoByAssetID[p.AssetID] = p
	}

	var borrowDocs []models.StockOpnameBorrowDocument
	config.DB.Where("transaction_id = ?", transaction.ID).Find(&borrowDocs)
	borrowDocByAssetID := make(map[uint]models.StockOpnameBorrowDocument, len(borrowDocs))
	for _, d := range borrowDocs {
		borrowDocByAssetID[d.AssetID] = d
	}

	revisionScope, revisionActive, err := loadStockOpnameRevisionScope(transaction.ID)
	if err != nil {
		return nil, err
	}

	itemResponses := make([]dto.StockOpnameFlowItemResponse, len(itemRows))
	for i, item := range itemRows {
		itemResponses[i] = buildStockOpnameItemResponse(item, photoByAssetID[item.AssetID], borrowDocByAssetID[item.AssetID])
		itemResponses[i].NeedsRevision = revisionScope[item.AssetID]
	}

	return &dto.StockOpnameFlowDetailResponse{
		Transaction:  mapTransactionHeaderToResponse(*transaction),
		Items:        itemResponses,
		Stages:       mapTransactionStagesToResponse(stages),
		IsSubmissive: transaction.IsSubmissive,
		RevisionMode: revisionActive,
	}, nil
}

// buildStockOpnameItemResponse menggabungkan temuan auditor (item) dengan
// data sistem aktif saat ini (asset + active asset_value) untuk perbandingan.
// photo.ID == 0 berarti belum ada foto, borrowDoc.ID == 0 berarti belum ada
// dokumen peminjaman buat item ini.
func buildStockOpnameItemResponse(item stockOpnameItemRow, photo models.StockOpnameAssetPhoto, borrowDoc models.StockOpnameBorrowDocument) dto.StockOpnameFlowItemResponse {
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

	if photo.ID != 0 {
		resp.PhotoID = &photo.ID
		url := fmt.Sprintf("/api/v1/transactions/stock-opname/photo/%d/file", photo.ID)
		resp.PhotoURL = &url
		resp.PhotoCapturedAt = &photo.CapturedAt
	}

	if borrowDoc.ID != 0 {
		resp.BorrowDocumentID = &borrowDoc.ID
		url := fmt.Sprintf("/api/v1/transactions/stock-opname/borrow-document/%d/file", borrowDoc.ID)
		resp.BorrowDocumentURL = &url
		resp.BorrowDocumentFileName = &borrowDoc.FileName
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
//
// Physical status & condition SEKARANG master data (bukan map hardcode
// lagi) — codeToLabel/labelToCode di bawah dibangun on-the-fly dari
// loadStockOpnamePhysicalStatusMap()/loadStockOpnameConditionMap() supaya
// status/kondisi baru yang ditambah admin lewat
// /transactions/stock-opname/status-master/* otomatis kepake juga di excel.
// Asset status TETAP hardcode (di luar scope master data ini).
// ============================================================

func codeToLabelPhysicalStatus(m map[string]models.StockOpnamePhysicalStatusMaster) map[string]string {
	out := make(map[string]string, len(m))
	for code, row := range m {
		out[code] = row.Label
	}
	return out
}

func labelToCodePhysicalStatus(m map[string]models.StockOpnamePhysicalStatusMaster) map[string]string {
	out := make(map[string]string, len(m))
	for code, row := range m {
		out[strings.ToLower(strings.TrimSpace(row.Label))] = code
	}
	return out
}

func codeToLabelCondition(m map[string]models.StockOpnameConditionMaster) map[string]string {
	out := make(map[string]string, len(m))
	for code, row := range m {
		out[code] = row.Label
	}
	return out
}

func labelToCodeCondition(m map[string]models.StockOpnameConditionMaster) map[string]string {
	out := make(map[string]string, len(m))
	for code, row := range m {
		out[strings.ToLower(strings.TrimSpace(row.Label))] = code
	}
	return out
}

// sortedLabels: dipakai buat nyusun teks hint header excel (mis. "Status
// Fisik (Ada/Tidak Ada/Dipinjam)") supaya opsi baru dari master data ikut
// kelihatan tanpa perlu ubah kode tiap kali admin nambah status.
func sortedLabelsPhysicalStatus(m map[string]models.StockOpnamePhysicalStatusMaster) []string {
	rows := make([]models.StockOpnamePhysicalStatusMaster, 0, len(m))
	for _, row := range m {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	labels := make([]string, len(rows))
	for i, row := range rows {
		labels[i] = row.Label
	}
	return labels
}

func sortedLabelsCondition(m map[string]models.StockOpnameConditionMaster) []string {
	rows := make([]models.StockOpnameConditionMaster, 0, len(m))
	for _, row := range m {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	labels := make([]string, len(rows))
	for i, row := range rows {
		labels[i] = row.Label
	}
	return labels
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

	var items []models.StockOpnameDraftItem
	if err := config.DB.
		Preload("Asset.Category").
		Where("transaction_id = ?", transaction.ID).
		Order("asset_number ASC").
		Find(&items).Error; err != nil {
		return nil, "", err
	}

	// Selama DRAFT revisi, template cuma berisi asset yang dichecklist —
	// asset lain dikunci, jadi gak ada gunanya ikut diisi.
	revisionScope, revisionActive, err := loadStockOpnameRevisionScope(transaction.ID)
	if err != nil {
		return nil, "", err
	}
	if revisionActive {
		editable := items[:0]
		for _, item := range items {
			if revisionScope[item.AssetID] {
				editable = append(editable, item)
			}
		}
		items = editable
	}

	physicalMap, err := loadStockOpnamePhysicalStatusMap()
	if err != nil {
		return nil, "", err
	}
	conditionMap, err := loadStockOpnameConditionMap()
	if err != nil {
		return nil, "", err
	}
	physicalStatusLabel := codeToLabelPhysicalStatus(physicalMap)
	conditionLabel := codeToLabelCondition(conditionMap)

	f := excelize.NewFile()
	f.SetSheetName("Sheet1", stockOpnameTemplateSheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	headers := []string{
		"No", "Nomor Asset", "Nama Asset", "Kategori",
		fmt.Sprintf("Status Fisik (%s)", strings.Join(sortedLabelsPhysicalStatus(physicalMap), "/")),
		fmt.Sprintf("Kondisi (%s)", strings.Join(sortedLabelsCondition(conditionMap), "/")),
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
			physicalStatusLabel[item.PhysicalStatus],
			conditionLabel[item.Condition],
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
// EXCEL DOKUMENTASI FOTO (download setelah submit)
//
// Beda dari template di atas (dipakai buat ISI temuan pas DRAFT), file ini
// buat bukti dokumentasi fisik ke pihak lain (auditor dst) — daftar aset +
// foto bukti fisiknya dalam 1 file excel. Cuma bisa diunduh SETELAH submit,
// karena baru saat itu foto per asset dijamin lengkap & tervalidasi (lihat
// SubmitStockOpname bagian 7.9 di feature flow doc).
// ============================================================

const stockOpnameDocumentationSheet = "Dokumentasi"

func GenerateStockOpnameDocumentationExcel(userID, transactionNumber string) (*excelize.File, string, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, "", err
	}
	if transaction.CreatedBy != userID {
		return nil, "", errors.New("you can only download the documentation for your own stock opname")
	}
	if transaction.CurrentStage == models.StageDraft {
		return nil, "", errors.New("dokumentasi baru bisa diunduh setelah stock opname disubmit (foto bukti fisik belum tervalidasi lengkap)")
	}

	itemRows := getStockOpnameItemRows(transaction)

	conditionMap, err := loadStockOpnameConditionMap()
	if err != nil {
		return nil, "", err
	}
	conditionLabel := codeToLabelCondition(conditionMap)

	var photos []models.StockOpnameAssetPhoto
	config.DB.Where("transaction_id = ?", transaction.ID).Find(&photos)
	photoByAssetID := make(map[uint]models.StockOpnameAssetPhoto, len(photos))
	for _, p := range photos {
		photoByAssetID[p.AssetID] = p
	}

	// Semua aset di 1 stock opname selalu dari branch yang sama (lihat
	// bagian 3 di feature flow doc), jadi cukup resolve nama branch sekali
	// dari aset pertama yang punya branch_code.
	var branchName string
	for _, item := range itemRows {
		if item.Asset != nil && item.Asset.BranchCode != nil && *item.Asset.BranchCode != "" {
			if b, err := GetBranchByKode(*item.Asset.BranchCode); err == nil {
				branchName = b.BranchName
			}
			break
		}
	}

	f := excelize.NewFile()
	f.SetSheetName("Sheet1", stockOpnameDocumentationSheet)

	titleStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Underline: "single", Size: 12}})
	headerStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Border: []excelize.Border{
		{Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1},
		{Type: "left", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1},
	}})
	cellStyle, _ := f.NewStyle(&excelize.Style{Border: []excelize.Border{
		{Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1},
		{Type: "left", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1},
	}})

	titleCell := fmt.Sprintf("DOKUMENTASI ASSET STOCK OPNAME %s", transactionNumber)
	f.SetCellValue(stockOpnameDocumentationSheet, "A1", titleCell)
	f.MergeCell(stockOpnameDocumentationSheet, "A1", "J1")
	f.SetCellStyle(stockOpnameDocumentationSheet, "A1", "A1", titleStyle)

	headers := []string{"NO", "NO. ASET", "DESKRIPSI", "PLANT", "AREA", "SATUAN", "KONDISI", "LOKASI", "GROUPING", "PICTURE"}
	headerRow := 3
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRow)
		f.SetCellValue(stockOpnameDocumentationSheet, cell, h)
		f.SetCellStyle(stockOpnameDocumentationSheet, cell, cell, headerStyle)
	}

	colWidths := map[string]float64{
		"A": 6, "B": 18, "C": 28, "D": 10, "E": 16, "F": 10, "G": 14, "H": 16, "I": 14, "J": 28,
	}
	for col, w := range colWidths {
		f.SetColWidth(stockOpnameDocumentationSheet, col, col, w)
	}

	const documentationRowHeight = 110.0

	for i, item := range itemRows {
		row := headerRow + 1 + i
		f.SetRowHeight(stockOpnameDocumentationSheet, row, documentationRowHeight)

		var assetName, unitOfMeasure, location, grouping, plant string
		if item.Asset != nil {
			assetName = item.Asset.AssetName
			if item.Asset.UnitOfMeasure != nil {
				unitOfMeasure = *item.Asset.UnitOfMeasure
			}
			if item.Asset.Location != nil {
				location = *item.Asset.Location
			}
			if item.Asset.Grouping != nil {
				grouping = *item.Asset.Grouping
			}
			if item.Asset.BranchCode != nil {
				plant = *item.Asset.BranchCode
			}
		}

		values := []interface{}{
			i + 1, item.AssetNumber, assetName, plant, branchName,
			unitOfMeasure, conditionLabel[item.Condition], location, grouping,
		}
		firstCell, _ := excelize.CoordinatesToCellName(1, row)
		lastCell, _ := excelize.CoordinatesToCellName(len(headers), row)
		f.SetCellStyle(stockOpnameDocumentationSheet, firstCell, lastCell, cellStyle)
		for col, v := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			f.SetCellValue(stockOpnameDocumentationSheet, cell, v)
		}

		if photo, ok := photoByAssetID[item.AssetID]; ok {
			if picturePath, cleanup, err := resolveStockOpnamePicturePath(photo.FilePath); err == nil {
				pictureCell, _ := excelize.CoordinatesToCellName(len(headers), row)
				_ = f.AddPicture(stockOpnameDocumentationSheet, pictureCell, picturePath, &excelize.GraphicOptions{
					AutoFit:         true,
					LockAspectRatio: true,
					OffsetX:         4,
					OffsetY:         4,
				})
				if cleanup != nil {
					cleanup()
				}
			}
		}
	}

	safeName := strings.NewReplacer("/", "-", " ", "_").Replace(transactionNumber)
	filename := fmt.Sprintf("stock_opname_dokumentasi_%s.xlsx", safeName)
	return f, filename, nil
}

// resolveStockOpnamePicturePath menyiapkan path file gambar yang siap dipakai
// excelize.AddPicture. Excelize gak dukung embed WEBP langsung (cuma
// EMF/EMZ/GIF/ICO/JPEG/JPG/PNG/SVG/TIF/TIFF/WMF/WMZ) — padahal foto stock
// opname boleh diupload dalam format WEBP (lihat UploadStockOpnameAssetPhoto)
// — jadi buat foto WEBP kita decode lalu re-encode ke PNG sementara di temp
// dir. Format lain (jpg/jpeg/png) dipakai langsung dari path aslinya, gak
// ada file sementara yang perlu dibersihkan (cleanup == nil).
func resolveStockOpnamePicturePath(originalPath string) (path string, cleanup func(), err error) {
	ext := strings.ToLower(filepath.Ext(originalPath))
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
		if _, statErr := os.Stat(originalPath); statErr != nil {
			return "", nil, statErr
		}
		return originalPath, nil, nil
	}

	data, readErr := os.ReadFile(originalPath)
	if readErr != nil {
		return "", nil, readErr
	}
	img, _, decodeErr := image.Decode(bytes.NewReader(data))
	if decodeErr != nil {
		return "", nil, decodeErr
	}

	tmpFile, tmpErr := os.CreateTemp("", "stock_opname_photo_*.png")
	if tmpErr != nil {
		return "", nil, tmpErr
	}
	if encodeErr := png.Encode(tmpFile, img); encodeErr != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, encodeErr
	}
	tmpFile.Close()

	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }, nil
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

	var items []models.StockOpnameDraftItem
	if err := config.DB.Where("transaction_id = ?", transaction.ID).Find(&items).Error; err != nil {
		return nil, err
	}
	itemByAssetNumber := make(map[string]models.StockOpnameDraftItem, len(items))
	for _, item := range items {
		itemByAssetNumber[item.AssetNumber] = item
	}

	revisionScope, revisionActive, err := loadStockOpnameRevisionScope(transaction.ID)
	if err != nil {
		return nil, err
	}

	var borrowDocAssetIDs []uint
	config.DB.Model(&models.StockOpnameBorrowDocument{}).
		Where("transaction_id = ?", transaction.ID).
		Pluck("asset_id", &borrowDocAssetIDs)
	hasBorrowDoc := make(map[uint]bool, len(borrowDocAssetIDs))
	for _, id := range borrowDocAssetIDs {
		hasBorrowDoc[id] = true
	}

	soCfg, err := getOrCreateStockOpnameConfig()
	if err != nil {
		return nil, err
	}

	physicalMap, err := loadStockOpnamePhysicalStatusMap()
	if err != nil {
		return nil, err
	}
	conditionMap, err := loadStockOpnameConditionMap()
	if err != nil {
		return nil, err
	}
	physicalStatusValue := labelToCodePhysicalStatus(physicalMap)
	conditionValue := labelToCodeCondition(conditionMap)

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

		if revisionActive && !revisionScope[item.AssetID] {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: assetNumber, Message: errStockOpnameAssetNotInRevision,
			})
			continue
		}

		physicalLabelCell := strings.ToLower(cellAt(row, 4))
		conditionLabelCell := strings.ToLower(cellAt(row, 5))
		assetStatusLabel := strings.ToLower(cellAt(row, 6))
		notesCell := cellAt(row, 7)

		physicalStatus, ok := physicalStatusValue[physicalLabelCell]
		if !ok {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: assetNumber,
				Message: fmt.Sprintf("status fisik tidak valid: %q (harus salah satu: %s)",
					cellAt(row, 4), strings.Join(sortedLabelsPhysicalStatus(physicalMap), " / ")),
			})
			continue
		}

		condition, ok := conditionValue[conditionLabelCell]
		if !ok {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: assetNumber,
				Message: fmt.Sprintf("kondisi tidak valid: %q (harus salah satu: %s)",
					cellAt(row, 5), strings.Join(sortedLabelsCondition(conditionMap), " / ")),
			})
			continue
		}

		if err := validateFindingPhysicalConditionPair(physicalMap, conditionMap, physicalStatus, condition); err != nil {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: assetNumber,
				Message: err.Error(),
			})
			continue
		}

		if physicalMap[physicalStatus].RequiresBorrowDocument && soCfg.BorrowDocIsRequired && !hasBorrowDoc[item.AssetID] {
			response.Errors = append(response.Errors, dto.StockOpnameTemplateRowError{
				Row: rowNum, AssetNumber: assetNumber,
				Message: "dokumen peminjaman wajib diupload dulu lewat aplikasi sebelum status Dipinjam bisa disimpan",
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

		if err := config.DB.Model(&models.StockOpnameDraftItem{}).
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

// ============================================================
// FOTO BUKTI FISIK PER ASSET
//
// Wajib diisi sebelum submit (lihat SubmitStockOpname). Divalidasi 3
// lapis: ukuran file, tanggal modifikasi file (dikirim dari browser via
// File.lastModified, maks 10 hari dari sekarang), dan foto gak boleh
// dipakai dobel buat asset lain di stock opname yang sama (dicek lewat
// hash SHA-256 isi file). Satu asset cuma boleh punya 1 foto aktif —
// upload ulang menimpa foto sebelumnya.
//
// Sebelumnya divalidasi lewat EXIF DateTimeOriginal, tapi itu kelewat
// strict — foto yang diteruskan lewat WhatsApp/aplikasi chat lain (atau
// format selain JPEG) biasanya EXIF-nya udah hilang/kehapus padahal
// fotonya masih valid & baru. Pakai tanggal "modified" file jauh lebih
// longgar karena hampir semua file punya metadata ini.
// ============================================================

const (
	stockOpnamePhotoMaxSize    = 2 * 1024 * 1024 // 2MB — biar enteng buat testing/upload dari lapangan
	stockOpnamePhotoMaxAge     = 10 * 24 * time.Hour
	stockOpnamePhotoStorageDir = "uploads/stock_opname_photos"
)

// resolvePhotoModifiedAt nentuin tanggal "modified" foto dari nilai
// File.lastModified (epoch milliseconds) yang dikirim frontend. Kalau
// gak dikirim/gak valid, dianggap baru saja (tidak ditolak) — daripada
// bikin upload gagal gara-gara metadata yang emang gak selalu ada.
func resolvePhotoModifiedAt(clientModifiedAtMs int64) time.Time {
	if clientModifiedAtMs <= 0 {
		return time.Now()
	}
	return time.UnixMilli(clientModifiedAtMs)
}

func UploadStockOpnameAssetPhoto(userID string, transactionNumber string, assetID uint, file multipart.File, header *multipart.FileHeader, clientModifiedAtMs int64) (*dto.StockOpnameFlowDetailResponse, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageDraft {
		return nil, errors.New("can only upload photos on DRAFT stock opnames")
	}
	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only modify your own stock opname draft")
	}

	if header.Size > stockOpnamePhotoMaxSize {
		return nil, fmt.Errorf("ukuran foto maksimal 2MB (file ini %.2f MB)", float64(header.Size)/1024/1024)
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return nil, errors.New("foto harus format JPEG, PNG, atau WEBP")
	}

	var item models.StockOpnameDraftItem
	if err := config.DB.
		Where("transaction_id = ? AND asset_id = ?", transaction.ID, assetID).
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("asset not found in this stock opname")
		}
		return nil, err
	}

	if err := ensureStockOpnameAssetEditable(transaction.ID, assetID); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read uploaded file: %w", err)
	}
	if int64(len(data)) > stockOpnamePhotoMaxSize {
		return nil, fmt.Errorf("ukuran foto maksimal 2MB (file ini %.2f MB)", float64(len(data))/1024/1024)
	}

	capturedAt := resolvePhotoModifiedAt(clientModifiedAtMs)
	if time.Since(capturedAt) > stockOpnamePhotoMaxAge {
		return nil, fmt.Errorf("foto ini terakhir dimodifikasi tanggal %s, sudah lebih dari 10 hari dari sekarang — upload foto yang lebih baru", capturedAt.Format("02 Jan 2006"))
	}
	if capturedAt.After(time.Now().Add(1 * time.Hour)) {
		return nil, errors.New("tanggal modifikasi foto ada di masa depan — cek pengaturan tanggal/jam HP kamu")
	}

	hashBytes := sha256.Sum256(data)
	hashHex := hex.EncodeToString(hashBytes[:])

	var dupCount int64
	config.DB.Model(&models.StockOpnameAssetPhoto{}).
		Where("transaction_id = ? AND file_hash = ? AND asset_id != ?", transaction.ID, hashHex, assetID).
		Count(&dupCount)
	if dupCount > 0 {
		return nil, errors.New("foto ini sudah dipakai buat asset lain di stock opname ini — tiap asset wajib pakai foto yang berbeda")
	}

	dir := filepath.Join(stockOpnamePhotoStorageDir, sanitizePathSegment(transactionNumber))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to prepare storage directory: %w", err)
	}

	baseName := sanitizePathSegment(strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename)))
	fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), baseName, ext)
	fullPath := filepath.Join(dir, fileName)
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	var existing models.StockOpnameAssetPhoto
	err = config.DB.Where("transaction_id = ? AND asset_id = ?", transaction.ID, assetID).First(&existing).Error
	switch {
	case err == nil:
		oldPath := existing.FilePath
		if updateErr := config.DB.Model(&existing).Updates(map[string]interface{}{
			"file_name":   header.Filename,
			"file_path":   fullPath,
			"file_size":   int64(len(data)),
			"file_hash":   hashHex,
			"captured_at": capturedAt,
			"uploaded_by": userID,
		}).Error; updateErr != nil {
			os.Remove(fullPath)
			return nil, updateErr
		}
		if oldPath != "" && oldPath != fullPath {
			os.Remove(oldPath) // best-effort, file lama gak dipakai lagi
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		newPhoto := models.StockOpnameAssetPhoto{
			TransactionID: transaction.ID,
			AssetID:       assetID,
			FileName:      header.Filename,
			FilePath:      fullPath,
			FileSize:      int64(len(data)),
			FileHash:      hashHex,
			CapturedAt:    capturedAt,
			UploadedBy:    userID,
		}
		if createErr := config.DB.Create(&newPhoto).Error; createErr != nil {
			os.Remove(fullPath)
			return nil, createErr
		}
	default:
		os.Remove(fullPath)
		return nil, err
	}

	return GetStockOpnameFlowDetail(transactionNumber)
}

// GetStockOpnameAssetPhotoFilePath dipakai controller buat serve file foto.
func GetStockOpnameAssetPhotoFilePath(photoID uint) (string, string, error) {
	var photo models.StockOpnameAssetPhoto
	if err := config.DB.First(&photo, photoID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", errors.New("photo not found")
		}
		return "", "", err
	}
	if _, err := os.Stat(photo.FilePath); err != nil {
		return "", "", errors.New("photo file not found on disk")
	}
	return photo.FilePath, photo.FileName, nil
}

// ============================================================
// DOKUMEN PEMINJAMAN (PDF) PER ASSET — status fisik BORROWED
//
// Wajib diupload sebelum status BORROWED bisa disimpan di
// UpdateStockOpnameFinding/BulkUpdateStockOpnameFinding/template upload, dan
// wajib ada sebelum submit (lihat SubmitStockOpname). Satu asset cuma boleh
// punya 1 dokumen aktif — upload ulang menimpa dokumen sebelumnya.
//
// Belum ada config buat ukuran max/dsb (nyusul) — untuk sekarang dipakai
// batas tetap 5MB, format PDF-only.
// ============================================================

const (
	stockOpnameBorrowDocMaxSize    = 5 * 1024 * 1024 // 5MB
	stockOpnameBorrowDocStorageDir = "uploads/stock_opname_borrow_documents"
)

func UploadStockOpnameBorrowDocument(userID string, transactionNumber string, assetID uint, file multipart.File, header *multipart.FileHeader) (*dto.StockOpnameFlowDetailResponse, error) {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageDraft {
		return nil, errors.New("can only upload borrow documents on DRAFT stock opnames")
	}
	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only modify your own stock opname draft")
	}

	if header.Size > stockOpnameBorrowDocMaxSize {
		return nil, fmt.Errorf("ukuran dokumen maksimal 5MB (file ini %.2f MB)", float64(header.Size)/1024/1024)
	}

	soCfg, err := getOrCreateStockOpnameConfig()
	if err != nil {
		return nil, err
	}
	allowedExts := allowedBorrowDocExtensions(soCfg)
	if len(allowedExts) == 0 {
		return nil, errors.New("belum ada format file yang diizinkan buat dokumen peminjaman — atur dulu di halaman config stock opname")
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := false
	for _, e := range allowedExts {
		if ext == e {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("format dokumen peminjaman harus salah satu dari: %s", strings.Join(allowedExts, ", "))
	}

	var item models.StockOpnameDraftItem
	if err := config.DB.
		Where("transaction_id = ? AND asset_id = ?", transaction.ID, assetID).
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("asset not found in this stock opname")
		}
		return nil, err
	}

	if err := ensureStockOpnameAssetEditable(transaction.ID, assetID); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read uploaded file: %w", err)
	}
	if int64(len(data)) > stockOpnameBorrowDocMaxSize {
		return nil, fmt.Errorf("ukuran dokumen maksimal 5MB (file ini %.2f MB)", float64(len(data))/1024/1024)
	}

	dir := filepath.Join(stockOpnameBorrowDocStorageDir, sanitizePathSegment(transactionNumber))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to prepare storage directory: %w", err)
	}

	baseName := sanitizePathSegment(strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename)))
	fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), baseName, ext)
	fullPath := filepath.Join(dir, fileName)
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	var existing models.StockOpnameBorrowDocument
	err = config.DB.Where("transaction_id = ? AND asset_id = ?", transaction.ID, assetID).First(&existing).Error
	switch {
	case err == nil:
		oldPath := existing.FilePath
		if updateErr := config.DB.Model(&existing).Updates(map[string]interface{}{
			"file_name":   header.Filename,
			"file_path":   fullPath,
			"file_size":   int64(len(data)),
			"uploaded_by": userID,
		}).Error; updateErr != nil {
			os.Remove(fullPath)
			return nil, updateErr
		}
		if oldPath != "" && oldPath != fullPath {
			os.Remove(oldPath) // best-effort, file lama gak dipakai lagi
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		newDoc := models.StockOpnameBorrowDocument{
			TransactionID: transaction.ID,
			AssetID:       assetID,
			FileName:      header.Filename,
			FilePath:      fullPath,
			FileSize:      int64(len(data)),
			UploadedBy:    userID,
		}
		if createErr := config.DB.Create(&newDoc).Error; createErr != nil {
			os.Remove(fullPath)
			return nil, createErr
		}
	default:
		os.Remove(fullPath)
		return nil, err
	}

	return GetStockOpnameFlowDetail(transactionNumber)
}

// GetStockOpnameBorrowDocumentFilePath dipakai controller buat serve file dokumen.
func GetStockOpnameBorrowDocumentFilePath(docID uint) (string, string, error) {
	var doc models.StockOpnameBorrowDocument
	if err := config.DB.First(&doc, docID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", errors.New("borrow document not found")
		}
		return "", "", err
	}
	if _, err := os.Stat(doc.FilePath); err != nil {
		return "", "", errors.New("borrow document file not found on disk")
	}
	return doc.FilePath, doc.FileName, nil
}
