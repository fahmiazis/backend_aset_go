package models

import (
	"time"
)

type TransactionMutation struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	TransactionID     uint      `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber string    `gorm:"size:100;not null;index" json:"transaction_number"`
	AssetID           uint      `gorm:"not null;index" json:"asset_id"`
	AssetNumber       string    `gorm:"size:100;not null" json:"asset_number"`
	FromBranchCode    string    `gorm:"size:50;index" json:"from_branch_code"`
	ToBranchCode      string    `gorm:"size:50;index" json:"to_branch_code"`
	FromLocation      *string   `gorm:"size:255" json:"from_location"` // NULLABLE
	ToLocation        *string   `gorm:"size:255" json:"to_location"`   // NULLABLE
	Notes             *string   `gorm:"type:text" json:"notes"`        // NULLABLE
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// Relations
	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	Asset       *Asset       `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (TransactionMutation) TableName() string {
	return "transaction_mutations"
}
