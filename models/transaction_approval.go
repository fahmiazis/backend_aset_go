package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TransactionApproval tracks approval status for each transaction
// One transaction can have multiple approval records (one per step)
type TransactionApproval struct {
	ID                string         `gorm:"type:char(36);primaryKey" json:"id"`
	FlowID            string         `gorm:"type:char(36);not null;index" json:"flow_id"`
	FlowStepID        string         `gorm:"type:char(36);not null;index" json:"flow_step_id"`
	TransactionNumber string         `gorm:"type:varchar(100);not null;index" json:"transaction_number"` // no_doc, no_pengadaan, no_set → digabung jadi transaction_number
	TransactionType   string         `gorm:"type:varchar(50);not null" json:"transaction_type"`          // misal: "purchase_request", "transfer", "mutation"
	ApproverUserID    *string        `gorm:"type:char(36);index" json:"approver_user_id"`                // user yang harus approve
	ApproverRoleID    *string        `gorm:"type:char(36)" json:"approver_role_id"`                      // role yang bisa approve (jika tidak spesifik user)
	Status            string         `gorm:"type:enum('pending','approved','rejected','skipped');default:'pending'" json:"status"`
	StatusView        string         `gorm:"type:enum('visible','hidden');default:'visible'" json:"status_view"` // apakah approval ini ditampilkan ke user tertentu
	ApprovedAt        *time.Time     `json:"approved_at"`
	ApprovedBy        *string        `gorm:"type:char(36)" json:"approved_by"` // user yang actually melakukan approve
	RejectedAt        *time.Time     `json:"rejected_at"`
	RejectedBy        *string        `gorm:"type:char(36)" json:"rejected_by"`
	Notes             *string        `gorm:"type:text" json:"notes"`    // catatan dari approver
	Metadata          *string        `gorm:"type:json" json:"metadata"` // data tambahan (JSON)
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	ApprovalFlow     ApprovalFlow      `gorm:"foreignKey:FlowID" json:"approval_flow,omitempty"`
	ApprovalFlowStep *ApprovalFlowStep `gorm:"foreignKey:FlowStepID" json:"approval_flow_step,omitempty"`
	ApproverUser     *User             `gorm:"foreignKey:ApproverUserID" json:"approver_user,omitempty"`
	ApproverRole     *Role             `gorm:"foreignKey:ApproverRoleID" json:"approver_role,omitempty"`
	ActualApprover   *User             `gorm:"foreignKey:ApprovedBy" json:"actual_approver,omitempty"`
	ActualRejecter   *User             `gorm:"foreignKey:RejectedBy" json:"actual_rejecter,omitempty"`
}

func (ta *TransactionApproval) BeforeCreate(tx *gorm.DB) error {
	if ta.ID == "" {
		ta.ID = uuid.New().String()
	}
	return nil
}

func (TransactionApproval) TableName() string {
	return "transaction_approvals"
}
