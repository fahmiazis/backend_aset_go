package models

import (
	"time"
)

type TransactionDisposal struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	TransactionID     uint      `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber string    `gorm:"size:100;not null;index" json:"transaction_number"`
	AssetID           uint      `gorm:"not null;index" json:"asset_id"`
	AssetNumber       string    `gorm:"size:100;not null" json:"asset_number"`
	DisposalMethod    string    `gorm:"size:50;index" json:"disposal_method"`
	DisposalValue     float64   `gorm:"type:decimal(18,2);not null;default:0" json:"disposal_value"`
	DisposalReason    *string   `gorm:"type:text" json:"disposal_reason"` // NULLABLE
	Notes             *string   `gorm:"type:text" json:"notes"`           // NULLABLE
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// Relations
	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	Asset       *Asset       `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (TransactionDisposal) TableName() string {
	return "transaction_disposals"
}
