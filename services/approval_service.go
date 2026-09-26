package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ============================================================================
// APPROVAL FLOW CRUD
// ============================================================================

func GetAllApprovalFlows() ([]dto.ApprovalFlowResponse, error) {
	var flows []models.ApprovalFlow

	if err := config.DB.
		Preload("FlowSteps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_order ASC")
		}).
		Preload("FlowSteps.Role").
		Preload("FlowSteps.Branch").
		// tanpa preload ini, assigned_username tidak pernah terisi di list
		// dan UI cuma bisa menampilkan UUID-nya
		Preload("AssignedUser").
		Find(&flows).Error; err != nil {
		return nil, err
	}

	return mapApprovalFlowsToResponse(flows), nil
}

func GetApprovalFlowByID(id string) (*dto.ApprovalFlowResponse, error) {
	var flow models.ApprovalFlow

	if err := config.DB.
		Preload("FlowSteps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_order ASC")
		}).
		Preload("FlowSteps.Role").
		Preload("FlowSteps.Branch").
		First(&flow, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("approval flow not found")
		}
		return nil, err
	}

	response := mapApprovalFlowToResponse(flow)
	return &response, nil
}

// GetApprovalFlowByCode - backward compatible, pakai ALL sebagai branch
func GetApprovalFlowByCode(code string) (*dto.ApprovalFlowResponse, error) {
	return GetApprovalFlowByCodeAndBranch(code, "ALL")
}

// GetApprovalFlowByCodeAndBranch - lookup dengan fallback:
// 1. flow_code + branch_code spesifik
// 2. flow_code + branch_code = ALL (fallback)
func GetApprovalFlowByCodeAndBranch(code string, branchCode string) (*dto.ApprovalFlowResponse, error) {
	var flow models.ApprovalFlow

	preloadFn := func(db *gorm.DB) *gorm.DB {
		return db.Order("step_order ASC")
	}

	// Coba cari yang spesifik dulu (bukan ALL)
	if branchCode != "ALL" && branchCode != "" {
		err := config.DB.
			Preload("FlowSteps", preloadFn).
			Preload("FlowSteps.Role").
			Preload("FlowSteps.Branch").
			Where("flow_code = ? AND branch_code = ? AND is_active = ?", code, branchCode, true).
			First(&flow).Error

		if err == nil {
			response := mapApprovalFlowToResponse(flow)
			return &response, nil
		}

		// Kalau tidak ketemu yang spesifik, lanjut ke fallback ALL
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// Fallback ke ALL
	err := config.DB.
		Preload("FlowSteps", preloadFn).
		Preload("FlowSteps.Role").
		Preload("FlowSteps.Branch").
		Where("flow_code = ? AND branch_code = ? AND is_active = ?", code, "ALL", true).
		First(&flow).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("approval flow '%s' not found for branch '%s' or ALL", code, branchCode)
		}
		return nil, err
	}

	response := mapApprovalFlowToResponse(flow)
	return &response, nil
}

