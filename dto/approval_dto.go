package dto

import "time"

// ============================================================================
// APPROVAL FLOW DTOs
// ============================================================================

type CreateApprovalFlowRequest struct {
	FlowCode            string   `json:"flow_code" binding:"required,min=2,max=50"`
	FlowName            string   `json:"flow_name" binding:"required,min=2,max=100"`
	ApprovalWay         string   `json:"approval_way" binding:"required,oneof=sequential parallel conditional"`
	AssignmentType      string   `json:"assignment_type" binding:"required,oneof=general user_specific"`
	AssignedUserID      *string  `json:"assigned_user_id"` // Required if assignment_type = user_specific
	IsCustomizable      bool     `json:"is_customizable"`
	AllowedCreatorRoles []string `json:"allowed_creator_roles"` // Array of role IDs
	Description         string   `json:"description"`
	IsActive            bool     `json:"is_active"`

	// Custom flow specific (optional, only for user creating custom)
	IsCustom   bool    `json:"is_custom"`    // Set true if user creating custom
	BaseFlowID *string `json:"base_flow_id"` // Required if is_custom = true
}

type UpdateApprovalFlowRequest struct {
	FlowCode            string   `json:"flow_code" binding:"omitempty,min=2,max=50"`
	FlowName            string   `json:"flow_name" binding:"omitempty,min=2,max=100"`
	ApprovalWay         string   `json:"approval_way" binding:"omitempty,oneof=sequential parallel conditional"`
	AssignmentType      string   `json:"assignment_type" binding:"omitempty,oneof=general user_specific"`
	AssignedUserID      *string  `json:"assigned_user_id"`
	IsCustomizable      *bool    `json:"is_customizable"`
	AllowedCreatorRoles []string `json:"allowed_creator_roles"`
	Description         string   `json:"description"`
	IsActive            *bool    `json:"is_active"`
}

type ApprovalFlowResponse struct {
	ID                  string  `json:"id"`
	FlowCode            string  `json:"flow_code"`
	FlowName            string  `json:"flow_name"`
	ApprovalWay         string  `json:"approval_way"`
	AssignmentType      string  `json:"assignment_type"`
	AssignedUserID      *string `json:"assigned_user_id"`
	AssignedUsername    *string `json:"assigned_username,omitempty"`
	IsCustomizable      bool    `json:"is_customizable"`
	AllowedCreatorRoles *string `json:"allowed_creator_roles,omitempty"` // JSON string

	// Custom flow fields
	IsCustom           bool       `json:"is_custom"`
	CreatedBy          *string    `json:"created_by"`
	CreatedByUsername  *string    `json:"created_by_username,omitempty"`
	BaseFlowID         *string    `json:"base_flow_id"`
	BaseFlowName       *string    `json:"base_flow_name,omitempty"`
	CustomStatus       *string    `json:"custom_status"`
	VerifiedBy         *string    `json:"verified_by"`
	VerifiedByUsername *string    `json:"verified_by_username,omitempty"`
	VerifiedAt         *time.Time `json:"verified_at"`
	VerificationNotes  *string    `json:"verification_notes"`
	RejectionReason    *string    `json:"rejection_reason"`

	Description string                     `json:"description"`
	IsActive    bool                       `json:"is_active"`
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
	FlowSteps   []ApprovalFlowStepResponse `json:"flow_steps,omitempty"`
}

// ============================================================================
// APPROVAL FLOW STEP DTOs
// ============================================================================

