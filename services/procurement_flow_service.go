package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ============================================================
// STAGE 1: SUBMIT
// DRAFT → VERIFIKASI_ASET
// ============================================================

func SubmitProcurement(userID string, transactionNumber string, req dto.SubmitProcurementRequest) (*dto.ProcurementDetailWithStageResponse, error) {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only submit your own transactions")
	}

	if err := validateStageTransition(transaction.CurrentStage, models.StageVerifikasiAset); err != nil {
		return nil, err
	}

	// Pastikan ada items yang aktif
	var itemCount int64
	config.DB.Model(&models.TransactionProcurement{}).
		Where("transaction_id = ?", transaction.ID).
		Count(&itemCount)
	if itemCount == 0 {
		return nil, errors.New("cannot submit procurement with no items")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageVerifikasiAset); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageVerifikasiAset,
		models.ActionSubmit, userID, nil, req.Notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetProcurementDetailWithStage(transactionNumber)
}

// ============================================================
// STAGE 2: VERIFIKASI ASET
// VERIFIKASI_ASET → APPROVAL
// PIC Asset marking setiap item: ASSET atau NON_ASSET
// Syarat: branch PIC Asset harus sesuai branch transaksi
// ============================================================

func VerifyProcurement(userID string, branchCode string, transactionNumber string, req dto.VerifyProcurementRequest) (*dto.ProcurementDetailWithStageResponse, error) {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageVerifikasiAset {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageVerifikasiAset)
	}

	// Get semua items di transaksi ini
	var procurements []models.TransactionProcurement
	if err := config.DB.Where("transaction_id = ?", transaction.ID).Find(&procurements).Error; err != nil {
		return nil, err
	}

	// Validasi semua item harus diverifikasi
	procMap := make(map[uint]bool)
	for _, p := range procurements {
		procMap[p.ID] = true
	}
	for _, item := range req.Items {
		if !procMap[item.TransactionProcurementID] {
			return nil, fmt.Errorf("item %d does not belong to this transaction", item.TransactionProcurementID)
		}
	}
	if len(req.Items) != len(procurements) {
		return nil, errors.New("all items must be verified")
	}

	// Cek apakah semua NON_ASSET — kalau iya wajib reject
	allNonAsset := true
	for _, item := range req.Items {
		if item.ItemType == models.ItemTypeAsset {
			allNonAsset = false
			break
		}
	}
	if allNonAsset {
		return nil, errors.New("all items are NON_ASSET, transaction must be rejected")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()

	for _, item := range req.Items {
		verif := models.TransactionItemVerification{
			TransactionID:            transaction.ID,
			TransactionProcurementID: item.TransactionProcurementID,
			ItemType:                 item.ItemType,
			IsActive:                 item.ItemType == models.ItemTypeAsset, // NON_ASSET langsung is_active = false
			VerifiedBy:               userID,
			VerifiedAt:               now,
			Notes:                    item.Notes,
		}

		// Upsert — kalau sudah ada (revisi), update
		if err := tx.Where("transaction_procurement_id = ?", item.TransactionProcurementID).
			Assign(verif).
			FirstOrCreate(&verif).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageApproval); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageApproval,
		models.ActionVerify, userID, nil, req.Notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetProcurementDetailWithStage(transactionNumber)
}

// ============================================================
// STAGE 3: APPROVAL
// APPROVAL → PROSES_BUDGET
// Trigger TransactionApproval system
// Flow otomatis dicari berdasarkan flow_code PROCUREMENT_APPROVAL
// ============================================================

func InitiateProcurementApproval(userID string, transactionNumber string, req dto.InitiateApprovalRequest) error {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return err
	}

	if transaction.CurrentStage != models.StageApproval {
		return fmt.Errorf("transaction is not in %s stage", models.StageApproval)
	}

	// Auto-lookup flow by code PROCUREMENT_APPROVAL
	// Tidak perlu input flow_id manual dari user
	flow, err := GetApprovalFlowByCode("PROCUREMENT_APPROVAL")
	if err != nil {
		return fmt.Errorf("approval flow PROCUREMENT_APPROVAL not found, please configure it first")
	}

	if !flow.IsActive {
		return errors.New("approval flow PROCUREMENT_APPROVAL is inactive")
	}

	approvalReq := dto.CreateTransactionApprovalRequest{
		FlowID:            flow.ID,
		TransactionNumber: transactionNumber,
		TransactionType:   TxProcurement,
		Metadata:          req.Metadata,
	}

	return InitiateTransactionApproval(approvalReq)
}

