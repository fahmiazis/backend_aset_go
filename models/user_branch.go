package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserBranch struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    string    `gorm:"type:char(36);not null;index:idx_user_branch" json:"user_id"`
	BranchID  string    `gorm:"type:char(36);not null;index:idx_user_branch" json:"branch_id"`
	CreatedAt time.Time `json:"created_at"`

	// Relations
	User   User   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Branch Branch `gorm:"foreignKey:BranchID;constraint:OnDelete:CASCADE" json:"branch,omitempty"`
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
