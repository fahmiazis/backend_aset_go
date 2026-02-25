package models

import "time"

type AssetStockOpname struct {
	ID                      uint      `gorm:"primaryKey" json:"id"`
	StockOpnameID           uint      `gorm:"not null;index;uniqueIndex:idx_so_asset" json:"stock_opname_id"`
	AssetID                 uint      `gorm:"not null;index;uniqueIndex:idx_so_asset" json:"asset_id"`
	TransactionID           *uint     `gorm:"index" json:"transaction_id"`              // FIX: tambah dari migration 22
	TransactionNumber       *string   `gorm:"size:100;index" json:"transaction_number"` // FIX: tambah dari migration 22
	AssetNumber             string    `gorm:"size:100;not null;index" json:"asset_number"`
	AssetName               string    `gorm:"size:255;not null" json:"asset_name"`
	Description             string    `gorm:"type:text" json:"description"`
	Brand                   string    `gorm:"size:100" json:"brand"`
	UnitOfMeasure           string    `gorm:"size:50" json:"unit_of_measure"`
	UnitQuantity            float64   `gorm:"type:decimal(15,2)" json:"unit_quantity"`
	Condition               string    `gorm:"column:condition;size:50" json:"condition"` // FIX: explicit column name
	PhysicalStatus          string    `gorm:"size:50" json:"physical_status"`
	AssetStatus             string    `gorm:"size:50" json:"asset_status"`
	Location                string    `gorm:"size:255" json:"location"`
	Grouping                string    `gorm:"size:100" json:"grouping"`
	Notes                   string    `gorm:"type:text" json:"notes"`
	BookValue               float64   `gorm:"type:decimal(18,2);not null;default:0" json:"book_value"`
	AcquisitionValue        float64   `gorm:"type:decimal(18,2);not null;default:0" json:"acquisition_value"`
	AccumulatedDepreciation float64   `gorm:"type:decimal(18,2);not null;default:0" json:"accumulated_depreciation"`
	CategoryID              *uint     `gorm:"index" json:"category_id"`
	BranchCode              string    `gorm:"size:50;index" json:"branch_code"`
	IONumber                string    `gorm:"size:100" json:"io_number"`
	RecordType              string    `gorm:"size:50" json:"record_type"`
	CreatedAt               time.Time `json:"created_at"`

	StockOpname *StockOpname   `gorm:"foreignKey:StockOpnameID" json:"stock_opname,omitempty"`
	Asset       *Asset         `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
	Category    *AssetCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Transaction *Transaction   `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
}

func (AssetStockOpname) TableName() string { return "asset_stock_opnames" }
