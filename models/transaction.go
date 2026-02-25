package models

import "time"

type Transaction struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	TransactionNumber string     `gorm:"size:100;uniqueIndex;not null" json:"transaction_number"`
	TransactionType   string     `gorm:"size:50;not null;index" json:"transaction_type"`
	TransactionDate   time.Time  `gorm:"type:date;not null;index" json:"transaction_date"`
	Status            string     `gorm:"size:50;not null;default:DRAFT;index" json:"status"`
	Notes             *string    `gorm:"type:text" json:"notes"`
	CreatedBy         string     `gorm:"size:100" json:"created_by"`
	ApprovedBy        *string    `gorm:"size:100" json:"approved_by"`
	ApprovedAt        *time.Time `json:"approved_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	// Relations to Transaction Details
	TransactionProcurements []TransactionProcurement `gorm:"foreignKey:TransactionID" json:"transaction_procurements,omitempty"`
	TransactionMutations    []TransactionMutation    `gorm:"foreignKey:TransactionID" json:"transaction_mutations,omitempty"`
	TransactionDisposals    []TransactionDisposal    `gorm:"foreignKey:TransactionID" json:"transaction_disposals,omitempty"`
	TransactionStockOpnames []TransactionStockOpname `gorm:"foreignKey:TransactionID" json:"transaction_stock_opnames,omitempty"`

	// Relations to Asset Records
	AssetAcquisitions []AssetAcquisition `gorm:"foreignKey:TransactionID" json:"asset_acquisitions,omitempty"`
	AssetMutations    []AssetMutation    `gorm:"foreignKey:TransactionID" json:"asset_mutations,omitempty"`
	AssetDisposals    []AssetDisposal    `gorm:"foreignKey:TransactionID" json:"asset_disposals,omitempty"`
	AssetStockOpnames []AssetStockOpname `gorm:"foreignKey:TransactionID" json:"asset_stock_opnames,omitempty"`
}

func (Transaction) TableName() string { return "transactions" }