func CreateApprovalFlow(req dto.CreateApprovalFlowRequest) (*dto.ApprovalFlowResponse, error) {
	// Validate assigned_user_id if assignment_type is user_specific
	if req.AssignmentType == "user_specific" && (req.AssignedUserID == nil || *req.AssignedUserID == "") {
		return nil, errors.New("assigned_user_id is required when assignment_type is user_specific")
	}

	// Convert allowed_creator_roles to JSON string
	var allowedRolesJSON *string
	if len(req.AllowedCreatorRoles) > 0 {
		rolesBytes, err := json.Marshal(req.AllowedCreatorRoles)
		if err != nil {
			return nil, err
		}
		rolesStr := string(rolesBytes)
		allowedRolesJSON = &rolesStr
	}

	branchCode := req.BranchCode
	if branchCode == "" {
		branchCode = "ALL"
	}

	// Index unik uq_flow_code_branch (flow_code, branch_code) TIDAK menyertakan
	// deleted_at, jadi baris yang sudah di-soft-delete tetap memblokir kode yang
	// sama. Tanpa penanganan ini, hapus lalu tambah ulang selalu gagal dengan
	// "Duplicate entry". Baris lamanya dipulihkan dan ditimpa nilai baru supaya
	// foreign key dari transaction_approvals / approval_flow_steps tetap utuh.
	var existing models.ApprovalFlow
	err := config.DB.Unscoped().
		Where("flow_code = ? AND branch_code = ?", req.FlowCode, branchCode).
		First(&existing).Error

	if err == nil {
		if !existing.DeletedAt.Valid {
			return nil, errors.New("approval flow with this code already exists for branch " + branchCode)
		}

		restore := map[string]interface{}{
			"deleted_at":            nil,
			"flow_name":             req.FlowName,
			"approval_way":          req.ApprovalWay,
			"assignment_type":       req.AssignmentType,
			"assigned_user_id":      req.AssignedUserID,
			"is_customizable":       req.IsCustomizable,
			"allowed_creator_roles": allowedRolesJSON,
			"description":           req.Description,
			"is_active":             req.IsActive,
		}

		if err := config.DB.Unscoped().Model(&existing).Updates(restore).Error; err != nil {
			return nil, err
		}

		return GetApprovalFlowByID(existing.ID)
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	flow := models.ApprovalFlow{
		FlowCode:            req.FlowCode,
		BranchCode:          branchCode,
		FlowName:            req.FlowName,
		ApprovalWay:         req.ApprovalWay,
		AssignmentType:      req.AssignmentType,
		AssignedUserID:      req.AssignedUserID,
		IsCustomizable:      req.IsCustomizable,
		AllowedCreatorRoles: allowedRolesJSON,
		Description:         req.Description,
		IsActive:            req.IsActive,
	}

	if err := config.DB.Create(&flow).Error; err != nil {
		return nil, err
	}

	return GetApprovalFlowByID(flow.ID)
}

func UpdateApprovalFlow(id string, req dto.UpdateApprovalFlowRequest) (*dto.ApprovalFlowResponse, error) {
	var flow models.ApprovalFlow
	if err := config.DB.First(&flow, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("approval flow not found")
		}
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.FlowCode != "" {
		updates["flow_code"] = req.FlowCode
	}
	if req.BranchCode != "" {
		updates["branch_code"] = req.BranchCode
	}
	if req.FlowName != "" {
		updates["flow_name"] = req.FlowName
	}
	if req.ApprovalWay != "" {
		updates["approval_way"] = req.ApprovalWay
	}
	if req.AssignmentType != "" {
		updates["assignment_type"] = req.AssignmentType
	}
	if req.AssignedUserID != nil {
		updates["assigned_user_id"] = req.AssignedUserID
	}
	if req.IsCustomizable != nil {
		updates["is_customizable"] = *req.IsCustomizable
	}
	if len(req.AllowedCreatorRoles) > 0 {
		rolesJSON, _ := json.Marshal(req.AllowedCreatorRoles)
		updates["allowed_creator_roles"] = string(rolesJSON)
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := config.DB.Model(&flow).Updates(updates).Error; err != nil {
		return nil, err
	}

	return GetApprovalFlowByID(id)
}

func DeleteApprovalFlow(id string) error {
	var flow models.ApprovalFlow
	if err := config.DB.First(&flow, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("approval flow not found")
		}
		return err
	}

	return config.DB.Delete(&flow).Error
}

// ============================================================================
// APPROVAL FLOW STEP CRUD
// ============================================================================

func CreateApprovalFlowStep(req dto.CreateApprovalFlowStepRequest) (*dto.ApprovalFlowStepResponse, error) {
	// Validate flow exists
	var flow models.ApprovalFlow
	if err := config.DB.First(&flow, "id = ?", req.FlowID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("approval flow not found")
		}
		return nil, err
	}

	// Validate role if provided
	if req.RoleID != nil && *req.RoleID != "" {
		var role models.Role
		if err := config.DB.First(&role, "id = ?", *req.RoleID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.New("role not found")
			}
			return nil, err
		}
	}

	// Validate branch if provided
	if req.BranchID != nil && *req.BranchID != "" {
		var branch models.Branch
		if err := config.DB.First(&branch, "id = ?", *req.BranchID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.New("branch not found")
			}
			return nil, err
		}
	}

	// Set defaults if not provided
	stepType := "all"
	stepCategory := "all"
	stepApprovalWay := "web"

	if req.Type != "" {
		stepType = req.Type
	}
	if req.Category != "" {
		stepCategory = req.Category
	}
	if req.ApprovalWay != "" {
		stepApprovalWay = req.ApprovalWay
	}

	step := models.ApprovalFlowStep{
		FlowID:       req.FlowID,
		StepOrder:    req.StepOrder,
		StepName:     req.StepName,
		StepRole:     req.StepRole,
		RoleID:       req.RoleID,
		BranchID:     req.BranchID,
		Structure:    req.Structure,
		IsRequired:   req.IsRequired,
		CanSkip:      req.CanSkip,
		IsVisible:    req.IsVisible,
		Type:         stepType,
		Category:     stepCategory,
		ApprovalWay:  stepApprovalWay,
		AutoApprove:  req.AutoApprove,
		TimeoutHours: req.TimeoutHours,
		Conditions:   req.Conditions,
	}

	if err := config.DB.Create(&step).Error; err != nil {
		return nil, err
	}

	return GetApprovalFlowStepByID(step.ID)
}

func GetApprovalFlowStepByID(id string) (*dto.ApprovalFlowStepResponse, error) {
	var step models.ApprovalFlowStep

	if err := config.DB.
		Preload("Role").
		Preload("Branch").
		First(&step, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("approval flow step not found")
		}
		return nil, err
	}

	response := mapApprovalFlowStepToResponse(step)
	return &response, nil
}

