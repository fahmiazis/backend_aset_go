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
)

// TxDisposalAgreement dipakai sebagai transaction_type pada tabel
// transaction_approvals, supaya mesin approval yang sudah ada bisa dipakai
// tanpa perubahan struktur.
const TxDisposalAgreement = "disposal_agreement"

// ============================================================
// NOMOR AGREEMENT
// Format: 0001/IX/2026-DPSL-AGMNT
// Tanpa kode cabang, karena satu agreement menggabungkan transaksi dari
// banyak cabang.
// ============================================================

func generateAgreementNumber(tx *gorm.DB) (string, error) {
	now := time.Now()
	romanMonth := GetRomanMonth(int(now.Month()))
	year := now.Year()

	var last models.DisposalAgreement
	err := tx.Unscoped().
		Where("MONTH(created_at) = ? AND YEAR(created_at) = ?", int(now.Month()), year).
		Order("id DESC").
		First(&last).Error

	sequence := 1
	if err == nil {
		parts := strings.Split(last.AgreementNumber, "/")
		if len(parts) > 0 {
			var lastSeq int
			fmt.Sscanf(parts[0], "%d", &lastSeq)
			sequence = lastSeq + 1
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	return fmt.Sprintf("%04d/%s/%d-DPSL-AGMNT", sequence, romanMonth, year), nil
}

// ============================================================
// KELAYAKAN
// ============================================================

// eligibleAgreementQuery — transaksi disposal yang boleh digabungkan:
// sudah di stage APPROVAL_AGREEMENT (artinya APPROVAL_REQUEST-nya lolos penuh)
// dan belum terikat agreement yang masih aktif.
//
// Agreement yang REJECTED tidak mengunci anggotanya — transaksinya kembali
// menunggu dan bisa dikelompokkan ulang.
func eligibleAgreementQuery() *gorm.DB {
	activeAgreementTx := config.DB.Model(&models.DisposalAgreementItem{}).
		Select("disposal_agreement_items.transaction_id").
		Joins("JOIN disposal_agreements ON disposal_agreements.id = disposal_agreement_items.agreement_id").
		Where("disposal_agreements.deleted_at IS NULL").
		Where("disposal_agreements.current_stage <> ?", models.StageAgreementRejected)

	return config.DB.Model(&models.Transaction{}).
		Where("transaction_type = ?", TxDisposalFlow).
		Where("current_stage = ?", models.StageDisposalApprovalAgreement).
		Where("id NOT IN (?)", activeAgreementTx)
}

func GetEligibleDisposalsForAgreement() ([]dto.DisposalAgreementItemResponse, error) {
	var transactions []models.Transaction
	if err := eligibleAgreementQuery().
		Order("created_at ASC").
		Find(&transactions).Error; err != nil {
		return nil, err
	}

	return mapAgreementItems(transactions), nil
}

func mapAgreementItems(transactions []models.Transaction) []dto.DisposalAgreementItemResponse {
	items := make([]dto.DisposalAgreementItemResponse, 0, len(transactions))

	for _, trx := range transactions {
		item := dto.DisposalAgreementItemResponse{
			TransactionID:     trx.ID,
			TransactionNumber: trx.TransactionNumber,
			DisposalType:      trx.DisposalType,
			CurrentStage:      trx.CurrentStage,
			BranchCode:        GetCreatorBranchCode(trx.CreatedBy),
			CreatedBy:         trx.CreatedBy,
		}

		if name, err := getUserFullName(trx.CreatedBy); err == nil && name != "" {
			item.CreatedByName = &name
		}

		var assets []models.TransactionDisposalAsset
		config.DB.
			Where("transaction_id = ? AND status = ?", trx.ID, models.DisposalAssetStatusPending).
			Find(&assets)

		item.TotalAssets = len(assets)

		var total float64
		hasValue := false
		for _, asset := range assets {
			if asset.SaleValue != nil {
				total += *asset.SaleValue
				hasValue = true
			}
		}
		if hasValue {
			item.TotalSaleValue = &total
		}

		items = append(items, item)
	}

	return items
}

func getUserFullName(userID string) (string, error) {
	var user models.User
	if err := config.DB.Select("fullname", "username").
		Where("id = ?", userID).First(&user).Error; err != nil {
		return "", err
	}
	if user.Fullname != "" {
		return user.Fullname, nil
	}
	return user.Username, nil
}

// ============================================================
// CREATE + INITIATE APPROVAL
// ============================================================

func CreateDisposalAgreement(userID string, req dto.CreateDisposalAgreementRequest) (*dto.DisposalAgreementResponse, error) {
	// Semua transaksi harus benar-benar layak — dicek ulang di server, bukan
	// mengandalkan daftar yang dikirim frontend.
	var transactions []models.Transaction
	if err := eligibleAgreementQuery().
		Where("transaction_number IN ?", req.TransactionNumbers).
		Find(&transactions).Error; err != nil {
		return nil, err
	}

	if len(transactions) != len(req.TransactionNumbers) {
		return nil, errors.New("some transactions are not eligible: they must have passed approval request and not already be in an active agreement")
	}

	// Flow harus siap sebelum apa pun dibuat, supaya tidak ada agreement
	// menggantung tanpa approval.
	flow, err := resolveAgreementApprovalFlow()
	if err != nil {
		return nil, err
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	agreementNumber, err := generateAgreementNumber(tx)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	agreement := models.DisposalAgreement{
		AgreementNumber: agreementNumber,
		CurrentStage:    models.StageAgreementApproval,
		Status:          models.AgreementStatusPending,
		Notes:           req.Notes,
		CreatedBy:       userID,
	}

	if err := tx.Create(&agreement).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, trx := range transactions {
		item := models.DisposalAgreementItem{
			AgreementID:       agreement.ID,
			TransactionID:     trx.ID,
			TransactionNumber: trx.TransactionNumber,
		}
		if err := tx.Create(&item).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Approval dibentuk setelah commit supaya id agreement sudah pasti ada
	approvalReq := dto.CreateTransactionApprovalRequest{
		FlowID:            flow.ID,
		TransactionNumber: agreementNumber,
		TransactionType:   TxDisposalAgreement,
	}

	if err := InitiateTransactionApproval(approvalReq); err != nil {
		return nil, fmt.Errorf("agreement created but approval initiation failed: %w", err)
	}

	return GetDisposalAgreementDetail(agreementNumber)
}

func resolveAgreementApprovalFlow() (*dto.ApprovalFlowResponse, error) {
	// Agreement lintas cabang, jadi selalu memakai konfigurasi global (ALL)
	flow, err := GetApprovalFlowByCodeAndBranch(models.FlowDisposalApprovalAgreement, "ALL")
	if err != nil {
		return nil, fmt.Errorf("approval flow %s not found, please configure it first", models.FlowDisposalApprovalAgreement)
	}

	if !flow.IsActive {
		return nil, fmt.Errorf("approval flow %s is inactive", models.FlowDisposalApprovalAgreement)
	}

	if len(flow.FlowSteps) == 0 {
		return nil, fmt.Errorf("approval flow %s has no steps configured", models.FlowDisposalApprovalAgreement)
	}

	return flow, nil
}

// ============================================================
// READ
// ============================================================

func GetDisposalAgreementByNumber(agreementNumber string) (*models.DisposalAgreement, error) {
	var agreement models.DisposalAgreement
	if err := config.DB.
		Where("agreement_number = ?", agreementNumber).
		First(&agreement).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("disposal agreement not found")
		}
		return nil, err
	}
	return &agreement, nil
}

func GetDisposalAgreementDetail(agreementNumber string) (*dto.DisposalAgreementResponse, error) {
	agreement, err := GetDisposalAgreementByNumber(agreementNumber)
	if err != nil {
		return nil, err
	}

	var items []models.DisposalAgreementItem
	config.DB.Where("agreement_id = ?", agreement.ID).Find(&items)

	ids := make([]uint, len(items))
	for i, item := range items {
		ids[i] = item.TransactionID
	}

	var transactions []models.Transaction
	if len(ids) > 0 {
		config.DB.Where("id IN ?", ids).Find(&transactions)
	}

	response := mapAgreementToResponse(*agreement)
	response.Items = mapAgreementItems(transactions)
	response.TotalItems = len(items)

	return &response, nil
}

func GetAllDisposalAgreements(filter dto.DisposalAgreementListFilter) ([]dto.DisposalAgreementResponse, int64, error) {
	query := config.DB.Model(&models.DisposalAgreement{})

	if filter.Stage != nil && *filter.Stage != "" {
		query = query.Where("current_stage = ?", *filter.Stage)
	}
	if filter.Search != nil && *filter.Search != "" {
		query = query.Where("agreement_number LIKE ?", "%"+*filter.Search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}

	var agreements []models.DisposalAgreement
	if err := query.
		Offset((filter.Page - 1) * filter.Limit).
		Limit(filter.Limit).
		Order("created_at DESC").
		Find(&agreements).Error; err != nil {
		return nil, 0, err
	}

	responses := make([]dto.DisposalAgreementResponse, len(agreements))
	for i, agreement := range agreements {
		responses[i] = mapAgreementToResponse(agreement)

		var count int64
		config.DB.Model(&models.DisposalAgreementItem{}).
			Where("agreement_id = ?", agreement.ID).Count(&count)
		responses[i].TotalItems = int(count)
	}

	return responses, total, nil
}

func mapAgreementToResponse(agreement models.DisposalAgreement) dto.DisposalAgreementResponse {
	response := dto.DisposalAgreementResponse{
		ID:              agreement.ID,
		AgreementNumber: agreement.AgreementNumber,
		CurrentStage:    agreement.CurrentStage,
		Status:          agreement.Status,
		Notes:           agreement.Notes,
		RejectionReason: agreement.RejectionReason,
		CreatedBy:       agreement.CreatedBy,
		CreatedAt:       agreement.CreatedAt,
		UpdatedAt:       agreement.UpdatedAt,
	}

	if name, err := getUserFullName(agreement.CreatedBy); err == nil && name != "" {
		response.CreatedByName = &name
	}

	return response
}

// ============================================================
// HOOK SETELAH APPROVE / REJECT
// ============================================================

// countAgreementApprovals menghitung baris approval milik agreement.
func countAgreementApprovals(agreementNumber string) (total int64, approved int64, err error) {
	if err = config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ?", agreementNumber, TxDisposalAgreement).
		Count(&total).Error; err != nil {
		return 0, 0, err
	}

	if err = config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ? AND status = ?",
			agreementNumber, TxDisposalAgreement, "approved").
		Count(&approved).Error; err != nil {
		return 0, 0, err
	}

	return total, approved, nil
}

// autoCompleteDisposalAgreement — dipanggil setelah sebuah step di-approve.
//
// Begitu seluruh step agreement disetujui, tiap transaksi anggota lanjut ke
// stage berikutnya memakai NOMOR TRANSAKSINYA SENDIRI, bukan nomor agreement.
// Nomor agreement hanya menaungi proses persetujuannya.
func autoCompleteDisposalAgreement(userID, agreementNumber, transactionType string) error {
	if transactionType != TxDisposalAgreement {
		return nil
	}

	total, approved, err := countAgreementApprovals(agreementNumber)
	if err != nil {
		return err
	}
	if total == 0 || approved < total {
		return nil
	}

	agreement, err := GetDisposalAgreementByNumber(agreementNumber)
	if err != nil {
		return err
	}
	if agreement.CurrentStage != models.StageAgreementApproval {
		return nil
	}

	var items []models.DisposalAgreementItem
	config.DB.Where("agreement_id = ?", agreement.ID).Find(&items)

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(agreement).Updates(map[string]interface{}{
		"current_stage": models.StageAgreementFinished,
		"status":        models.AgreementStatusApproved,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, item := range items {
		var transaction models.Transaction
		if err := tx.First(&transaction, item.TransactionID).Error; err != nil {
			tx.Rollback()
			return err
		}

		// transaksi yang sudah pindah duluan (mis. dibatalkan) dilewati
		if transaction.CurrentStage != models.StageDisposalApprovalAgreement {
			continue
		}

		disposalType := ""
		if transaction.DisposalType != nil {
			disposalType = *transaction.DisposalType
		}

		nextStage, err := nextStageForDisposal(disposalType, transaction.CurrentStage)
		if err != nil {
			tx.Rollback()
			return err
		}

		fromStage := transaction.CurrentStage
		if err := updateTransactionStage(tx, &transaction, nextStage); err != nil {
			tx.Rollback()
			return err
		}

		notes := fmt.Sprintf("Disetujui lewat kesepakatan %s", agreementNumber)
		if err := recordStage(tx, transaction.ID, transaction.TransactionNumber,
			fromStage, nextStage,
			models.ActionApprove, userID, nil, &notes); err != nil {
			tx.Rollback()
			return err
		}

		// nomor agreement disimpan di transaksi supaya jejaknya terbaca dari
		// sisi transaksi juga, tidak hanya dari sisi agreement
		if err := tx.Model(&models.Transaction{}).
			Where("id = ?", transaction.ID).
			Update("approval_agreement_number", agreementNumber).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// autoRejectDisposalAgreement — agreement ditolak.
//
// Transaksi anggotanya TIDAK ikut ditolak: mereka tetap di APPROVAL_AGREEMENT
// dan otomatis layak dikelompokkan lagi, karena agreement yang REJECTED tidak
// mengunci anggotanya (lihat eligibleAgreementQuery). Approval request yang
// sudah lolos jadi tidak terbuang percuma.
func autoRejectDisposalAgreement(userID, agreementNumber, transactionType, notes string) error {
	if transactionType != TxDisposalAgreement {
		return nil
	}

	agreement, err := GetDisposalAgreementByNumber(agreementNumber)
	if err != nil {
		return err
	}
	if agreement.CurrentStage != models.StageAgreementApproval {
		return nil
	}

	reason := "Rejected by approver"
	if notes != "" {
		reason = notes
	}

	return config.DB.Model(agreement).Updates(map[string]interface{}{
		"current_stage":    models.StageAgreementRejected,
		"status":           models.AgreementStatusRejected,
		"rejection_reason": reason,
	}).Error
}

// agreementMemberBranches — kode cabang seluruh pembuat transaksi anggota.
func agreementMemberBranches(agreementNumber string) ([]string, error) {
	agreement, err := GetDisposalAgreementByNumber(agreementNumber)
	if err != nil {
		return nil, err
	}

	var items []models.DisposalAgreementItem
	if err := config.DB.Where("agreement_id = ?", agreement.ID).Find(&items).Error; err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	branches := make([]string, 0, len(items))

	for _, item := range items {
		var transaction models.Transaction
		if err := config.DB.First(&transaction, item.TransactionID).Error; err != nil {
			continue
		}
		branch := GetCreatorBranchCode(transaction.CreatedBy)
		if branch != "" && !seen[branch] {
			seen[branch] = true
			branches = append(branches, branch)
		}
	}

	return branches, nil
}
