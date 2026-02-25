package models

import "time"

type AssetDisposal struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	DocumentNumber      string     `gorm:"size:100;uniqueIndex;not null" json:"document_number"`
	TransactionID       *uint      `gorm:"index" json:"transaction_id"`
	TransactionNumber   string     `gorm:"size:100;index" json:"transaction_number"`
	TransactionDate     time.Time  `gorm:"type:date;not null;index" json:"transaction_date"`
	AssetID             uint       `gorm:"not null;index" json:"asset_id"`
	AssetNumber         string     `gorm:"size:100;not null" json:"asset_number"`
	AssetName           string     `gorm:"size:255;not null" json:"asset_name"`
	BookValueAtDisposal float64    `gorm:"type:decimal(18,2);not null;default:0" json:"book_value_at_disposal"`
	DisposalValue       float64    `gorm:"type:decimal(18,2);not null;default:0" json:"disposal_value"`
	DisposalReason      string     `gorm:"type:text" json:"disposal_reason"`
	DisposalMethod      string     `gorm:"size:50;index" json:"disposal_method"`
	Notes               string     `gorm:"type:text" json:"notes"`
	Status              string     `gorm:"size:50;not null;default:DRAFT;index" json:"status"`
	CreatedBy           string     `gorm:"size:100" json:"created_by"`
	ApprovedBy          string     `gorm:"size:100" json:"approved_by"`
	ApprovedAt          *time.Time `json:"approved_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`

	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	Asset       *Asset       `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (AssetDisposal) TableName() string { return "asset_disposals" }
