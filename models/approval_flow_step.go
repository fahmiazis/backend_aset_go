package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ApprovalFlowStep defines each step in an approval flow
// Example: Step 1 = Creator, Step 2 = Reviewer, Step 3 = Approver, Step 4 = Receiver
type ApprovalFlowStep struct {
	ID           string         `gorm:"type:char(36);primaryKey" json:"id"`
	FlowID       string         `gorm:"type:char(36);not null;index" json:"flow_id"`
	StepOrder    int            `gorm:"type:int;not null" json:"step_order"`                                             // urutan step (1, 2, 3, ...)
	StepName     string         `gorm:"type:varchar(100);not null" json:"step_name"`                                     // e.g., "Creator", "Reviewer", "Approver"
	StepRole     string         `gorm:"type:enum('creator','reviewer','approver','receiver');not null" json:"step_role"` // role dalam approval
	RoleID       *string        `gorm:"type:char(36)" json:"role_id"`                                                    // role yang bisa approve di step ini (optional, bisa juga by user)
	BranchID     *string        `gorm:"type:char(36)" json:"branch_id"`                                                  // untuk kasus approval lintas branch (misal manager penerima/pengirim)
	Structure    *string        `gorm:"type:varchar(100)" json:"structure"`                                              // struktur khusus, misal "sender_manager" atau "receiver_manager"
	IsRequired   bool           `gorm:"type:boolean;default:true" json:"is_required"`                                    // apakah step ini wajib?
	CanSkip      bool           `gorm:"type:boolean;default:false" json:"can_skip"`                                      // apakah bisa di-skip?
	IsVisible    bool           `gorm:"type:boolean;default:true" json:"is_visible"`                                     // apakah step ini visible untuk user tertentu (status_view)
	Type         string         `gorm:"type:enum('it','non-it','all');default:'all'" json:"type"`
	Category     string         `gorm:"type:enum('budget','non-budget','return','all');default:'all'" json:"category"`
	ApprovalWay  string         `gorm:"type:enum('web', 'upload');default:'web'" json:"approval_way"`
	AutoApprove  bool           `gorm:"type:boolean;default:false" json:"auto_approve"` // auto approve jika kondisi terpenuhi
	TimeoutHours *int           `gorm:"type:int" json:"timeout_hours"`                  // timeout dalam jam (optional)
	Conditions   *string        `gorm:"type:json" json:"conditions"`                    // kondisi untuk conditional approval (JSON)
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	ApprovalFlow ApprovalFlow `gorm:"foreignKey:FlowID" json:"approval_flow,omitempty"`
	Role         *Role        `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	Branch       *Branch      `gorm:"foreignKey:BranchID" json:"branch,omitempty"`
}

func (afs *ApprovalFlowStep) BeforeCreate(tx *gorm.DB) error {
	if afs.ID == "" {
		afs.ID = uuid.New().String()
	}
	return nil
}

func (ApprovalFlowStep) TableName() string {
	return "approval_flow_steps"
}