func UpdateApprovalFlowStep(id string, req dto.UpdateApprovalFlowStepRequest) (*dto.ApprovalFlowStepResponse, error) {
	var step models.ApprovalFlowStep
	if err := config.DB.First(&step, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("approval flow step not found")
		}
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.StepOrder != nil {
		updates["step_order"] = *req.StepOrder
	}
	if req.StepName != "" {
		updates["step_name"] = req.StepName
	}
	if req.StepRole != "" {
		updates["step_role"] = req.StepRole
	}
	if req.RoleID != nil {
		updates["role_id"] = req.RoleID
	}
	if req.BranchID != nil {
		updates["branch_id"] = req.BranchID
	}
	if req.Structure != nil {
		updates["structure"] = req.Structure
	}
	if req.IsRequired != nil {
		updates["is_required"] = *req.IsRequired
	}
	if req.CanSkip != nil {
		updates["can_skip"] = *req.CanSkip
	}
	if req.IsVisible != nil {
		updates["is_visible"] = *req.IsVisible
	}
	if req.Type != nil && *req.Type != "" {
		updates["type"] = *req.Type
	}
	if req.Category != nil && *req.Category != "" {
		updates["category"] = *req.Category
	}
	if req.ApprovalWay != nil && *req.ApprovalWay != "" {
		updates["approval_way"] = *req.ApprovalWay
	}
	if req.AutoApprove != nil {
		updates["auto_approve"] = *req.AutoApprove
	}
	if req.TimeoutHours != nil {
		updates["timeout_hours"] = req.TimeoutHours
	}
	if req.Conditions != nil {
		updates["conditions"] = req.Conditions
	}

	if err := config.DB.Model(&step).Updates(updates).Error; err != nil {
		return nil, err
	}

	return GetApprovalFlowStepByID(id)
}

