package models

import (
	"time"
)

type TransactionProcurement struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	TransactionID     uint      `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber string    `gorm:"size:100;not null;index" json:"transaction_number"`
	ItemName          string    `gorm:"size:255;not null" json:"item_name"`
	CategoryID        *uint     `gorm:"index" json:"category_id"`
	Quantity          int       `gorm:"not null" json:"quantity"`
	UnitPrice         float64   `gorm:"type:decimal(18,2);not null;default:0" json:"unit_price"`
	TotalPrice        float64   `gorm:"type:decimal(18,2);not null;default:0" json:"total_price"`
	BranchCode        string    `gorm:"size:50;index" json:"branch_code"`
	Notes             *string   `gorm:"type:text" json:"notes"` // NULLABLE - FIXED!
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// Relations
	Transaction                   *Transaction                   `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	Category                      *AssetCategory                 `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	TransactionProcurementDetails []TransactionProcurementDetail `gorm:"foreignKey:TransactionProcurementID" json:"transaction_procurement_details,omitempty"`
}

func (TransactionProcurement) TableName() string {
	return "transaction_procurements"
}
