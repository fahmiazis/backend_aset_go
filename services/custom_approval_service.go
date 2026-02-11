package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ============================================================================
// CUSTOM APPROVAL - Using existing approval_flows table
// ============================================================================

// CreateCustomApproval - User create custom approval (stored in approval_flows with is_custom=true)
func CreateCustomApproval(userID string, req dto.CreateCustomApprovalRequest) (*dto.ApprovalFlowResponse, error) {
	// 1. Validate base flow exists
	var baseFlow models.ApprovalFlow
	if err := config.DB.First(&baseFlow, "id = ?", req.BaseFlowID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("base approval flow not found")
		}
		return nil, err
	}

	// 2. Check if user can customize this flow
	canCustomize, err := CanUserCustomizeFlow(userID, req.BaseFlowID)
	if err != nil {
		return nil, err
	}
	if !canCustomize {
		return nil, errors.New("you don't have permission to customize this flow")
	}

	// 3. Check if user already has custom approval for this base flow
	var existingCustom models.ApprovalFlow
	err = config.DB.Where("is_custom = ? AND created_by = ? AND base_flow_id = ? AND deleted_at IS NULL",
		true, userID, req.BaseFlowID).
		First(&existingCustom).Error

	if err == nil {
		return nil, errors.New("you already have a custom approval based on this flow. Please update it instead")
	}

	// 4. Create custom approval flow
	customStatus := "pending_verification"
	customFlow := models.ApprovalFlow{
		FlowCode:       fmt.Sprintf("CUSTOM_%s_%s", userID[:8], baseFlow.FlowCode),
		FlowName:       req.CustomFlowName,
		ApprovalWay:    baseFlow.ApprovalWay, // Copy from base
		AssignmentType: "user_specific",      // Always user_specific for custom
		AssignedUserID: &userID,              // Assign to creator
		IsCustomizable: false,                // Custom flows cannot be customized further
		IsCustom:       true,                 // Mark as custom
		CreatedBy:      &userID,
		BaseFlowID:     &req.BaseFlowID,
		CustomStatus:   &customStatus,
		Description:    fmt.Sprintf("Custom approval created by user based on %s", baseFlow.FlowName),
		IsActive:       false, // Will be active after verification
	}

	if err := config.DB.Create(&customFlow).Error; err != nil {
		return nil, err
	}

	// 5. Create custom steps (using existing approval_flow_steps table)
	for _, stepReq := range req.Steps {
		// Set defaults if not provided
		stepType := "all"
		stepCategory := "all"
		stepApprovalWay := "web"

		if stepReq.Type != "" {
			stepType = stepReq.Type
		}
		if stepReq.Category != "" {
			stepCategory = stepReq.Category
		}
		if stepReq.ApprovalWay != "" {
			stepApprovalWay = stepReq.ApprovalWay
		}

		step := models.ApprovalFlowStep{
			FlowID:       customFlow.ID,
			StepOrder:    stepReq.StepOrder,
			StepName:     stepReq.StepName,
			StepRole:     stepReq.StepRole,
			RoleID:       stepReq.RoleID,
			BranchID:     stepReq.BranchID,
			Structure:    stepReq.Structure,
			IsRequired:   stepReq.IsRequired,
			CanSkip:      stepReq.CanSkip,
			IsVisible:    stepReq.IsVisible,
			Type:         stepType,
			Category:     stepCategory,
			ApprovalWay:  stepApprovalWay,
			AutoApprove:  stepReq.AutoApprove,
			TimeoutHours: stepReq.TimeoutHours,
			Conditions:   stepReq.Conditions,
		}

		if err := config.DB.Create(&step).Error; err != nil {
			return nil, err
		}
	}

	// 6. Return response
	return GetApprovalFlowByID(customFlow.ID)
}

// GetUserCustomApprovals - Get all custom approvals created by user
func GetUserCustomApprovals(userID string) ([]dto.ApprovalFlowResponse, error) {
	var customFlows []models.ApprovalFlow

	if err := config.DB.
		Preload("BaseFlow").
		Preload("Creator").
		Preload("Verifier").
		Preload("FlowSteps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_order ASC")
		}).
		Preload("FlowSteps.Role").
		Preload("FlowSteps.Branch").
		Where("is_custom = ? AND created_by = ?", true, userID).
		Find(&customFlows).Error; err != nil {
		return nil, err
	}

	return mapApprovalFlowsToResponse(customFlows), nil
}