// CompleteProcurementApproval dipanggil setelah semua approval step approved
// Transisi APPROVAL → PROSES_BUDGET
func CompleteProcurementApproval(userID string, transactionNumber string) (*dto.ProcurementDetailWithStageResponse, error) {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageApproval {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageApproval)
	}

	// Cek semua approval sudah approved
	approvalSummary, err := GetTransactionApprovalStatus(transactionNumber, TxProcurement)
	if err != nil {
		return nil, err
	}
	if approvalSummary.Status != "approved" {
		return nil, errors.New("not all approval steps have been approved")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageProsesBudget); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageProsesBudget,
		models.ActionApprove, userID, nil, nil); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetProcurementDetailWithStage(transactionNumber)
}

// ============================================================
// STAGE 4: PROSES BUDGET
// PROSES_BUDGET → EKSEKUSI_ASET
// PIC Budget generate nomor IO berdasarkan branch
// ============================================================

func ProcessProcurementBudget(userID string, transactionNumber string, req dto.ProcessBudgetRequest) (*dto.ProcurementDetailWithStageResponse, error) {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageProsesBudget {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageProsesBudget)
	}

	// Ambil branch_code dari created_by user homebase
	homebase, err := GetUserActiveHomebase(transaction.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch for IO number generation: %w", err)
	}
	branchCode := homebase.Branch.BranchCode

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Generate nomor IO
	ioNumber, err := GenerateIONumber(tx, branchCode)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to generate IO number: %w", err)
	}

	// Simpan io_number ke transaction
	if err := tx.Model(transaction).Update("io_number", ioNumber).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageEksekusiAset); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageEksekusiAset,
		models.ActionProcessBudget, userID, nil, req.Notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetProcurementDetailWithStage(transactionNumber)
}

// ============================================================
// STAGE 5: EKSEKUSI ASET
// EKSEKUSI_ASET → GR
// PIC Asset generate nomor asset & create asset records
// Asset status = BELUM_GR
// ============================================================

func ExecuteProcurementAsset(userID string, transactionNumber string, req dto.ExecuteAssetRequest) (*dto.ProcurementDetailWithStageResponse, error) {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageEksekusiAset {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageEksekusiAset)
	}

	// Ambil hanya items yang verified sebagai ASSET (is_active = true)
	var verifiedItems []models.TransactionItemVerification
	if err := config.DB.
		Preload("TransactionProcurement.Category").
		Preload("TransactionProcurement.TransactionProcurementDetails").
		Where("transaction_id = ? AND item_type = ? AND is_active = ?",
			transaction.ID, models.ItemTypeAsset, true).
		Find(&verifiedItems).Error; err != nil {
		return nil, err
	}

	if len(verifiedItems) == 0 {
		return nil, errors.New("no verified ASSET items found")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, verif := range verifiedItems {
		if verif.TransactionProcurement == nil || verif.TransactionProcurement.Category == nil {
			tx.Rollback()
			return nil, fmt.Errorf("procurement or category data missing for item %d", verif.TransactionProcurementID)
		}

		proc := verif.TransactionProcurement
		category := proc.Category

		// Tentukan branch per asset:
		// - Kalau ada details → ikut branch per detail (split sesuai qty detail)
		// - Kalau tidak ada details → semua asset pakai branch item parent
		type branchQty struct {
			BranchCode string
			Quantity   int
		}

		var branchList []branchQty

		// Load details dari preload
		details := proc.TransactionProcurementDetails

		if len(details) > 0 {
			for _, d := range details {
				branchList = append(branchList, branchQty{
					BranchCode: d.BranchCode,
					Quantity:   d.Quantity,
				})
			}
		} else {
			// Tidak ada details → semua pakai branch item parent
			branchList = append(branchList, branchQty{
				BranchCode: proc.BranchCode,
				Quantity:   proc.Quantity,
			})
		}

		// Generate asset per branch sesuai quantity
		for _, bq := range branchList {
			for i := 0; i < bq.Quantity; i++ {
				assetNumber, err := GenerateAssetNumber(tx, category.CategoryCode)
				if err != nil {
					tx.Rollback()
					return nil, fmt.Errorf("failed to generate asset number: %w", err)
				}

				branchCode := bq.BranchCode
				asset := models.Asset{
					AssetNumber: assetNumber,
					AssetName:   proc.ItemName,
					CategoryID:  &category.ID,
					BranchCode:  &branchCode,
					AssetStatus: models.AssetStatusBelumGR,
				}

				if err := tx.Create(&asset).Error; err != nil {
					tx.Rollback()
					return nil, fmt.Errorf("failed to create asset record: %w", err)
				}

				documentNumber, err := GenerateDocumentNumber(tx)
				if err != nil {
					tx.Rollback()
					return nil, fmt.Errorf("failed to generate document number: %w", err)
				}

				assetID := asset.ID
				procID := proc.ID
				acquisition := models.AssetAcquisition{
					DocumentNumber:           documentNumber,
					AssetID:                  &assetID,
					AssetNumber:              assetNumber,
					AssetName:                proc.ItemName,
					TransactionID:            &transaction.ID,
					TransactionNumber:        transaction.TransactionNumber,
					TransactionProcurementID: &procID,
					AcquisitionValue:         proc.UnitPrice,
					CategoryID:               &category.ID,
					BranchCode:               branchCode,
					Status:                   "DRAFT",
					CreatedBy:                userID,
				}

				if err := tx.Create(&acquisition).Error; err != nil {
					tx.Rollback()
					return nil, fmt.Errorf("failed to create asset acquisition: %w", err)
				}

				// TODO: Create initial asset value setelah model AssetValue dikonfirmasi
				// assetValue := models.AssetValue{...}
			}
		}
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageGR); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageGR,
		models.ActionExecute, userID, nil, req.Notes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetProcurementDetailWithStage(transactionNumber)
}