func UpdateBulkStepOrderFlowStep(flowID string, req dto.UpdateBulkStepOrderFlowStep) error {
	// Check if user exists
	var approvalStep models.ApprovalFlowStep
	if err := config.DB.First(&approvalStep, "flow_id = ?", flowID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("approval flow step not found")
		}
		return err
	}

	// Assign new roles
	for i, listID := range req.ListIDs {
		var approvalStepId models.ApprovalFlowStep
		err := config.DB.
			Where("id = ?", listID).
			First(&approvalStepId).Error
		if err == nil {
			updateFields := map[string]any{
				"step_order": i + 1,
			}

			if err := config.DB.
				Model(&approvalStepId).
				Updates(updateFields).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func DeleteApprovalFlowStep(id string) error {
	var step models.ApprovalFlowStep
	if err := config.DB.First(&step, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("approval flow step not found")
		}
		return err
	}

	return config.DB.Delete(&step).Error
}

// ============================================================================
// TRANSACTION APPROVAL - INITIATE & PROCESS
// ============================================================================

// transactionCreatorID mengambil pembuat transaksi dari tabel transactions.
// Dipakai untuk auto-approve step yang approver-nya adalah si pembuat sendiri.
// Sengaja tidak memakai user yang memanggil endpoint initiate — untuk
// POST /transaction-approvals/initiate pemanggilnya bisa saja admin, sementara
// yang "menandatangani" tetap pembuat transaksi.
func transactionCreatorID(transactionNumber string) (string, error) {
	var transaction models.Transaction
	if err := config.DB.
		Select("created_by").
		Where("transaction_number = ?", transactionNumber).
		First(&transaction).Error; err == nil {
		return transaction.CreatedBy, nil
	}

	// Agreement disposal punya nomor sendiri dan tidak ada di tabel transactions
	var agreement models.DisposalAgreement
	if err := config.DB.
		Select("created_by").
		Where("agreement_number = ?", transactionNumber).
		First(&agreement).Error; err != nil {
		return "", err
	}
	return agreement.CreatedBy, nil
}

// userHasRole — aturannya sama dengan pengecekan di ApproveTransaction.
func userHasRole(userID, roleID string) bool {
	var count int64
	config.DB.Model(&models.UserRole{}).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Count(&count)
	return count > 0
}

// runPostApprovalHooks menjalankan auto-complete per jenis transaksi.
// Masing-masing fungsi sudah memfilter transaction_type-nya sendiri.
// Dipakai setelah approve manual maupun setelah auto-approve saat initiate.
// validateAgreementApproverBranches memastikan approver punya akses ke semua
// cabang yang transaksinya ikut dalam agreement.
func validateAgreementApproverBranches(approverUserID, agreementNumber string) error {
	memberBranches, err := agreementMemberBranches(agreementNumber)
	if err != nil {
		return err
	}

	var approverBranches []models.UserBranch
	if err := config.DB.
		Preload("Branch").
		Where("user_id = ?", approverUserID).
		Find(&approverBranches).Error; err != nil {
		return fmt.Errorf("failed to get approver branches: %w", err)
	}

	owned := map[string]bool{}
	for _, ub := range approverBranches {
		if ub.Branch != nil {
			owned[ub.Branch.BranchCode] = true
		}
	}

	missing := make([]string, 0)
	for _, branch := range memberBranches {
		if !owned[branch] {
			missing = append(missing, branch)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("you do not have access to branch %s, which is part of this agreement",
			strings.Join(missing, ", "))
	}

	return nil
}

func runPostApprovalHooks(userID, transactionNumber, transactionType string) {
	if err := autoCompleteProcurementApproval(userID, transactionNumber, transactionType); err != nil {
		fmt.Printf("auto complete procurement approval warning: %v\n", err)
	}

	if err := autoCompleteMutationApproval(userID, transactionNumber, transactionType); err != nil {
		fmt.Printf("auto complete mutation approval warning: %v\n", err)
	}

	if err := autoCompleteDisposalApprovalRequest(userID, transactionNumber, transactionType); err != nil {
		fmt.Printf("auto complete disposal approval request warning: %v\n", err)
	}

	if err := autoCompleteDisposalApprovalAgreement(userID, transactionNumber, transactionType); err != nil {
		fmt.Printf("auto complete disposal approval agreement warning: %v\n", err)
	}

	if err := autoCompleteDisposalAgreement(userID, transactionNumber, transactionType); err != nil {
		fmt.Printf("auto complete disposal agreement warning: %v\n", err)
	}
}

// InitiateTransactionApproval creates all approval records for a transaction based on flow
func InitiateTransactionApproval(req dto.CreateTransactionApprovalRequest) error {
	// Get approval flow
	flow, err := GetApprovalFlowByID(req.FlowID)
	if err != nil {
		return err
	}

	if !flow.IsActive {
		return errors.New("approval flow is inactive")
	}

	// Cek duplikasi PER FLOW, bukan per transaksi.
	//
	// Disposal memakai dua flow berurutan pada transaksi yang sama
	// (DISPOSAL_APPROVAL_REQUEST lalu DISPOSAL_APPROVAL_AGREEMENT). Kalau
	// pengecekannya hanya transaction_number + transaction_type, tahap
	// agreement SELALU ditolak "approval already initiated" karena baris dari
	// tahap request masih ada. Untuk procurement/mutation yang cuma punya satu
	// flow per transaksi, hasilnya sama persis seperti sebelumnya.
	var existingCount int64
	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ? AND flow_id = ?",
			req.TransactionNumber, req.TransactionType, req.FlowID).
		Count(&existingCount)

	if existingCount > 0 {
		return errors.New("approval already initiated for this transaction")
	}

	// Mark transaction number as used in reservoir
	if err := MarkTransactionAsUsed(req.TransactionNumber); err != nil {
		// If error, it might not exist in reservoir (manual entry), continue anyway
		// Or you can make it strict by returning the error
		// return err
	}

	// Pembuat transaksi sering juga menjadi approver di step pertama (PIC yang
	// mengajukan menandatangani permohonannya sendiri). Step seperti itu
	// di-approve otomatis supaya dia tidak perlu menyetujui pengajuannya
	// sendiri secara manual, dan flow langsung lompat ke approver berikutnya.
	//
	// Hanya berlaku untuk step di BARISAN DEPAN: begitu ketemu satu step yang
	// approver-nya bukan si pembuat, sisanya tetap pending. Step milik pembuat
	// yang berada di tengah/akhir flow (misal peran receiver) tetap harus
	// disetujui manual karena maknanya berbeda.
	creatorID, creatorErr := transactionCreatorID(req.TransactionNumber)
	autoApprove := creatorErr == nil && creatorID != ""
	now := time.Now()
	allApproved := true

	// Create approval records for each step
	for _, step := range flow.FlowSteps {
		approval := models.TransactionApproval{
			FlowID:            req.FlowID,
			FlowStepID:        step.ID,
			TransactionNumber: req.TransactionNumber,
			TransactionType:   req.TransactionType,
			Status:            "pending",
			StatusView:        "visible",
			Metadata:          req.Metadata,
		}

		// Assign approver based on step configuration
		if step.RoleID != nil {
			approval.ApproverRoleID = step.RoleID
		}

		// Set status_view based on step configuration
		if !step.IsVisible {
			approval.StatusView = "hidden"
		}

		// approver step ini adalah si pembuat transaksi?
		isCreatorStep := autoApprove &&
			step.RoleID != nil &&
			userHasRole(creatorID, *step.RoleID)

		if isCreatorStep {
			notes := "Auto-approved: pembuat transaksi adalah approver di step ini"
			approval.Status = "approved"
			approval.ApprovedAt = &now
			approval.ApprovedBy = &creatorID
			approval.Notes = &notes
		} else {
			// step pertama yang bukan milik pembuat menghentikan auto-approve
			autoApprove = false
			allApproved = false
		}

		if err := config.DB.Create(&approval).Error; err != nil {
			return err
		}

		// Signature dibuat juga untuk auto-approve, supaya jejaknya sama
		// dengan approve manual di ApproveTransaction.
		if isCreatorStep {
			signature := models.ApprovalSignature{
				TransactionNumber: req.TransactionNumber,
				TransactionType:   req.TransactionType,
				UserID:            creatorID,
				RoleID:            step.RoleID,
				StepRole:          step.StepRole,
				SignedAt:          now,
				Status:            "signed",
				Notes:             approval.Notes,
				IsRecent:          true,
			}

			if err := config.DB.Create(&signature).Error; err != nil {
				return err
			}
		}
	}

	// Kalau SELURUH step ternyata milik pembuat, tidak akan ada approve manual
	// yang memicu transisi stage — jadi hook-nya dipanggil dari sini.
	if allApproved && len(flow.FlowSteps) > 0 {
		runPostApprovalHooks(creatorID, req.TransactionNumber, req.TransactionType)
	}

	return nil
}

// ApproveTransaction approves a specific approval step
func ApproveTransaction(userID string, req dto.ApproveTransactionRequest) error {
	var approval models.TransactionApproval

	if err := config.DB.
		Preload("ApprovalFlowStep").
		First(&approval, "id = ?", req.TransactionApprovalID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("transaction approval not found")
		}
		return err
	}

	// Check if already processed
	if approval.Status != "pending" {
		return errors.New("approval already processed")
	}

	// Check if user has permission to approve
	if approval.ApproverUserID != nil && *approval.ApproverUserID != userID {
		return errors.New("you are not authorized to approve this transaction")
	}

	// If approval is by role, check if user has the role
	if approval.ApproverRoleID != nil {
		var userRole models.UserRole
		if err := config.DB.
			Where("user_id = ? AND role_id = ?", userID, *approval.ApproverRoleID).
			First(&userRole).Error; err != nil {
			return errors.New("you do not have the required role to approve this transaction")
		}
	}

	// Validasi branch approver harus sama dengan branch creator transaksi
	if err := validateApproverBranch(userID, approval.TransactionNumber, approval.TransactionType); err != nil {
		return err
	}

	// Dokumen wajib harus sudah direview dulu. Sebelumnya approve sama sekali
	// tidak memeriksa attachment, jadi transaksi bisa disetujui walau dokumen
	// pendukungnya masih PENDING atau malah REJECTED.
	if approval.TransactionType == TxDisposalFlow {
		if err := assertDisposalDocsApproved(approval.TransactionNumber); err != nil {
			return err
		}
	}

	// Update approval
	now := time.Now()
	updates := map[string]interface{}{
		"status":      "approved",
		"approved_at": now,
		"approved_by": userID,
		"notes":       req.Notes,
	}

	if err := config.DB.Model(&approval).Updates(updates).Error; err != nil {
		return err
	}

	// Create signature record
	signature := models.ApprovalSignature{
		TransactionNumber: approval.TransactionNumber,
		TransactionType:   approval.TransactionType,
		UserID:            userID,
		RoleID:            approval.ApproverRoleID,
		StepRole:          approval.ApprovalFlowStep.StepRole,
		SignedAt:          now,
		Status:            "signed",
		Notes:             req.Notes,
		IsRecent:          true,
	}

	if err := config.DB.Create(&signature).Error; err != nil {
		return err
	}

	// Auto-trigger complete approval jika semua step sudah approved.
	// Disposal punya dua tahap dengan flow masing-masing; sebelumnya kedua
	// fungsinya tidak pernah dipanggil sehingga disposal mandek di
	// APPROVAL_REQUEST walau semua step sudah approved.
	runPostApprovalHooks(userID, approval.TransactionNumber, approval.TransactionType)

	return nil
}