// UpdateCustomApproval - Update custom approval
func UpdateCustomApproval(userID, flowID string, req dto.UpdateCustomApprovalRequest) (*dto.ApprovalFlowResponse, error) {
	// 1. Get custom flow
	var customFlow models.ApprovalFlow
	if err := config.DB.First(&customFlow, "id = ?", flowID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("custom approval not found")
		}
		return nil, err
	}

	// 2. Validate is custom
	if !customFlow.IsCustom {
		return nil, errors.New("this is not a custom approval flow")
	}

	// 3. Check ownership
	if customFlow.CreatedBy == nil || *customFlow.CreatedBy != userID {
		// Check if user is admin
		var userRoles []models.UserRole
		config.DB.Preload("Role").Where("user_id = ?", userID).Find(&userRoles)

		isAdmin := false
		for _, ur := range userRoles {
			if ur.Role != nil && ur.Role.Name == "admin" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			return nil, errors.New("you can only edit your own custom approval")
		}
	}

	// 4. Update flow
	pendingStatus := "pending_verification"
	updates := map[string]interface{}{
		"flow_name":     req.CustomFlowName,
		"custom_status": pendingStatus,
		"is_active":     false, // Reset to inactive, needs re-verification
	}

	if err := config.DB.Model(&customFlow).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 5. Delete old steps
	if err := config.DB.Where("flow_id = ?", flowID).Delete(&models.ApprovalFlowStep{}).Error; err != nil {
		return nil, err
	}

	// 6. Create new steps
	for _, stepReq := range req.Steps {
		// Set defaults if not provided
		stepType := "all"
		stepCategory := "all"
		stepApprovalWay := "web"

		if stepReq.Type != nil && *stepReq.Type != "" {
			stepType = *stepReq.Type
		}
		if stepReq.Category != nil && *stepReq.Category != "" {
			stepCategory = *stepReq.Category
		}
		if stepReq.ApprovalWay != nil && *stepReq.ApprovalWay != "" {
			stepApprovalWay = *stepReq.ApprovalWay
		}

		step := models.ApprovalFlowStep{
			FlowID:       flowID,
			StepOrder:    *stepReq.StepOrder,
			StepName:     stepReq.StepName,
			StepRole:     stepReq.StepRole,
			RoleID:       stepReq.RoleID,
			BranchID:     stepReq.BranchID,
			Structure:    stepReq.Structure,
			IsRequired:   *stepReq.IsRequired,
			CanSkip:      *stepReq.CanSkip,
			IsVisible:    *stepReq.IsVisible,
			Type:         stepType,
			Category:     stepCategory,
			ApprovalWay:  stepApprovalWay,
			AutoApprove:  *stepReq.AutoApprove,
			TimeoutHours: stepReq.TimeoutHours,
			Conditions:   stepReq.Conditions,
		}

		if err := config.DB.Create(&step).Error; err != nil {
			return nil, err
		}
	}

	return GetApprovalFlowByID(flowID)
}

// DeleteCustomApproval - Delete custom approval (soft delete)
func DeleteCustomApproval(userID, flowID string) error {
	var customFlow models.ApprovalFlow
	if err := config.DB.First(&customFlow, "id = ?", flowID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("custom approval not found")
		}
		return err
	}

	// Validate is custom
	if !customFlow.IsCustom {
		return errors.New("this is not a custom approval flow")
	}

	// Check ownership
	if customFlow.CreatedBy == nil || *customFlow.CreatedBy != userID {
		return errors.New("you can only delete your own custom approval")
	}

	return config.DB.Delete(&customFlow).Error
}

// ============================================================================
// VERIFICATION (Tim Asset)
// ============================================================================

// VerifyCustomApproval - Tim asset verify/reject custom approval
func VerifyCustomApproval(verifierID, flowID string, req dto.VerifyCustomApprovalRequest) error {
	// 1. Get custom flow
	var customFlow models.ApprovalFlow
	if err := config.DB.First(&customFlow, "id = ?", flowID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("custom approval not found")
		}
		return err
	}

	// 2. Validate is custom
	if !customFlow.IsCustom {
		return errors.New("this is not a custom approval flow")
	}

	// 3. Check if already verified
	if customFlow.CustomStatus != nil && (*customFlow.CustomStatus == "approved" || *customFlow.CustomStatus == "rejected") {
		return errors.New("custom approval already verified")
	}

	// 4. Verify action
	now := gorm.Expr("NOW()")
	updates := map[string]interface{}{
		"verified_by": verifierID,
		"verified_at": now,
	}

	if req.Action == "approve" {
		approved := "approved"
		updates["custom_status"] = approved
		updates["is_active"] = true
		if req.Notes != nil {
			updates["verification_notes"] = *req.Notes
		}
	} else if req.Action == "reject" {
		rejected := "rejected"
		updates["custom_status"] = rejected
		updates["is_active"] = false
		if req.Notes != nil {
			updates["rejection_reason"] = *req.Notes
		}
	}

	return config.DB.Model(&customFlow).Updates(updates).Error
}

