package models

import (
	"time"
)

// StockOpname represents stock opname header
type StockOpname struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	DocumentNumber string     `gorm:"size:100;uniqueIndex;not null" json:"document_number"`
	OpnameDate     time.Time  `gorm:"type:date;not null;index" json:"opname_date"`
	Period         string     `gorm:"size:7;not null;index" json:"period"` // YYYY-MM
	Status         string     `gorm:"size:50;not null;default:DRAFT;index" json:"status"`
	Notes          string     `gorm:"type:text" json:"notes"`
	CreatedBy      string     `gorm:"size:100" json:"created_by"`
	ApprovedBy     string     `gorm:"size:100" json:"approved_by"`
	ApprovedAt     *time.Time `json:"approved_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relations
	AssetStockOpnames []AssetStockOpname `gorm:"foreignKey:StockOpnameID" json:"asset_stock_opnames,omitempty"`
}

// TableName specifies the table name
func (StockOpname) TableName() string {
	return "stock_opnames"
}

// StockOpnameStatus constants
const (
	StockOpnameStatusDraft    = "DRAFT"
	StockOpnameStatusLocked   = "LOCKED"
	StockOpnameStatusApproved = "APPROVED"
)
