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
	"strings"
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

// GetCreatorBranchCode — ambil branch code homebase creator transaksi.
// Fallback ke "ALL" kalau homebase tidak ditemukan, supaya tetap match config global.
func GetCreatorBranchCode(createdBy string) string {
	homebase, err := GetUserActiveHomebase(createdBy)
	if err != nil || homebase == nil {
		return "ALL"
	}
	return homebase.Branch.BranchCode
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

	// Saat transaksi sedang dalam revisi, lingkupnya sudah ditentukan approver —
	// menambah aset baru akan keluar dari lingkup itu.
	if disposalHasPendingRevision(transaction.ID) {
		return nil, errors.New("transaction is under revision, you can only fix the assets marked by the approver")
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

	// Validasi branch code asset harus sama dengan homebase creator
	creatorHomebase, err := GetUserActiveHomebase(transaction.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to get creator homebase: %w", err)
	}
	if asset.BranchCode == nil || *asset.BranchCode != creatorHomebase.Branch.BranchCode {
		return nil, fmt.Errorf("asset %s is not in your branch (%s)", req.AssetNumber, creatorHomebase.Branch.BranchCode)
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

	// Selama revisi, aset yang tidak ditandai approver tidak boleh diutak-atik
	if err := assertAssetRevisable(transaction.ID, disposalAsset.ID); err != nil {
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
// DRAFT → PURCHASING (SELL) / APPROVAL_REQUEST (DISPOSE)
// ============================================================

// resolveDisposalApprovalFlow mencari flow approval milik cabang creator.
// Dipakai sebagai pre-check SEBELUM stage dipindahkan, supaya transaksi tidak
// terlanjur pindah stage lalu mentok karena flow-nya belum dikonfigurasi.
func resolveDisposalApprovalFlow(transaction *models.Transaction, flowCode string) (*dto.ApprovalFlowResponse, error) {
	branchCode := GetCreatorBranchCode(transaction.CreatedBy)

	flow, err := GetApprovalFlowByCodeAndBranch(flowCode, branchCode)
	if err != nil {
		return nil, fmt.Errorf("approval flow %s not found for branch %s or ALL, please configure it first", flowCode, branchCode)
	}

	if !flow.IsActive {
		return nil, fmt.Errorf("approval flow %s is inactive", flowCode)
	}

	if len(flow.FlowSteps) == 0 {
		return nil, fmt.Errorf("approval flow %s has no steps configured", flowCode)
	}

	return flow, nil
}

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

	// Cek attachment DRAFT sudah semua diupload (cukup PENDING, tidak boleh REJECTED/missing)
	allOK, err := checkAllDisposalAttachments(transactionNumber, transaction.ID,
		models.StageDisposalDraft, GetCreatorBranchCode(transaction.CreatedBy), false)
	if err != nil {
		return nil, err
	}
	if !allOK {
		return nil, errors.New("not all required draft documents are uploaded for all assets")
	}

	nextStage, err := nextStageForDisposal(*transaction.DisposalType, transaction.CurrentStage)
	if err != nil {
		return nil, err
	}

	// DISPOSE lompat dari DRAFT langsung ke APPROVAL_REQUEST (SELL mampir ke
	// PURCHASING dulu). Kalau tujuannya APPROVAL_REQUEST, flow-nya dipastikan
	// siap DULU — kalau belum, submit dibatalkan dengan pesan jelas daripada
	// transaksi pindah stage tapi tidak ada approval yang terbentuk.
	initiateApproval := nextStage == models.StageDisposalApprovalRequest
	if initiateApproval {
		if _, err := resolveDisposalApprovalFlow(transaction, models.FlowDisposalApprovalRequest); err != nil {
			return nil, err
		}
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

	// Revisi dianggap selesai begitu transaksi disubmit ulang
	if err := tx.Model(&models.TransactionDisposalAsset{}).
		Where("transaction_id = ?", transaction.ID).
		Updates(map[string]interface{}{
			"needs_revision": false,
			"revision_notes": nil,
		}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Langsung bentuk baris approval — creator tidak perlu menekan tombol
	// "ajukan" terpisah (dan tidak perlu permission manage_approval).
	if initiateApproval {
		if err := InitiateDisposalApprovalRequest(userID, transactionNumber,
			dto.InitiateDisposalApprovalRequest{}); err != nil {
			return nil, fmt.Errorf("disposal submitted but approval initiation failed: %w", err)
		}
	}

	return GetDisposalDetail(transactionNumber)
}

// ============================================================
// PURCHASING — set sale_value per asset (SELL only)
// PURCHASING → APPROVAL_REQUEST (setelah purchasing confirm)
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

	branchCode := GetCreatorBranchCode(transaction.CreatedBy)

	// Cek attachment DRAFT sudah semua APPROVED
	draftOK, err := checkAllDisposalAttachments(transactionNumber, transaction.ID, models.StageDisposalDraft, branchCode, true)
	if err != nil {
		return nil, err
	}
	if !draftOK {
		return nil, errors.New("not all required draft documents are approved for all assets")
	}

	// Cek attachment PURCHASING sudah semua diupload (cukup PENDING)
	allOK, err := checkAllDisposalAttachments(transactionNumber, transaction.ID, models.StageDisposalPurchasing, branchCode, false)
	if err != nil {
		return nil, err
	}
	if !allOK {
		return nil, errors.New("not all required purchasing documents are uploaded for all assets")
	}

	// Sama seperti SubmitDisposal: jalur SELL masuk APPROVAL_REQUEST dari sini,
	// jadi flow-nya dipastikan siap sebelum stage dipindahkan.
	if _, err := resolveDisposalApprovalFlow(transaction, models.FlowDisposalApprovalRequest); err != nil {
		return nil, err
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

	if err := InitiateDisposalApprovalRequest(userID, transactionNumber,
		dto.InitiateDisposalApprovalRequest{}); err != nil {
		return nil, fmt.Errorf("sale values saved but approval initiation failed: %w", err)
	}

	return GetDisposalDetail(transactionNumber)
}

// ============================================================
// INITIATE APPROVAL REQUEST
// DISPOSE: langsung dari DRAFT → APPROVAL_REQUEST via submit
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

// countDisposalApprovalsByFlowCode menghitung baris approval milik satu flow.
//
// Kolom `approval_flow_code` TIDAK ADA di tabel transaction_approvals —
// query lama memakainya sehingga MySQL error, Count tidak terisi, dan
// auto-complete selalu keluar lebih awal tanpa melakukan apa pun. Pembedanya
// yang benar adalah flow_id, karena kedua tahap disposal berbagi
// transaction_number dan transaction_type yang sama.
func countDisposalApprovalsByFlowCode(transactionNumber, flowCode string) (total int64, approved int64, err error) {
	flowIDs := config.DB.Model(&models.ApprovalFlow{}).
		Select("id").
		Where("flow_code = ?", flowCode)

	if err = config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ? AND flow_id IN (?)",
			transactionNumber, TxDisposalFlow, flowIDs).
		Count(&total).Error; err != nil {
		return 0, 0, err
	}

	flowIDsApproved := config.DB.Model(&models.ApprovalFlow{}).
		Select("id").
		Where("flow_code = ?", flowCode)

	if err = config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ? AND flow_id IN (?) AND status = ?",
			transactionNumber, TxDisposalFlow, flowIDsApproved, "approved").
		Count(&approved).Error; err != nil {
		return 0, 0, err
	}

	return total, approved, nil
}

func autoCompleteDisposalApprovalRequest(userID, transactionNumber, transactionType string) error {
	if transactionType != TxDisposalFlow {
		return nil
	}

	total, approved, err := countDisposalApprovalsByFlowCode(
		transactionNumber, models.FlowDisposalApprovalRequest)
	if err != nil {
		return err
	}

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

	// Tidak ada auto-initiate approval di sini.
	//
	// Tahap kesepakatan tidak lagi dijalankan per transaksi: transaksi berhenti
	// di APPROVAL_AGREEMENT dan menunggu dikelompokkan ke dalam sebuah
	// disposal agreement (lihat disposal_agreement_service.go), yang punya
	// nomor sendiri dan disetujui sekaligus oleh manajemen puncak.
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

func autoCompleteDisposalApprovalAgreement(userID, transactionNumber, transactionType string) error {
	if transactionType != TxDisposalFlow {
		return nil
	}

	total, approved, err := countDisposalApprovalsByFlowCode(
		transactionNumber, models.FlowDisposalApprovalAgreement)
	if err != nil {
		return err
	}

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

// disposalHasPendingRevision — true kalau ada aset yang sedang ditandai revisi.
// Selama true, pengaju hanya boleh menyentuh aset yang ditandai.
func disposalHasPendingRevision(transactionID uint) bool {
	var count int64
	config.DB.Model(&models.TransactionDisposalAsset{}).
		Where("transaction_id = ? AND needs_revision = ? AND status = ?",
			transactionID, true, models.DisposalAssetStatusPending).
		Count(&count)
	return count > 0
}

// assertAssetRevisable memastikan aset yang mau diubah memang bagian dari
// revisi yang diminta approver.
func assertAssetRevisable(transactionID uint, disposalAssetID uint) error {
	if !disposalHasPendingRevision(transactionID) {
		return nil
	}

	var da models.TransactionDisposalAsset
	if err := config.DB.
		Where("id = ? AND transaction_id = ?", disposalAssetID, transactionID).
		First(&da).Error; err != nil {
		return errors.New("disposal asset not found in this transaction")
	}

	if !da.NeedsRevision {
		return fmt.Errorf("asset %s is not part of the requested revision", da.AssetNumber)
	}

	return nil
}

// ============================================================
// REVISI
// APPROVAL_REQUEST / APPROVAL_AGREEMENT → DRAFT
// Approver step berjalan mengembalikan transaksi ke pembuatnya
// ============================================================

// currentPendingApproval mengambil baris approval yang sedang menunggu
// (step pending pertama) untuk flow disposal mana pun pada transaksi ini.
func currentPendingApproval(transactionNumber string) (*models.TransactionApproval, error) {
	var approvals []models.TransactionApproval
	if err := config.DB.
		Preload("ApprovalFlowStep").
		Where("transaction_number = ? AND transaction_type = ? AND status = ?",
			transactionNumber, TxDisposalFlow, "pending").
		Find(&approvals).Error; err != nil {
		return nil, err
	}

	if len(approvals) == 0 {
		return nil, errors.New("no pending approval found for this transaction")
	}

	// urutkan berdasarkan step_order — kolomnya ada di tabel lain, jadi
	// pengurutannya dilakukan setelah fetch (sama seperti
	// GetTransactionApprovalStatus)
	current := approvals[0]
	for _, item := range approvals[1:] {
		if item.ApprovalFlowStep == nil || current.ApprovalFlowStep == nil {
			continue
		}
		if item.ApprovalFlowStep.StepOrder < current.ApprovalFlowStep.StepOrder {
			current = item
		}
	}

	return &current, nil
}

// ReviseDisposal mengembalikan transaksi ke DRAFT supaya pembuatnya bisa
// memperbaiki aset/dokumen lalu submit ulang.
//
// Berbeda dengan RejectDisposal yang bersifat terminal (stage REJECTED, aset
// dilepas), revisi menyisakan draft-nya utuh — aset tetap menempel dan dokumen
// yang sudah diupload tidak dihapus.
func ReviseDisposal(userID string, transactionNumber string, req dto.ReviseDisposalRequest) (*dto.DisposalDetailResponse, error) {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	// Hanya dari stage approval. Stage lain sengaja tidak diizinkan: mulai
	// EXECUTE ke atas sudah ada efek samping (dokumen hasil, penghapusan aset)
	// yang tidak bisa dibatalkan hanya dengan memindahkan stage.
	if transaction.CurrentStage != models.StageDisposalApprovalRequest &&
		transaction.CurrentStage != models.StageDisposalApprovalAgreement {
		return nil, fmt.Errorf("cannot revise transaction in %s stage", transaction.CurrentStage)
	}

	// Yang boleh meminta revisi adalah approver step yang sedang berjalan —
	// aturannya sama dengan ApproveTransaction.
	pending, err := currentPendingApproval(transactionNumber)
	if err != nil {
		return nil, err
	}

	authorized := false
	if pending.ApproverUserID != nil && *pending.ApproverUserID == userID {
		authorized = true
	}
	if !authorized && pending.ApproverRoleID != nil {
		authorized = userHasRole(userID, *pending.ApproverRoleID)
	}
	if !authorized {
		return nil, errors.New("you are not the approver of the current step")
	}

	if err := validateApproverBranch(userID, transactionNumber, TxDisposalFlow); err != nil {
		return nil, err
	}

	// Aset yang ditandai harus benar-benar milik transaksi ini dan masih aktif
	var targets []models.TransactionDisposalAsset
	if err := config.DB.
		Where("transaction_id = ? AND id IN ? AND status = ?",
			transaction.ID, req.DisposalAssetIDs, models.DisposalAssetStatusPending).
		Find(&targets).Error; err != nil {
		return nil, err
	}

	if len(targets) != len(req.DisposalAssetIDs) {
		return nil, errors.New("some selected assets do not belong to this transaction or are no longer active")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Bersihkan penanda lama dulu supaya sisa revisi sebelumnya tidak menumpuk
	if err := tx.Model(&models.TransactionDisposalAsset{}).
		Where("transaction_id = ?", transaction.ID).
		Updates(map[string]interface{}{
			"needs_revision": false,
			"revision_notes": nil,
		}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Model(&models.TransactionDisposalAsset{}).
		Where("transaction_id = ? AND id IN ?", transaction.ID, req.DisposalAssetIDs).
		Updates(map[string]interface{}{
			"needs_revision": true,
			"revision_notes": req.RevisionNotes,
		}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Semua baris approval disposal dibuang supaya submit ulang bisa
	// meng-initiate flow dari awal. Tanpa ini, InitiateTransactionApproval
	// menolak dengan "approval already initiated" karena pengecekannya per
	// flow_id dan barisnya masih ada.
	if err := tx.Where("transaction_number = ? AND transaction_type = ?",
		transactionNumber, TxDisposalFlow).
		Delete(&models.TransactionApproval{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Tanda tangan lama tidak dihapus (jejak audit), cuma tidak lagi dianggap
	// yang terbaru — submit ulang akan membuat tanda tangan baru.
	if err := tx.Model(&models.ApprovalSignature{}).
		Where("transaction_number = ? AND transaction_type = ?",
			transactionNumber, TxDisposalFlow).
		Update("is_recent", false).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Nomor permohonan/kesepakatan ikut dikosongkan — akan diisi lagi saat
	// transaksi disubmit ulang.
	if err := tx.Model(transaction).
		Updates(map[string]interface{}{
			"approval_request_number":   nil,
			"approval_agreement_number": nil,
		}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, models.StageDisposalDraft); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageDisposalDraft,
		models.ActionRevise, userID, nil, &req.RevisionNotes); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalDetail(transactionNumber)
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

	branchCode := GetCreatorBranchCode(transaction.CreatedBy)

	// Cek attachment DRAFT sudah semua APPROVED
	draftOK, err := checkAllDisposalAttachments(transactionNumber, transaction.ID, models.StageDisposalDraft, branchCode, true)
	if err != nil {
		return nil, err
	}
	if !draftOK {
		return nil, errors.New("not all required draft documents are approved for all assets")
	}

	// Cek attachment EXECUTE sudah semua diupload (cukup PENDING)
	allOK, err := checkAllDisposalAttachments(transactionNumber, transaction.ID, models.StageDisposalExecute, branchCode, false)
	if err != nil {
		return nil, err
	}
	if !allOK {
		return nil, errors.New("not all required execute documents are uploaded for all assets")
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

	branchCode := GetCreatorBranchCode(transaction.CreatedBy)

	// Cek attachment DRAFT sudah semua APPROVED
	draftOK, err := checkAllDisposalAttachments(transactionNumber, transaction.ID, models.StageDisposalDraft, branchCode, true)
	if err != nil {
		return nil, err
	}
	if !draftOK {
		return nil, errors.New("not all required draft documents are approved for all assets")
	}

	// Cek attachment FINANCE sudah semua diupload (cukup PENDING)
	allOK, err := checkAllDisposalAttachments(transactionNumber, transaction.ID, models.StageDisposalFinance, branchCode, false)
	if err != nil {
		return nil, err
	}
	if !allOK {
		return nil, errors.New("not all required finance documents are uploaded for all assets")
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

	branchCode := GetCreatorBranchCode(transaction.CreatedBy)

	// Cek attachment DRAFT sudah semua APPROVED
	draftOK, err := checkAllDisposalAttachments(transactionNumber, transaction.ID, models.StageDisposalDraft, branchCode, true)
	if err != nil {
		return nil, err
	}
	if !draftOK {
		return nil, errors.New("not all required draft documents are approved for all assets")
	}

	// Cek attachment TAX sudah semua diupload (cukup PENDING)
	allOK, err := checkAllDisposalAttachments(transactionNumber, transaction.ID, models.StageDisposalTax, branchCode, false)
	if err != nil {
		return nil, err
	}
	if !allOK {
		return nil, errors.New("not all required tax documents are uploaded for all assets")
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

// terminateDisposal menutup transaksi disposal: aset dilepas kembali ke
// AVAILABLE, baris disposal asset ditandai CANCELLED, stage pindah ke stage
// terminal yang diminta, lalu nomor transaksi dilepas dari reservoir.
//
// Dipakai oleh tiga jalur yang berbeda aktornya:
//   - RejectDisposal   → pemegang permission reject_transaction
//   - autoRejectDisposal → approver step berjalan (lewat /transaction-approvals/reject)
//   - CancelDisposal   → pengaju transaksi itu sendiri
func terminateDisposal(userID string, transaction *models.Transaction, toStage, action, reason string) error {
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var disposalAssets []models.TransactionDisposalAsset
	config.DB.Where("transaction_id = ? AND status = ?", transaction.ID, models.DisposalAssetStatusPending).
		Find(&disposalAssets)

	for _, da := range disposalAssets {
		if err := tx.Model(&models.Asset{}).
			Where("id = ?", da.AssetID).
			Update("asset_status", models.AssetStatusAvailable).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Model(&da).Update("status", models.DisposalAssetStatusCancelled).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	fromStage := transaction.CurrentStage
	if err := updateTransactionStage(tx, transaction, toStage); err != nil {
		tx.Rollback()
		return err
	}

	if err := recordStage(tx, transaction.ID, transaction.TransactionNumber,
		fromStage, toStage,
		action, userID, nil, &reason); err != nil {
		tx.Rollback()
		return err
	}

	MarkTransactionAsExpired(transaction.TransactionNumber)

	return tx.Commit().Error
}

func RejectDisposal(userID string, transactionNumber string, req dto.RejectDisposalRequest) (*dto.DisposalDetailResponse, error) {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if isDisposalTerminalStage(transaction.CurrentStage) ||
		transaction.CurrentStage == models.StageDisposalDraft {
		return nil, fmt.Errorf("cannot reject transaction in %s stage", transaction.CurrentStage)
	}

	if err := terminateDisposal(userID, transaction,
		models.StageDisposalRejected, models.ActionReject, req.Reason); err != nil {
		return nil, err
	}

	return GetDisposalDetail(transactionNumber)
}

// isDisposalTerminalStage — stage yang sudah tidak bisa diapa-apakan lagi
func isDisposalTerminalStage(stage string) bool {
	return stage == models.StageDisposalFinished ||
		stage == models.StageDisposalRejected ||
		stage == models.StageDisposalCancelled
}

// autoRejectDisposal — dipanggil setelah salah satu step approval di-reject.
//
// Sebelumnya hanya autoRejectTransaction (khusus procurement) yang dipanggil,
// jadi reject step pada disposal cuma mengubah baris approval-nya: transaksi
// tetap menggantung di APPROVAL_REQUEST dan asetnya tidak pernah dilepas.
func autoRejectDisposal(userID, transactionNumber, transactionType, notes string) error {
	if transactionType != TxDisposalFlow {
		return nil
	}

	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return err
	}

	if isDisposalTerminalStage(transaction.CurrentStage) {
		return nil
	}

	reason := "Rejected by approver"
	if notes != "" {
		reason = notes
	}

	return terminateDisposal(userID, transaction,
		models.StageDisposalRejected, models.ActionReject, reason)
}

// CancelDisposal — pembatalan oleh PENGAJU transaksi.
//
// Berbeda dengan reject yang datang dari approver di stage berjalan,
// pembatalan datang dari pembuatnya sendiri dan berakhir di stage CANCELLED
// supaya keduanya bisa dibedakan di laporan maupun riwayat stage.
func CancelDisposal(userID string, transactionNumber string, req dto.CancelDisposalRequest) (*dto.DisposalDetailResponse, error) {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("only the creator can cancel this transaction")
	}

	if isDisposalTerminalStage(transaction.CurrentStage) {
		return nil, fmt.Errorf("cannot cancel transaction in %s stage", transaction.CurrentStage)
	}

	// Mulai EXECUTE sudah ada efek samping (dokumen hasil, penghapusan aset)
	// yang tidak bisa dibatalkan hanya dengan memindahkan stage.
	cancelableStages := map[string]bool{
		models.StageDisposalDraft:             true,
		models.StageDisposalPurchasing:        true,
		models.StageDisposalApprovalRequest:   true,
		models.StageDisposalApprovalAgreement: true,
	}
	if !cancelableStages[transaction.CurrentStage] {
		return nil, fmt.Errorf("cannot cancel transaction in %s stage, it is already being executed", transaction.CurrentStage)
	}

	// Approval yang sudah terbentuk ikut dibuang supaya tidak menggantung
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxDisposalFlow).
		Delete(&models.TransactionApproval{}).Error; err != nil {
		return nil, err
	}

	if err := terminateDisposal(userID, transaction,
		models.StageDisposalCancelled, models.ActionCancel, req.Reason); err != nil {
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
			NeedsRevision:     da.NeedsRevision,
			RevisionNotes:     da.RevisionNotes,
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
			CreatedByName:           resolveUserFullname(transaction.CreatedBy),
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

	// Selama revisi, upload hanya untuk aset yang ditandai approver
	if err := assertAssetRevisable(disposalAsset.TransactionID, disposalAsset.ID); err != nil {
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

func GetDisposalAttachmentStatus(transactionNumber string, transactionID uint, stage string, branchCode string) (*dto.DisposalAllAttachmentStatus, error) {
	var disposalAssets []models.TransactionDisposalAsset
	config.DB.
		Where("transaction_id = ? AND status = ?", transactionID, models.DisposalAssetStatusPending).
		Find(&disposalAssets)

	allCanProceed := true
	assetStatuses := make([]dto.DisposalAttachmentStatusSummary, 0, len(disposalAssets))

	for _, da := range disposalAssets {
		// Get required configs untuk disposal di stage ini
		requiredConfigs, err := getRequiredConfigs(TxDisposalFlow, stage, branchCode)
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

// GetDisposalAttachmentFile mengembalikan metadata + path absolut file
// attachment untuk di-stream ke client.
//
// File tidak dilayani sebagai static folder karena butuh autentikasi —
// dokumen disposal berisi data aset dan nilai jual.
func GetDisposalAttachmentFile(attachmentID uint) (*models.TransactionDisposalAttachment, string, error) {
	var att models.TransactionDisposalAttachment
	if err := config.DB.First(&att, attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", errors.New("attachment not found")
		}
		return nil, "", err
	}

	// Path disimpan server saat upload, tapi tetap dipastikan tidak keluar dari
	// folder penyimpanan sebelum dibuka.
	cleanPath := filepath.Clean(att.FilePath)
	storageRoot := filepath.Clean(AttachmentStoragePath)
	if !strings.HasPrefix(cleanPath, storageRoot+string(os.PathSeparator)) {
		return nil, "", errors.New("invalid attachment path")
	}

	if _, err := os.Stat(cleanPath); err != nil {
		return nil, "", errors.New("attachment file is missing on the server")
	}

	return &att, cleanPath, nil
}

// assertDisposalDocsApproved memastikan tidak ada dokumen wajib yang belum
// disetujui sebelum approver menyetujui step-nya.
//
// Dicek untuk seluruh stage yang sudah dilalui (termasuk stage berjalan),
// bukan cuma DRAFT — jalur SELL punya dokumen di PURCHASING juga.
func assertDisposalDocsApproved(transactionNumber string) error {
	transaction, err := getDisposalTransaction(transactionNumber)
	if err != nil {
		return err
	}

	branchCode := GetCreatorBranchCode(transaction.CreatedBy)
	stages := stagesForDisposalType(*transaction.DisposalType)

	for _, stage := range stages {
		// berhenti setelah stage berjalan
		ok, err := checkAllDisposalAttachments(
			transactionNumber, transaction.ID, stage, branchCode, true)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("there are %s documents that have not been approved yet, review them first", stage)
		}

		if stage == transaction.CurrentStage {
			break
		}
	}

	return nil
}

// checkAllDisposalAttachments — cek attachment di stage tertentu
// requireApproved=true  → semua wajib APPROVED (untuk cek stage sebelumnya)
// requireApproved=false → cukup uploaded PENDING/APPROVED, tidak boleh REJECTED/missing (untuk cek stage sekarang)
func checkAllDisposalAttachments(transactionNumber string, transactionID uint, stage string, branchCode string, requireApproved bool) (bool, error) {
	status, err := GetDisposalAttachmentStatus(transactionNumber, transactionID, stage, branchCode)
	if err != nil {
		return false, err
	}

	if requireApproved {
		// Semua wajib APPROVED
		return status.AllCanProceed, nil
	}

	// Cukup uploaded (PENDING atau APPROVED), tidak boleh REJECTED dan tidak boleh missing
	for _, asset := range status.Assets {
		if asset.TotalRejected > 0 {
			return false, nil
		}
		uploaded := asset.TotalApproved + asset.TotalPending
		if uploaded < asset.TotalRequired {
			return false, nil
		}
	}
	return true, nil
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
