package models

import "time"

type AssetMutation struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	DocumentNumber    string     `gorm:"size:100;uniqueIndex;not null" json:"document_number"`
	TransactionID     *uint      `gorm:"index" json:"transaction_id"`
	TransactionNumber string     `gorm:"size:100;index" json:"transaction_number"`
	TransactionDate   time.Time  `gorm:"type:date;not null;index" json:"transaction_date"`
	AssetID           uint       `gorm:"not null;index" json:"asset_id"`
	AssetNumber       string     `gorm:"size:100;not null" json:"asset_number"`
	FromBranchCode    string     `gorm:"size:50;index" json:"from_branch_code"`
	ToBranchCode      string     `gorm:"size:50;index" json:"to_branch_code"`
	FromLocation      string     `gorm:"size:255" json:"from_location"`
	ToLocation        string     `gorm:"size:255" json:"to_location"`
	Notes             string     `gorm:"type:text" json:"notes"`
	Status            string     `gorm:"size:50;not null;default:DRAFT;index" json:"status"`
	CreatedBy         string     `gorm:"size:100" json:"created_by"`
	ApprovedBy        string     `gorm:"size:100" json:"approved_by"`
	ApprovedAt        *time.Time `json:"approved_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	Asset       *Asset       `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (AssetMutation) TableName() string { return "asset_mutations" }
