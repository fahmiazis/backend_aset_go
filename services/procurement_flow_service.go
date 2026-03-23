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

	if err := validateStageTransition(transaction.CurrentStage, models.StageAssetVerification); err != nil {
		return nil, err
	}

	// Cek attachment DRAFT sebelum submit
	creatorHomebase, err := GetUserActiveHomebase(transaction.CreatedBy)
	if err == nil {
		if err := checkAttachmentCanProceed(transactionNumber, TxProcurement, models.StageDraft, creatorHomebase.Branch.BranchCode); err != nil {
			return nil, err
		}
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
	if err := updateTransactionStage(tx, transaction, models.StageAssetVerification); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageAssetVerification,
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

	if transaction.CurrentStage != models.StageAssetVerification {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageAssetVerification)
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

	// Cek attachment VERIFIKASI_ASET sebelum lanjut ke APPROVAL
	if err := checkAttachmentCanProceed(transactionNumber, TxProcurement, models.StageAssetVerification, branchCode); err != nil {
		return nil, err
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
	// Cari berdasarkan branch creator dulu, fallback ke ALL
	creatorHomebase, homebaseErr := GetUserActiveHomebase(transaction.CreatedBy)
	branchCode := "ALL"
	if homebaseErr == nil {
		branchCode = creatorHomebase.Branch.BranchCode
	}

	flow, err := GetApprovalFlowByCodeAndBranch("PROCUREMENT_APPROVAL", branchCode)
	if err != nil {
		return fmt.Errorf("approval flow PROCUREMENT_APPROVAL not found for branch %s or ALL, please configure it first", branchCode)
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
	if err := updateTransactionStage(tx, transaction, models.StageProcessBudget); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageProcessBudget,
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

	if transaction.CurrentStage != models.StageProcessBudget {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageProcessBudget)
	}

	// Cek attachment PROSES_BUDGET sebelum generate IO dan lanjut ke EKSEKUSI_ASET
	creatorHomebaseBudget, err := GetUserActiveHomebase(transaction.CreatedBy)
	if err == nil {
		if err := checkAttachmentCanProceed(transactionNumber, TxProcurement, models.StageProcessBudget, creatorHomebaseBudget.Branch.BranchCode); err != nil {
			return nil, err
		}
	}

	// Kumpulkan semua branch unik dari items dan details
	branchSet := make(map[string]bool)

	var procurements []models.TransactionProcurement
	config.DB.
		Preload("TransactionProcurementDetails").
		Where("transaction_id = ?", transaction.ID).
		Find(&procurements)

	for _, proc := range procurements {
		if len(proc.TransactionProcurementDetails) > 0 {
			for _, d := range proc.TransactionProcurementDetails {
				branchSet[d.BranchCode] = true
			}
		} else {
			branchSet[proc.BranchCode] = true
		}
	}

	now := time.Now()

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Generate IO number per branch unik → simpan ke transaction_io_numbers
	// IO number pertama juga disimpan di transactions.io_number sebagai referensi
	firstIONumber := ""
	for branchCode := range branchSet {
		ioNumber, err := GenerateIONumber(tx, branchCode)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to generate IO number for branch %s: %w", branchCode, err)
		}

		ioRecord := models.TransactionIONumber{
			TransactionID:     transaction.ID,
			TransactionNumber: transactionNumber,
			BranchCode:        branchCode,
			IONumber:          ioNumber,
			ProcessedBy:       userID,
			ProcessedAt:       now,
		}

		if err := tx.Create(&ioRecord).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to save IO number for branch %s: %w", branchCode, err)
		}

		if firstIONumber == "" {
			firstIONumber = ioNumber
		}
	}

	// Simpan IO number pertama ke transactions.io_number sebagai referensi utama
	if err := tx.Model(transaction).Update("io_number", firstIONumber).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageExecuteAsset); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageExecuteAsset,
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
// Asset status = PENDING_RECEIPT
// ============================================================