type CreateApprovalFlowStepRequest struct {
	FlowID       string  `json:"flow_id" binding:"required"`
	StepOrder    int     `json:"step_order" binding:"required,min=1"`
	StepName     string  `json:"step_name" binding:"required,min=2,max=100"`
	StepRole     string  `json:"step_role" binding:"required,oneof=creator reviewer approver receiver"`
	RoleID       *string `json:"role_id"`
	BranchID     *string `json:"branch_id"`
	Structure    *string `json:"structure"`
	IsRequired   bool    `json:"is_required"`
	CanSkip      bool    `json:"can_skip"`
	IsVisible    bool    `json:"is_visible"`
	Type         string  `json:"type" binding:"omitempty,oneof=it non-it all"`                    // it, non-it, all
	Category     string  `json:"category" binding:"omitempty,oneof=budget non-budget return all"` // budget, non-budget, return, all
	ApprovalWay  string  `json:"approval_way" binding:"omitempty,oneof=web upload"`               // web, upload
	AutoApprove  bool    `json:"auto_approve"`
	TimeoutHours *int    `json:"timeout_hours"`
	Conditions   *string `json:"conditions"`
}

type UpdateApprovalFlowStepRequest struct {
	StepOrder    *int    `json:"step_order" binding:"omitempty,min=1"`
	StepName     string  `json:"step_name" binding:"omitempty,min=2,max=100"`
	StepRole     string  `json:"step_role" binding:"omitempty,oneof=creator reviewer approver receiver"`
	RoleID       *string `json:"role_id"`
	BranchID     *string `json:"branch_id"`
	Structure    *string `json:"structure"`
	IsRequired   *bool   `json:"is_required"`
	CanSkip      *bool   `json:"can_skip"`
	IsVisible    *bool   `json:"is_visible"`
	Type         *string `json:"type" binding:"omitempty,oneof=it non-it all"`
	Category     *string `json:"category" binding:"omitempty,oneof=budget non-budget return all"`
	ApprovalWay  *string `json:"approval_way" binding:"omitempty,oneof=web upload"`
	AutoApprove  *bool   `json:"auto_approve"`
	TimeoutHours *int    `json:"timeout_hours"`
	Conditions   *string `json:"conditions"`
}

type UpdateBulkStepOrderFlowStep struct {
	ListIDs []string `json:"list_ids" binding:"required,min=1"`
}

type ApprovalFlowStepResponse struct {
	ID           string    `json:"id"`
	FlowID       string    `json:"flow_id"`
	StepOrder    int       `json:"step_order"`
	StepName     string    `json:"step_name"`
	StepRole     string    `json:"step_role"`
	RoleID       *string   `json:"role_id"`
	RoleName     *string   `json:"role_name,omitempty"`
	BranchID     *string   `json:"branch_id"`
	BranchName   *string   `json:"branch_name,omitempty"`
	Structure    *string   `json:"structure"`
	IsRequired   bool      `json:"is_required"`
	CanSkip      bool      `json:"can_skip"`
	IsVisible    bool      `json:"is_visible"`
	Type         string    `json:"type"`
	Category     string    `json:"category"`
	ApprovalWay  string    `json:"approval_way"`
	AutoApprove  bool      `json:"auto_approve"`
	TimeoutHours *int      `json:"timeout_hours"`
	Conditions   *string   `json:"conditions"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ============================================================================
// TRANSACTION APPROVAL DTOs
// ============================================================================

type CreateTransactionApprovalRequest struct {
	FlowID            string  `json:"flow_id" binding:"required"`
	TransactionNumber string  `json:"transaction_number" binding:"required"`
	TransactionType   string  `json:"transaction_type" binding:"required"`
	Metadata          *string `json:"metadata"`
}

type ApproveTransactionRequest struct {
	TransactionApprovalID string  `json:"transaction_approval_id" binding:"required"`
	Notes                 *string `json:"notes"`
}

type RejectTransactionRequest struct {
	TransactionApprovalID string  `json:"transaction_approval_id" binding:"required"`
	Notes                 *string `json:"notes"`
}

type TransactionApprovalResponse struct {
	ID                string     `json:"id"`
	FlowID            string     `json:"flow_id"`
	FlowStepID        string     `json:"flow_step_id"`
	TransactionNumber string     `json:"transaction_number"`
	TransactionType   string     `json:"transaction_type"`
	ApproverUserID    *string    `json:"approver_user_id"`
	ApproverUsername  *string    `json:"approver_username,omitempty"`
	ApproverRoleID    *string    `json:"approver_role_id"`
	ApproverRoleName  *string    `json:"approver_role_name,omitempty"`
	Status            string     `json:"status"`
	StatusView        string     `json:"status_view"`
	ApprovedAt        *time.Time `json:"approved_at"`
	ApprovedBy        *string    `json:"approved_by"`
	ApprovedByName    *string    `json:"approved_by_name,omitempty"`
	RejectedAt        *time.Time `json:"rejected_at"`
	RejectedBy        *string    `json:"rejected_by"`
	RejectedByName    *string    `json:"rejected_by_name,omitempty"`
	Notes             *string    `json:"notes"`
	Metadata          *string    `json:"metadata"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	// Nested objects
	FlowStep *ApprovalFlowStepResponse `json:"flow_step,omitempty"`
}

