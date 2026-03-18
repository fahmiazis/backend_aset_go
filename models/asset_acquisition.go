package models

import "time"

type AssetAcquisition struct {
	ID                       uint       `gorm:"primaryKey" json:"id"`
	DocumentNumber           string     `gorm:"size:100;uniqueIndex;not null" json:"document_number"`
	TransactionID            *uint      `gorm:"index" json:"transaction_id"`
	TransactionNumber        string     `gorm:"size:100;index" json:"transaction_number"`
	TransactionProcurementID *uint      `gorm:"index" json:"transaction_procurement_id"` // ADD
	TransactionDate          time.Time  `gorm:"type:date;not null;index" json:"transaction_date"`
	AssetID                  *uint      `gorm:"index" json:"asset_id"`
	AssetNumber              string     `gorm:"size:100" json:"asset_number"`
	AssetName                string     `gorm:"size:255;not null" json:"asset_name"`
	AcquisitionValue         float64    `gorm:"type:decimal(18,2);not null;default:0" json:"acquisition_value"`
	CategoryID               *uint      `gorm:"index" json:"category_id"`
	BranchCode               string     `gorm:"size:50;index" json:"branch_code"`
	Location                 string     `gorm:"size:255" json:"location"`
	IONumber                 string     `gorm:"size:100" json:"io_number"`
	Notes                    string     `gorm:"type:text" json:"notes"`
	Status                   string     `gorm:"size:50;not null;default:DRAFT;index" json:"status"`
	CreatedBy                string     `gorm:"size:100" json:"created_by"`
	ApprovedBy               string     `gorm:"size:100" json:"approved_by"`
	ApprovedAt               *time.Time `json:"approved_at"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`

	Transaction            *Transaction            `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	TransactionProcurement *TransactionProcurement `gorm:"foreignKey:TransactionProcurementID" json:"transaction_procurement,omitempty"` // ADD
	Asset                  *Asset                  `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
	Category               *AssetCategory          `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

func (AssetAcquisition) TableName() string { return "asset_acquisitions" }
