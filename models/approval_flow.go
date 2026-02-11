package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ApprovalFlow defines the approval workflow template
// Can be:
// - Master template (is_custom = false, created by admin)
// - User custom flow (is_custom = true, created by user, needs verification)
type ApprovalFlow struct {
	ID          string `gorm:"type:char(36);primaryKey" json:"id"`
	FlowCode    string `gorm:"type:varchar(50);uniqueIndex;not null" json:"flow_code"`
	FlowName    string `gorm:"type:varchar(100);not null" json:"flow_name"`
	ApprovalWay string `gorm:"type:enum('sequential','parallel','conditional');default:'sequential'" json:"approval_way"`

	// Assignment System
	AssignmentType string  `gorm:"type:enum('general','user_specific');default:'general'" json:"assignment_type"` // general: all users, user_specific: specific user
	AssignedUserID *string `gorm:"type:char(36)" json:"assigned_user_id"`                                         // If user_specific, which user

	// Customization Control
	IsCustomizable      bool    `gorm:"type:boolean;default:false" json:"is_customizable"` // Can users customize this flow?
	AllowedCreatorRoles *string `gorm:"type:json" json:"allowed_creator_roles"`            // JSON array of role IDs allowed to customize

	// Custom Flow Tracking
	IsCustom     bool    `gorm:"type:boolean;default:false" json:"is_custom"`                                          // Is this a custom flow created by user?
	CreatedBy    *string `gorm:"type:char(36)" json:"created_by"`                                                      // User who created this (NULL if admin/system)
	BaseFlowID   *string `gorm:"type:char(36)" json:"base_flow_id"`                                                    // Reference to master flow if custom
	CustomStatus *string `gorm:"type:enum('draft','pending_verification','approved','rejected')" json:"custom_status"` // Status if custom

	// Verification (for custom flows)
	VerifiedBy        *string    `gorm:"type:char(36)" json:"verified_by"`
	VerifiedAt        *time.Time `json:"verified_at"`
	VerificationNotes *string    `gorm:"type:text" json:"verification_notes"`
	RejectionReason   *string    `gorm:"type:text" json:"rejection_reason"`

	// Common
	Description string         `gorm:"type:text" json:"description"`
	IsActive    bool           `gorm:"type:boolean;default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	FlowSteps    []ApprovalFlowStep `gorm:"foreignKey:FlowID" json:"flow_steps,omitempty"`
	AssignedUser *User              `gorm:"foreignKey:AssignedUserID" json:"assigned_user,omitempty"`
	Creator      *User              `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	BaseFlow     *ApprovalFlow      `gorm:"foreignKey:BaseFlowID" json:"base_flow,omitempty"`
	Verifier     *User              `gorm:"foreignKey:VerifiedBy" json:"verifier,omitempty"`
}

func (af *ApprovalFlow) BeforeCreate(tx *gorm.DB) error {
	if af.ID == "" {
		af.ID = uuid.New().String()
	}
	return nil
}

func (ApprovalFlow) TableName() string {
	return "approval_flows"
}
