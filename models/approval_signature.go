package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ApprovalSignature stores digital signature / approval log
// This is similar to your 'ttd' table - tracking who signed what and when
type ApprovalSignature struct {
	ID                string         `gorm:"type:char(36);primaryKey" json:"id"`
	TransactionNumber string         `gorm:"type:varchar(100);not null;index" json:"transaction_number"`
	TransactionType   string         `gorm:"type:varchar(50);not null" json:"transaction_type"`
	UserID            string         `gorm:"type:char(36);not null;index" json:"user_id"`
	RoleID            *string        `gorm:"type:char(36)" json:"role_id"`
	StepRole          string         `gorm:"type:enum('creator','reviewer','approver','receiver');not null" json:"step_role"`
	SignaturePath     *string        `gorm:"type:varchar(255)" json:"signature_path"` // path ke file tanda tangan digital (optional)
	SignedAt          time.Time      `gorm:"not null" json:"signed_at"`
	Status            string         `gorm:"type:enum('signed','rejected');default:'signed'" json:"status"`
	Notes             *string        `gorm:"type:text" json:"notes"`
	IPAddress         *string        `gorm:"type:varchar(45)" json:"ip_address"`         // untuk audit trail
	UserAgent         *string        `gorm:"type:varchar(255)" json:"user_agent"`        // untuk audit trail
	Structure         *string        `gorm:"type:varchar(100)" json:"structure"`         // untuk kasus khusus seperti sender_manager/receiver_manager
	IsRecent          bool           `gorm:"type:boolean;default:true" json:"is_recent"` // marker untuk signature terbaru
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	User User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role *Role `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

func (as *ApprovalSignature) BeforeCreate(tx *gorm.DB) error {
	if as.ID == "" {
		as.ID = uuid.New().String()
	}
	return nil
}

func (ApprovalSignature) TableName() string {
	return "approval_signatures"
}
