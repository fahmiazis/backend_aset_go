package models

import "time"

type DocumentSequence struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Prefix     string    `gorm:"size:10;not null;uniqueIndex:idx_prefix_year_month" json:"prefix"`
	Year       int       `gorm:"not null;uniqueIndex:idx_prefix_year_month;index" json:"year"`
	Month      int       `gorm:"not null;uniqueIndex:idx_prefix_year_month;index" json:"month"`
	LastNumber int       `gorm:"not null;default:0" json:"last_number"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (DocumentSequence) TableName() string { return "document_sequences" }

const (
	DocumentPrefixAcquisition = "ACQ"
	DocumentPrefixMutation    = "MUT"
	DocumentPrefixDisposal    = "DIS"
	DocumentPrefixStockOpname = "SO"
)