// ============================================================
// STAGE 6: GOOD RECEIPT (GR)
// GR per item — user branch tujuan
// Setelah semua item GR → status transaksi = SELESAI
// ============================================================

func CreateAssetGR(userID string, userBranchCode string, transactionNumber string, req dto.CreateGRRequest) (*dto.AssetGRResponse, error) {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageGR {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageGR)
	}

	// Validasi asset ada dan milik transaksi ini
	var asset models.Asset
	if err := config.DB.First(&asset, req.AssetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("asset not found")
		}
		return nil, err
	}

	if asset.AssetNumber != req.AssetNumber {
		return nil, errors.New("asset number mismatch")
	}

	if asset.AssetStatus != models.AssetStatusBelumGR {
		return nil, fmt.Errorf("asset %s has already been received", req.AssetNumber)
	}

	// Validasi user dari branch yang sesuai dengan asset branch
	if asset.BranchCode == nil || *asset.BranchCode != userBranchCode {
		return nil, errors.New("you can only do GR for assets in your branch")
	}

	grDate, err := time.Parse("2006-01-02", req.GRDate)
	if err != nil {
		return nil, errors.New("invalid gr_date format, use YYYY-MM-DD")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()

	// Create GR record
	gr := models.AssetGR{
		TransactionID:     transaction.ID,
		TransactionNumber: transactionNumber,
		AssetID:           asset.ID,
		AssetNumber:       asset.AssetNumber,
		BranchCode:        userBranchCode,
		GRDate:            grDate,
		GRBy:              userID,
		GRAt:              now,
		Notes:             req.Notes,
	}

	if err := tx.Create(&gr).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Update asset status → AVAILABLE
	if err := tx.Model(&asset).Update("asset_status", models.AssetStatusAvailable).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Cek apakah semua asset di transaksi ini sudah GR
	// Ambil total asset dari transaksi ini
	var totalAssets int64
	tx.Model(&models.AssetAcquisition{}).
		Where("transaction_id = ?", transaction.ID).
		Count(&totalAssets)

	var totalGR int64
	tx.Model(&models.AssetGR{}).
		Where("transaction_id = ?", transaction.ID).
		Count(&totalGR)

	// +1 karena GR yang baru saja dibuat belum ter-count (masih dalam tx)
	if totalGR+1 >= totalAssets {
		// Semua sudah GR → update stage ke SELESAI
		fromStage := transaction.CurrentStage
		if err := updateTransactionStage(tx, transaction, models.StageSelesai); err != nil {
			tx.Rollback()
			return nil, err
		}

		notes := "All assets received"
		if err := recordStage(tx, transaction.ID, transactionNumber,
			fromStage, models.StageSelesai,
			models.ActionGR, userID, nil, &notes); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	response := mapAssetGRToResponse(gr)
	return &response, nil
}

// ============================================================
// REJECT
// Bisa dilakukan di semua stage kecuali DRAFT & SELESAI
// ============================================================

func RejectProcurement(userID string, transactionNumber string, req dto.RejectProcurementRequest) (*dto.ProcurementDetailWithStageResponse, error) {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage == models.StageDraft ||
		transaction.CurrentStage == models.StageSelesai ||
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

	// Mark reservoir as expired
	MarkTransactionAsExpired(transactionNumber)

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetProcurementDetailWithStage(transactionNumber)
}

// ============================================================
// REVISI
// Bisa dilakukan di semua stage kecuali DRAFT, SELESAI, REJECTED
// Kalau stage = APPROVAL → ulang dari APPROVAL
// Kalau stage lain → langsung ke stage tersebut
// ============================================================

func ReviseProcurement(userID string, transactionNumber string, req dto.ReviseProcurementRequest) (*dto.ProcurementDetailWithStageResponse, error) {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	// Validasi stage yang boleh direvisi
	nonRevisableStages := []string{
		models.StageDraft,
		models.StageSelesai,
		models.StageRejected,
	}
	for _, s := range nonRevisableStages {
		if transaction.CurrentStage == s {
			return nil, fmt.Errorf("cannot revise transaction in %s stage", transaction.CurrentStage)
		}
	}

	// Hanya creator yang bisa revisi
	if transaction.CreatedBy != userID {
		return nil, errors.New("only the creator can revise this transaction")
	}

	// Tentukan target stage setelah revisi
	targetStage := transaction.CurrentStage
	if transaction.CurrentStage == models.StageApproval {
		// Kalau sedang di APPROVAL, revisi → ulang dari APPROVAL
		targetStage = models.StageApproval
	}
	// Stage lain → tetap di stage yang sama (akan diproses ulang)

	homebase, err := GetUserActiveHomebase(userID)
	if err != nil {
		return nil, err
	}

	// Validasi semua branch code sebelum mulai transaksi DB
	isHO := homebase.Branch.BranchType == "HO"
	for _, item := range req.Items {
		if item.BranchCode != nil && *item.BranchCode != "" {
			if !isHO {
				if *item.BranchCode != homebase.Branch.BranchCode {
					return nil, fmt.Errorf("item '%s': branch_code must match your homebase (%s)", item.ItemName, homebase.Branch.BranchCode)
				}
			} else {
				if err := validateBranchExists(*item.BranchCode); err != nil {
					return nil, fmt.Errorf("item '%s': %w", item.ItemName, err)
				}
			}
		}
		for _, detail := range item.Details {
			if err := validateBranchExists(detail.BranchCode); err != nil {
				return nil, fmt.Errorf("item '%s' detail: %w", item.ItemName, err)
			}
			if !isHO && detail.BranchCode != homebase.Branch.BranchCode {
				return nil, fmt.Errorf("item '%s' detail: only HO users can add details for other branches (your homebase: %s)", item.ItemName, homebase.Branch.BranchCode)
			}
		}
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Hapus items lama
	if err := tx.Where("transaction_id = ?", transaction.ID).
		Delete(&models.TransactionProcurement{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Hapus verifikasi lama jika ada
	tx.Where("transaction_id = ?", transaction.ID).
		Delete(&models.TransactionItemVerification{})

	// Update transaction header jika ada perubahan
	updates := map[string]interface{}{}
	if req.TransactionDate != nil {
		date, err := time.Parse("2006-01-02", *req.TransactionDate)
		if err != nil {
			tx.Rollback()
			return nil, errors.New("invalid transaction_date format, use YYYY-MM-DD")
		}
		updates["transaction_date"] = date
	}
	if req.Notes != nil {
		updates["notes"] = req.Notes
	}
	if len(updates) > 0 {
		if err := tx.Model(transaction).Updates(updates).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Recreate items
	for _, item := range req.Items {
		var category models.AssetCategory
		if err := tx.First(&category, item.CategoryID).Error; err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("category not found: %d", item.CategoryID)
			}
			return nil, err
		}

		totalPrice := float64(item.Quantity) * item.UnitPrice
		branchCode := homebase.Branch.BranchCode
		if item.BranchCode != nil && *item.BranchCode != "" {
			branchCode = *item.BranchCode
		}

		procurement := models.TransactionProcurement{
			TransactionID:     transaction.ID,
			TransactionNumber: transaction.TransactionNumber,
			ItemName:          item.ItemName,
			CategoryID:        &category.ID,
			Quantity:          item.Quantity,
			UnitPrice:         item.UnitPrice,
			TotalPrice:        totalPrice,
			BranchCode:        branchCode,
			Notes:             item.Notes,
		}

		if err := tx.Create(&procurement).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		if len(item.Details) > 0 {
			totalDetailQty := 0
			for _, detail := range item.Details {
				procDetail := models.TransactionProcurementDetail{
					TransactionProcurementID: procurement.ID,
					BranchCode:               detail.BranchCode,
					Quantity:                 detail.Quantity,
					RequesterName:            detail.RequesterName,
					Notes:                    detail.Notes,
				}
				if err := tx.Create(&procDetail).Error; err != nil {
					tx.Rollback()
					return nil, err
				}
				totalDetailQty += detail.Quantity
			}

			if totalDetailQty != item.Quantity {
				tx.Rollback()
				return nil, fmt.Errorf("total detail quantity must match item quantity for item: %s", item.ItemName)
			}
		}
	}

	// Record stage revisi
	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, targetStage); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, targetStage,
		models.ActionRevise, userID, nil, &req.RevisionNotes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetProcurementDetailWithStage(transactionNumber)
}
