package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserBranch struct {
	ID         string    `gorm:"type:char(36);primaryKey" json:"id"`
	UserID     string    `gorm:"type:char(36);not null;index:idx_user_branch" json:"user_id"`
	BranchID   string    `gorm:"type:char(36);not null;index:idx_user_branch" json:"branch_id"`
	BranchType string    `gorm:"type:enum('homebase','temporary','assignment');default:'homebase'" json:"branch_type"` // homebase, temporary, assignment
	IsActive   bool      `gorm:"type:boolean;default:true" json:"is_active"`                                           // untuk handle multiple homebase saat login
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Relations
	User   *User   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Branch *Branch `gorm:"foreignKey:BranchID;constraint:OnDelete:CASCADE" json:"branch,omitempty"`
}

// BeforeCreate will set a UUID
func (uc *UserBranch) BeforeCreate(tx *gorm.DB) error {
	if uc.ID == "" {
		uc.ID = uuid.New().String()
	}
	return nil
}

// TableName overrides the table name
func (UserBranch) TableName() string {
	return "user_branchs"
}
