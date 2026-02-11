package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Reservoir tracks transaction numbers (used and unused)
type Reservoir struct {
	ID          string         `gorm:"type:char(36);primaryKey" json:"id"`
	NoTransaksi string         `gorm:"type:varchar(100);not null;uniqueIndex" json:"no_transaksi"`
	KodePlant   string         `gorm:"type:varchar(50)" json:"kode_plant"`                                    // branch_code
	Transaksi   string         `gorm:"type:varchar(50);not null" json:"transaksi"`                            // procurement, disposal, mutation, stock_opname
	Tipe        string         `gorm:"type:varchar(50)" json:"tipe"`                                          // area, etc
	Status      string         `gorm:"type:enum('delayed','used','expired');default:'delayed'" json:"status"` // delayed=reserved, used=submitted, expired=cancelled
	BranchID    *string        `gorm:"type:char(36)" json:"branch_id"`                                        // Reference to branch (homebase)
	UserID      *string        `gorm:"type:char(36)" json:"user_id"`                                          // User who created the transaction
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Branch *Branch `gorm:"foreignKey:BranchID" json:"branch,omitempty"`
	User   *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (r *Reservoir) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

func (Reservoir) TableName() string {
	return "reservoirs"
}
