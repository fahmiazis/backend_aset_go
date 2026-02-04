package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Branch struct {
	ID         string         `gorm:"type:char(36);primaryKey" json:"id"`
	BranchCode string         `gorm:"type:varchar(10);uniqueIndex;not null" json:"branch_code"`
	BranchName string         `gorm:"type:varchar(255);not null" json:"branch_name"`
	BranchType string         `gorm:"type:varchar(50);not null" json:"branch_type"`
	Status     string         `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	UserBranchs []UserBranch `gorm:"foreignKey:BranchID" json:"user_branchs,omitempty"`
}

// BeforeCreate will set a UUID rather than numeric ID
func (c *Branch) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}

	// Auto-generate branch_code if not provided
	if c.BranchCode == "" {
		var BranchCode string

		// Single atomic query: ambil max number dan increment dalam satu go
		// COALESCE handles the case when table is empty (returns 0)
		tx.Raw(`SELECT CONCAT('C', LPAD(COALESCE(MAX(CAST(SUBSTRING(branch_code, 2) AS UNSIGNED)), 0) + 1, 5, '0')) FROM branchs`).Scan(&BranchCode)

		c.BranchCode = BranchCode
	}

	return nil
}

// TableName overrides the table name
func (Branch) TableName() string {
	return "branchs"
}