// ============================================================================
// APPROVAL SIGNATURE DTOs
// ============================================================================

type CreateApprovalSignatureRequest struct {
	TransactionNumber string  `json:"transaction_number" binding:"required"`
	TransactionType   string  `json:"transaction_type" binding:"required"`
	StepRole          string  `json:"step_role" binding:"required,oneof=creator reviewer approver receiver"`
	SignaturePath     *string `json:"signature_path"`
	Notes             *string `json:"notes"`
	Structure         *string `json:"structure"`
}

type ApprovalSignatureResponse struct {
	ID                string    `json:"id"`
	TransactionNumber string    `json:"transaction_number"`
	TransactionType   string    `json:"transaction_type"`
	UserID            string    `json:"user_id"`
	Username          string    `json:"username,omitempty"`
	RoleID            *string   `json:"role_id"`
	RoleName          *string   `json:"role_name,omitempty"`
	StepRole          string    `json:"step_role"`
	SignaturePath     *string   `json:"signature_path"`
	SignedAt          time.Time `json:"signed_at"`
	Status            string    `json:"status"`
	Notes             *string   `json:"notes"`
	Structure         *string   `json:"structure"`
	IsRecent          bool      `json:"is_recent"`
	CreatedAt         time.Time `json:"created_at"`
}

// ============================================================================
// APPROVAL DASHBOARD / LIST DTOs
// ============================================================================

type ApprovalListRequest struct {
	UserID   string  `json:"user_id"`
	Status   *string `json:"status"` // pending, approved, rejected
	Type     *string `json:"type"`
	Category *string `json:"category"`
	Page     int     `json:"page"`
	Limit    int     `json:"limit"`
}

type TransactionApprovalSummary struct {
	TransactionNumber string                        `json:"transaction_number"`
	TransactionType   string                        `json:"transaction_type"`
	TotalSteps        int                           `json:"total_steps"`
	CompletedSteps    int                           `json:"completed_steps"`
	CurrentStep       *ApprovalFlowStepResponse     `json:"current_step,omitempty"`
	Status            string                        `json:"status"` // pending, in_progress, approved, rejected
	Approvals         []TransactionApprovalResponse `json:"approvals"`
	CreatedAt         time.Time                     `json:"created_at"`
}

// ============================================================================
// CUSTOM APPROVAL SPECIFIC DTOs (Using existing approval_flows table)
// ============================================================================

type CreateCustomApprovalRequest struct {
	BaseFlowID      string                          `json:"base_flow_id" binding:"required"`
	TransactionType string                          `json:"transaction_type" binding:"required"` // Added for context
	CustomFlowName  string                          `json:"custom_flow_name" binding:"required"`
	Steps           []CreateApprovalFlowStepRequest `json:"steps" binding:"required,min=1"`
}

type UpdateCustomApprovalRequest struct {
	CustomFlowName string                          `json:"custom_flow_name"`
	Steps          []UpdateApprovalFlowStepRequest `json:"steps" binding:"required,min=1"`
}

type VerifyCustomApprovalRequest struct {
	Action string  `json:"action" binding:"required,oneof=approve reject"`
	Notes  *string `json:"notes"`
}