// RejectTransaction rejects a specific approval step
func RejectTransaction(userID string, req dto.RejectTransactionRequest) error {
	var approval models.TransactionApproval

	if err := config.DB.
		Preload("ApprovalFlowStep").
		First(&approval, "id = ?", req.TransactionApprovalID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("transaction approval not found")
		}
		return err
	}

	// Check if already processed
	if approval.Status != "pending" {
		return errors.New("approval already processed")
	}

	// Check if user has permission to reject
	if approval.ApproverUserID != nil && *approval.ApproverUserID != userID {
		return errors.New("you are not authorized to reject this transaction")
	}

	// If approval is by role, check if user has the role
	if approval.ApproverRoleID != nil {
		var userRole models.UserRole
		if err := config.DB.
			Where("user_id = ? AND role_id = ?", userID, *approval.ApproverRoleID).
			First(&userRole).Error; err != nil {
			return errors.New("you do not have the required role to reject this transaction")
		}
	}

	// Validasi branch approver harus sama dengan branch creator transaksi
	if err := validateApproverBranch(userID, approval.TransactionNumber, approval.TransactionType); err != nil {
		return err
	}

	// Update approval
	now := time.Now()
	updates := map[string]interface{}{
		"status":      "rejected",
		"rejected_at": now,
		"rejected_by": userID,
		"notes":       req.Notes,
	}

	if err := config.DB.Model(&approval).Updates(updates).Error; err != nil {
		return err
	}

	// Create signature record
	signature := models.ApprovalSignature{
		TransactionNumber: approval.TransactionNumber,
		TransactionType:   approval.TransactionType,
		UserID:            userID,
		RoleID:            approval.ApproverRoleID,
		StepRole:          approval.ApprovalFlowStep.StepRole,
		SignedAt:          now,
		Status:            "rejected",
		Notes:             req.Notes,
		IsRecent:          true,
	}

	if err := config.DB.Create(&signature).Error; err != nil {
		return err
	}

	// FIX: dulu di sini `*req.Notes` langsung di-dereference, padahal Notes
	// adalah pointer opsional — reject tanpa catatan bikin panic nil pointer.
	notesText := ""
	if req.Notes != nil {
		notesText = *req.Notes
	}

	// Auto-reject transaksi jika approval di-reject
	if err := autoRejectDisposalAgreement(userID, approval.TransactionNumber, approval.TransactionType, notesText); err != nil {
		fmt.Printf("auto reject disposal agreement warning: %v\n", err)
	}

	if err := autoRejectDisposal(userID, approval.TransactionNumber, approval.TransactionType, notesText); err != nil {
		fmt.Printf("auto reject disposal warning: %v\n", err)
	}

	if err := autoRejectTransaction(userID, approval.TransactionNumber, approval.TransactionType, notesText); err != nil {
		fmt.Printf("auto reject transaction warning: %v\n", err)
	}

	return nil
}

