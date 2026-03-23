package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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

	flow := models.ApprovalFlow{
		FlowCode:            req.FlowCode,
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

	// Check if approval already exists for this transaction
	var existingCount int64
	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ?", req.TransactionNumber, req.TransactionType).
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

		if err := config.DB.Create(&approval).Error; err != nil {
			return err
		}
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

	// Auto-trigger complete approval jika semua step sudah approved
	if err := autoCompleteProcurementApproval(userID, approval.TransactionNumber, approval.TransactionType); err != nil {
		// Log error tapi tidak return error — approval tetap berhasil
		// Complete approval bisa di-trigger manual jika auto gagal
		fmt.Printf("auto complete approval warning: %v\n", err)
	}

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

	notes := ""

	if req.Notes != nil {
		notes = *req.Notes
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

	// Auto-reject transaksi jika approval di-reject
	if err := autoRejectTransaction(userID, approval.TransactionNumber, approval.TransactionType, notes); err != nil {
		fmt.Printf("auto reject transaction warning: %v\n", err)
	}

	return nil
}

// GetTransactionApprovalStatus gets full approval status for a transaction
func GetTransactionApprovalStatus(transactionNumber, transactionType string) (*dto.TransactionApprovalSummary, error) {
	var approvals []models.TransactionApproval

	if err := config.DB.
		Preload("ApprovalFlowStep").
		Preload("ApproverUser").
		Preload("ApproverRole").
		Preload("ActualApprover").
		Preload("ActualRejecter").
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, transactionType).
		Find(&approvals).Error; err != nil {
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

	if approval.ActualApprover != nil {
		response.ApprovedByName = &approval.ActualApprover.Username
	}

	if approval.ActualRejecter != nil {
		response.RejectedByName = &approval.ActualRejecter.Username
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
	// Ambil transaksi
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, transactionType).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("transaction not found")
		}
		return err
	}

	// Ambil homebase creator (pakai function lu yang udah ada)
	creatorHomebase, err := GetUserActiveHomebase(transaction.CreatedBy)
	if err != nil {
		return fmt.Errorf("failed to get creator homebase: %w", err)
	}

	// Ambil semua branch approver
	approverBranches, err := GetUserBranchs(approverUserID)
	if err != nil {
		return fmt.Errorf("failed to get approver branches: %w", err)
	}

	// Cek apakah branch creator ada di branch approver
	for _, branch := range approverBranches {
		if branch.BranchCode == creatorHomebase.Branch.BranchCode {
			return nil
		}
	}

	return fmt.Errorf(
		"you can only approve transactions from branch %s",
		creatorHomebase.Branch.BranchCode,
	)
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

	// Kalau sudah REJECTED atau SELESAI, skip
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
