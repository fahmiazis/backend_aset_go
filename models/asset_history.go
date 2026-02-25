package models

import "time"

type AssetHistory struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	AssetID         uint       `gorm:"not null;index" json:"asset_id"`
	TransactionType string     `gorm:"size:50;not null;index" json:"transaction_type"`
	TransactionID   *uint      `gorm:"index" json:"transaction_id"` // FIX: *string -> *uint
	DocumentNumber  *string    `gorm:"size:100" json:"document_number"`
	TransactionDate *time.Time `gorm:"type:date" json:"transaction_date"`
	BeforeData      *string    `gorm:"type:json" json:"before_data"` // FIX: jsonb -> json
	AfterData       *string    `gorm:"type:json" json:"after_data"`  // FIX: jsonb -> json
	ChangedBy       *string    `gorm:"size:100" json:"changed_by"`
	CreatedAt       time.Time  `json:"created_at"`

	Asset *Asset `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (AssetHistory) TableName() string { return "asset_histories" }