// GetTransactionApprovalStatus gets full approval status for a transaction
func GetTransactionApprovalStatus(transactionNumber, transactionType string) (*dto.TransactionApprovalSummary, error) {
	return GetTransactionApprovalStatusByFlowCode(transactionNumber, transactionType, "")
}

// GetTransactionApprovalStatusByFlowCode sama dengan GetTransactionApprovalStatus,
// tapi bisa dibatasi ke satu flow.
//
// Disposal memakai dua flow pada transaction_number yang sama
// (DISPOSAL_APPROVAL_REQUEST lalu DISPOSAL_APPROVAL_AGREEMENT). Tanpa filter
// ini, panel status tahap kesepakatan ikut menghitung step tahap permohonan
// sehingga total_steps dan completed_steps-nya salah.
// flowCode kosong = tidak difilter (perilaku lama, dipakai procurement/mutation).
func GetTransactionApprovalStatusByFlowCode(transactionNumber, transactionType, flowCode string) (*dto.TransactionApprovalSummary, error) {
	var approvals []models.TransactionApproval

	query := config.DB.
		Preload("ApprovalFlowStep").
		Preload("ApproverUser").
		Preload("ApproverRole").
		Preload("ActualApprover").
		Preload("ActualRejecter").
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, transactionType)

	if flowCode != "" {
		flowIDs := config.DB.Model(&models.ApprovalFlow{}).
			Select("id").
			Where("flow_code = ?", flowCode)
		query = query.Where("flow_id IN (?)", flowIDs)
	}

	if err := query.Find(&approvals).Error; err != nil {
		return nil, err
	}

	if len(approvals) == 0 {
		return nil, errors.New("no approval found for this transaction")
	}

	// Sort by step_order setelah fetch karena ORDER BY tidak bisa langsung
	// ke kolom di table lain tanpa explicit JOIN
	sort.Slice(approvals, func(i, j int) bool {
		stepI, stepJ := 0, 0
		if approvals[i].ApprovalFlowStep != nil {
			stepI = approvals[i].ApprovalFlowStep.StepOrder
		}
		if approvals[j].ApprovalFlowStep != nil {
			stepJ = approvals[j].ApprovalFlowStep.StepOrder
		}
		return stepI < stepJ
	})

	// Calculate summary
	totalSteps := len(approvals)
	completedSteps := 0
	var currentStep *dto.ApprovalFlowStepResponse
	overallStatus := "pending"

	for i, approval := range approvals {
		if approval.Status == "approved" {
			completedSteps++
		} else if approval.Status == "rejected" {
			overallStatus = "rejected"
			break
		} else if approval.Status == "pending" && currentStep == nil {
			if approval.ApprovalFlowStep != nil {
				step := mapApprovalFlowStepToResponse(*approval.ApprovalFlowStep)
				currentStep = &step
			}
		}

		// Check if this is the last approval and it's approved
		if i == len(approvals)-1 && approval.Status == "approved" {
			overallStatus = "approved"
		}
	}

	if completedSteps > 0 && completedSteps < totalSteps && overallStatus != "rejected" {
		overallStatus = "in_progress"
	}

	summary := dto.TransactionApprovalSummary{
		TransactionNumber: transactionNumber,
		TransactionType:   transactionType,
		TotalSteps:        totalSteps,
		CompletedSteps:    completedSteps,
		CurrentStep:       currentStep,
		Status:            overallStatus,
		Approvals:         mapTransactionApprovalsToResponse(approvals),
		CreatedAt:         approvals[0].CreatedAt,
	}

	return &summary, nil
}

// GetUserPendingApprovals gets all pending approvals for a user
func GetUserPendingApprovals(userID string) ([]dto.TransactionApprovalResponse, error) {
	// Get user roles
	var userRoles []models.UserRole
	if err := config.DB.Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, err
	}

	roleIDs := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roleIDs[i] = ur.RoleID
	}

	// Get pending approvals
	var approvals []models.TransactionApproval

	query := config.DB.
		Preload("ApprovalFlowStep").
		Preload("ApproverUser").
		Preload("ApproverRole").
		Where("status = ? AND status_view = ?", "pending", "visible")

	// Filter by user ID or user's roles
	if len(roleIDs) > 0 {
		query = query.Where("approver_user_id = ? OR approver_role_id IN ?", userID, roleIDs)
	} else {
		query = query.Where("approver_user_id = ?", userID)
	}

	if err := query.Find(&approvals).Error; err != nil {
		return nil, err
	}

	return mapTransactionApprovalsToResponse(approvals), nil
}

