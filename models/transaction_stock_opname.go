package models

import "time"

type TransactionStockOpname struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	TransactionID     uint      `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber string    `gorm:"size:100;not null;index" json:"transaction_number"`
	AssetID           uint      `gorm:"not null;index" json:"asset_id"`
	AssetNumber       string    `gorm:"size:100;not null" json:"asset_number"`
	PhysicalStatus    string    `gorm:"size:50" json:"physical_status"`
	Condition         string    `gorm:"column:condition;size:50" json:"condition"` // FIX: explicit column name
	AssetStatus       string    `gorm:"size:50" json:"asset_status"`
	Notes             *string   `gorm:"type:text" json:"notes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	Asset       *Asset       `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (TransactionStockOpname) TableName() string { return "transaction_stock_opnames" }