// GetPendingVerifications - Get all custom approvals pending verification
func GetPendingVerifications() ([]dto.ApprovalFlowResponse, error) {
	var customFlows []models.ApprovalFlow

	if err := config.DB.
		Preload("Creator").
		Preload("BaseFlow").
		Preload("FlowSteps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_order ASC")
		}).
		Preload("FlowSteps.Role").
		Preload("FlowSteps.Branch").
		Where("is_custom = ? AND custom_status = ?", true, "pending_verification").
		Find(&customFlows).Error; err != nil {
		return nil, err
	}

	return mapApprovalFlowsToResponse(customFlows), nil
}

// ============================================================================
// INTEGRATION - Get Flow for User (with fallback)
// ============================================================================

// GetApprovalFlowForUser - Get approval flow for user (custom > user-specific > general)
func GetApprovalFlowForUser(userID, transactionType string) (*dto.ApprovalFlowResponse, error) {
	// 1. Check if user has approved custom approval
	var customFlow models.ApprovalFlow
	err := config.DB.
		Preload("FlowSteps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_order ASC")
		}).
		Preload("FlowSteps.Role").
		Preload("FlowSteps.Branch").
		Where("is_custom = ? AND created_by = ? AND custom_status = ? AND is_active = ?",
			true, userID, "approved", true).
		First(&customFlow).Error

	if err == nil {
		// User has approved custom flow
		response := mapApprovalFlowToResponse(customFlow)
		return &response, nil
	}

	// 2. Check for user-specific flow (assigned by admin)
	var userSpecificFlow models.ApprovalFlow
	err = config.DB.
		Preload("FlowSteps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_order ASC")
		}).
		Preload("FlowSteps.Role").
		Preload("FlowSteps.Branch").
		Where("is_custom = ? AND assignment_type = ? AND assigned_user_id = ? AND is_active = ?",
			false, "user_specific", userID, true).
		First(&userSpecificFlow).Error

	if err == nil {
		response := mapApprovalFlowToResponse(userSpecificFlow)
		return &response, nil
	}

	// 3. Fallback to general flow
	var generalFlow models.ApprovalFlow
	err = config.DB.
		Preload("FlowSteps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_order ASC")
		}).
		Preload("FlowSteps.Role").
		Preload("FlowSteps.Branch").
		Where("is_custom = ? AND assignment_type = ? AND is_active = ?",
			false, "general", true).
		First(&generalFlow).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("no approval flow found for this transaction type")
		}
		return nil, err
	}

	response := mapApprovalFlowToResponse(generalFlow)
	return &response, nil
}

// ============================================================================
// HELPER - Check Permission
// ============================================================================

// CanUserCustomizeFlow - Check if user can customize a flow
func CanUserCustomizeFlow(userID, flowID string) (bool, error) {
	// Get flow
	var flow models.ApprovalFlow
	if err := config.DB.First(&flow, "id = ?", flowID).Error; err != nil {
		return false, err
	}

	// Custom flows cannot be customized further
	if flow.IsCustom {
		return false, errors.New("custom flows cannot be customized")
	}

	// Check if flow is customizable
	if !flow.IsCustomizable {
		return false, nil
	}

	// If allowed_creator_roles is null or empty, everyone can customize
	if flow.AllowedCreatorRoles == nil || *flow.AllowedCreatorRoles == "" || *flow.AllowedCreatorRoles == "[]" {
		return true, nil
	}

	// Parse allowed roles
	var allowedRoleIDs []string
	if err := json.Unmarshal([]byte(*flow.AllowedCreatorRoles), &allowedRoleIDs); err != nil {
		return false, err
	}

	// Get user roles
	var userRoles []models.UserRole
	config.DB.Where("user_id = ?", userID).Find(&userRoles)

	// Check if user has allowed role
	for _, ur := range userRoles {
		for _, allowedRoleID := range allowedRoleIDs {
			if ur.RoleID == allowedRoleID {
				return true, nil
			}
		}
	}

	return false, nil
}