// ============================================================================
// HELPERS - Mappers
// ============================================================================

func mapApprovalFlowToResponse(flow models.ApprovalFlow) dto.ApprovalFlowResponse {
	response := dto.ApprovalFlowResponse{
		ID:                  flow.ID,
		FlowCode:            flow.FlowCode,
		BranchCode:          flow.BranchCode,
		FlowName:            flow.FlowName,
		ApprovalWay:         flow.ApprovalWay,
		AssignmentType:      flow.AssignmentType,
		AssignedUserID:      flow.AssignedUserID,
		IsCustomizable:      flow.IsCustomizable,
		AllowedCreatorRoles: flow.AllowedCreatorRoles,
		IsCustom:            flow.IsCustom,
		CreatedBy:           flow.CreatedBy,
		BaseFlowID:          flow.BaseFlowID,
		CustomStatus:        flow.CustomStatus,
		VerifiedBy:          flow.VerifiedBy,
		VerifiedAt:          flow.VerifiedAt,
		VerificationNotes:   flow.VerificationNotes,
		RejectionReason:     flow.RejectionReason,
		Description:         flow.Description,
		IsActive:            flow.IsActive,
		CreatedAt:           flow.CreatedAt,
		UpdatedAt:           flow.UpdatedAt,
	}

	if flow.AssignedUser != nil {
		response.AssignedUsername = &flow.AssignedUser.Username
	}

	if flow.Creator != nil {
		response.CreatedByUsername = &flow.Creator.Username
	}

	if flow.BaseFlow != nil {
		response.BaseFlowName = &flow.BaseFlow.FlowName
	}

	if flow.Verifier != nil {
		response.VerifiedByUsername = &flow.Verifier.Username
	}

	if len(flow.FlowSteps) > 0 {
		response.FlowSteps = mapApprovalFlowStepsToResponse(flow.FlowSteps)
	}

	return response
}

func mapApprovalFlowsToResponse(flows []models.ApprovalFlow) []dto.ApprovalFlowResponse {
	response := make([]dto.ApprovalFlowResponse, len(flows))
	for i, flow := range flows {
		response[i] = mapApprovalFlowToResponse(flow)
	}
	return response
}

func mapApprovalFlowStepToResponse(step models.ApprovalFlowStep) dto.ApprovalFlowStepResponse {
	response := dto.ApprovalFlowStepResponse{
		ID:           step.ID,
		FlowID:       step.FlowID,
		StepOrder:    step.StepOrder,
		StepName:     step.StepName,
		StepRole:     step.StepRole,
		RoleID:       step.RoleID,
		BranchID:     step.BranchID,
		Structure:    step.Structure,
		IsRequired:   step.IsRequired,
		CanSkip:      step.CanSkip,
		IsVisible:    step.IsVisible,
		Type:         step.Type,
		Category:     step.Category,
		ApprovalWay:  step.ApprovalWay,
		AutoApprove:  step.AutoApprove,
		TimeoutHours: step.TimeoutHours,
		Conditions:   step.Conditions,
		CreatedAt:    step.CreatedAt,
		UpdatedAt:    step.UpdatedAt,
	}

	if step.Role != nil {
		response.RoleName = &step.Role.Name
	}

	if step.Branch != nil {
		response.BranchName = &step.Branch.BranchName
	}

	return response
}

func mapApprovalFlowStepsToResponse(steps []models.ApprovalFlowStep) []dto.ApprovalFlowStepResponse {
	response := make([]dto.ApprovalFlowStepResponse, len(steps))
	for i, step := range steps {
		response[i] = mapApprovalFlowStepToResponse(step)
	}
	return response
}

func mapTransactionApprovalToResponse(approval models.TransactionApproval) dto.TransactionApprovalResponse {
	response := dto.TransactionApprovalResponse{
		ID:                approval.ID,
		FlowID:            approval.FlowID,
		FlowStepID:        approval.FlowStepID,
		TransactionNumber: approval.TransactionNumber,
		TransactionType:   approval.TransactionType,
		ApproverUserID:    approval.ApproverUserID,
		ApproverRoleID:    approval.ApproverRoleID,
		Status:            approval.Status,
		StatusView:        approval.StatusView,
		ApprovedAt:        approval.ApprovedAt,
		ApprovedBy:        approval.ApprovedBy,
		RejectedAt:        approval.RejectedAt,
		RejectedBy:        approval.RejectedBy,
		Notes:             approval.Notes,
		Metadata:          approval.Metadata,
		CreatedAt:         approval.CreatedAt,
		UpdatedAt:         approval.UpdatedAt,
	}

	if approval.ApproverUser != nil {
		response.ApproverUsername = &approval.ApproverUser.Username
	}

	if approval.ApproverRole != nil {
		response.ApproverRoleName = &approval.ApproverRole.Name
	}

	// Nama yang ditampilkan memakai fullname, sama dengan created_by_name di
	// detail transaksi. Username dipakai kalau fullname kosong.
	if approval.ActualApprover != nil {
		response.ApprovedByName = userDisplayName(approval.ActualApprover)
	}

	if approval.ActualRejecter != nil {
		response.RejectedByName = userDisplayName(approval.ActualRejecter)
	}

	if approval.ApprovalFlowStep != nil {
		step := mapApprovalFlowStepToResponse(*approval.ApprovalFlowStep)
		response.FlowStep = &step
	}

	return response
}