func ExecuteProcurementAsset(userID string, transactionNumber string, req dto.ExecuteAssetRequest) (*dto.ProcurementDetailWithStageResponse, error) {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageExecuteAsset {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageExecuteAsset)
	}

	// Cek attachment EKSEKUSI_ASET sebelum generate asset
	creatorHomebaseExec, err := GetUserActiveHomebase(transaction.CreatedBy)
	if err == nil {
		if err := checkAttachmentCanProceed(transactionNumber, TxProcurement, models.StageExecuteAsset, creatorHomebaseExec.Branch.BranchCode); err != nil {
			return nil, err
		}
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

	// Load IO numbers per branch untuk transaksi ini
	var ioNumbers []models.TransactionIONumber
	config.DB.Where("transaction_id = ?", transaction.ID).Find(&ioNumbers)
	ioNumberMap := make(map[string]string) // key: branch_code, value: io_number
	for _, io := range ioNumbers {
		ioNumberMap[io.BranchCode] = io.IONumber
	}

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
				assetIONum := ioNumberMap[branchCode]
				asset := models.Asset{
					AssetNumber: assetNumber,
					AssetName:   proc.ItemName,
					CategoryID:  &category.ID,
					BranchCode:  &branchCode,
					IONumber:    &assetIONum, // *string
					AssetStatus: models.AssetStatusPendingReceipt,
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
				// Ambil IO number sesuai branch asset
				ioNum := ioNumberMap[branchCode]
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
					IONumber:                 ioNum, // IO number sesuai branch
					Status:                   "DRAFT",
					CreatedBy:                userID,
				}

				if err := tx.Create(&acquisition).Error; err != nil {
					tx.Rollback()
					return nil, fmt.Errorf("failed to create asset acquisition: %w", err)
				}

				// AssetValue dibuat saat GR, bukan saat eksekusi
				// karena nilai asset mulai berlaku setelah barang diterima
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
// Setelah semua item GR → status transaksi = FINISHED
// ============================================================

func CreateAssetGR(userID string, transactionNumber string, req dto.CreateGRRequest) (*dto.AssetGRResponse, error) {
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageGR {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageGR)
	}

	// Validasi asset ada
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

	if asset.AssetStatus != models.AssetStatusPendingReceipt {
		return nil, fmt.Errorf("asset %s has already been received", req.AssetNumber)
	}

	// Validasi branch — branch asset harus sama dengan homebase user
	// Hanya user yang homebase-nya di branch tujuan asset yang bisa GR
	if asset.BranchCode == nil {
		return nil, errors.New("asset has no branch assigned")
	}

	homebase, err := GetUserActiveHomebase(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user homebase: %w", err)
	}

	if homebase.Branch.BranchCode != *asset.BranchCode {
		return nil, fmt.Errorf("you can only do GR for assets in your homebase branch (%s)", homebase.Branch.BranchCode)
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
		BranchCode:        *asset.BranchCode,
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

	// Update asset_acquisition status → APPROVED
	if err := tx.Model(&models.AssetAcquisition{}).
		Where("asset_id = ?", asset.ID).
		Update("status", "APPROVED").Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update acquisition status: %w", err)
	}

	// Create initial AssetValue — effective_date = tanggal GR
	// book_value = acquisition_value = harga beli, accumulated_depreciation = 0
	var acquisition models.AssetAcquisition
	config.DB.Where("asset_id = ?", asset.ID).First(&acquisition)

	acquisitionValue := acquisition.AcquisitionValue
	assetValue := models.AssetValue{
		AssetID:                 asset.ID,
		EffectiveDate:           grDate,
		BookValue:               acquisitionValue,
		AcquisitionValue:        acquisitionValue,
		AccumulatedDepreciation: 0,
		AssetStatus:             &[]string{models.AssetStatusAvailable}[0],
		IsActive:                true,
	}

	if err := tx.Create(&assetValue).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create asset value: %w", err)
	}

	// Cek apakah semua asset di transaksi ini sudah GR
	// Pakai config.DB (bukan tx) untuk count yang akurat karena GR baru
	// sudah di-commit via tx.Create di atas
	var totalAssets int64
	config.DB.Model(&models.AssetAcquisition{}).
		Where("transaction_id = ?", transaction.ID).
		Count(&totalAssets)

	// Count dari DB setelah commit GR baru
	var totalGR int64
	config.DB.Model(&models.AssetGR{}).
		Where("transaction_id = ?", transaction.ID).
		Count(&totalGR)

	// +1 karena GR baru belum ter-commit ke DB (masih dalam tx)
	if totalAssets > 0 && totalGR+1 >= totalAssets {
		// Semua sudah GR → update stage ke FINISHED
		fromStage := transaction.CurrentStage
		if err := updateTransactionStage(tx, transaction, models.StageFinished); err != nil {
			tx.Rollback()
			return nil, err
		}

		notes := "All assets received"
		if err := recordStage(tx, transaction.ID, transactionNumber,
			fromStage, models.StageFinished,
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
// Bisa dilakukan di semua stage kecuali DRAFT & FINISHED
// ============================================================

func RejectProcurement(userID string, transactionNumber string, req dto.RejectProcurementRequest) (*dto.ProcurementDetailWithStageResponse, error) {
	transaction, err := getProcurementTransaction(transactionNumber)
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

	// Mark reservoir as expired
	MarkTransactionAsExpired(transactionNumber)

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetProcurementDetailWithStage(transactionNumber)
}

// ============================================================
// REVISI
// Bisa dilakukan di semua stage kecuali DRAFT, FINISHED, REJECTED
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
		models.StageFinished,
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