func mapTransactionApprovalsToResponse(approvals []models.TransactionApproval) []dto.TransactionApprovalResponse {
	response := make([]dto.TransactionApprovalResponse, len(approvals))
	for i, approval := range approvals {
		response[i] = mapTransactionApprovalToResponse(approval)
	}
	return response
}

// func validateApproverBranch(approverUserID, transactionNumber, transactionType string) error {
// 	// Ambil branch homebase approver
// 	approverHomebase, err := GetUserActiveHomebase(approverUserID)
// 	if err != nil {
// 		return fmt.Errorf("failed to get approver homebase: %w", err)
// 	}

// 	// Ambil branch creator dari transaksi
// 	var transaction models.Transaction
// 	if err := config.DB.
// 		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, transactionType).
// 		First(&transaction).Error; err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return errors.New("transaction not found")
// 		}
// 		return err
// 	}

// 	// Ambil branch homebase creator transaksi
// 	creatorHomebase, err := GetUserActiveHomebase(transaction.CreatedBy)
// 	if err != nil {
// 		return fmt.Errorf("failed to get transaction creator homebase: %w", err)
// 	}

// 	if approverHomebase.Branch.BranchCode != creatorHomebase.Branch.BranchCode {
// 		return fmt.Errorf("you can only approve transactions from your branch (%s)", approverHomebase.Branch.BranchCode)
// 	}

// 	return nil
// }

func validateApproverBranch(approverUserID, transactionNumber, transactionType string) error {
	// Agreement menggabungkan transaksi dari banyak cabang, jadi tidak ada
	// satu "cabang pembuat" untuk dibandingkan. Approver wajib punya akses ke
	// SELURUH cabang anggotanya.
	if transactionType == TxDisposalAgreement {
		return validateAgreementApproverBranches(approverUserID, transactionNumber)
	}

	// Ambil transaksi untuk dapat CreatedBy
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, transactionType).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("transaction not found")
		}
		return err
	}

	// Ambil homebase creator transaksi
	creatorHomebase, err := GetUserActiveHomebase(transaction.CreatedBy)
	if err != nil {
		return fmt.Errorf("failed to get transaction creator homebase: %w", err)
	}
	creatorBranchCode := creatorHomebase.Branch.BranchCode

	// Ambil semua branch yang dimiliki approver (semua tipe: homebase, temporary, assignment)
	var approverBranches []models.UserBranch
	if err := config.DB.
		Preload("Branch").
		Where("user_id = ?", approverUserID).
		Find(&approverBranches).Error; err != nil {
		return fmt.Errorf("failed to get approver branches: %w", err)
	}

	// Cek apakah branch creator ada di list branch approver
	for _, ub := range approverBranches {
		if ub.Branch != nil && ub.Branch.BranchCode == creatorBranchCode {
			return nil // valid
		}
	}

	return fmt.Errorf("you do not have access to approve transactions from branch %s", creatorBranchCode)
}

// autoCompleteProcurementApproval auto-trigger complete approval
// jika semua step sudah approved
func autoCompleteProcurementApproval(userID, transactionNumber, transactionType string) error {
	// Hanya untuk procurement
	if transactionType != "procurement" {
		return nil
	}

	// Cek status semua approval step
	var total, approved int64
	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, transactionType).
		Count(&total)

	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ? AND status = ?", transactionNumber, transactionType, "approved").
		Count(&approved)

	if total == 0 || approved < total {
		return nil // belum semua approved
	}

	// Semua approved — transisi ke PROSES_BUDGET
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return err
	}

	if transaction.CurrentStage != models.StageApproval {
		return nil // sudah bukan di stage APPROVAL, skip
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
		return err
	}

	if err := recordStage(tx, transaction.ID, transactionNumber,
		fromStage, models.StageProcessBudget,
		models.ActionApprove, userID, nil, nil); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// autoRejectTransaction auto-reject transaksi ketika salah satu approval step di-reject
func autoRejectTransaction(userID, transactionNumber, transactionType, notes string) error {
	// Hanya untuk procurement
	if transactionType != "procurement" {
		return nil
	}

	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return err
	}

	// Kalau sudah REJECTED atau FINISHED, skip
	if transaction.CurrentStage == models.StageRejected ||
		transaction.CurrentStage == models.StageFinished {
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

// userDisplayName mengembalikan nama yang layak ditampilkan untuk satu user.
func userDisplayName(user *models.User) *string {
	if user == nil {
		return nil
	}
	if name := strings.TrimSpace(user.Fullname); name != "" {
		return &name
	}
	return &user.Username
}
