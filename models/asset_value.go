package models

import "time"

type AssetValue struct {
	ID                      uint      `gorm:"primaryKey" json:"id"`
	AssetID                 uint      `gorm:"not null;index" json:"asset_id"`
	EffectiveDate           time.Time `gorm:"type:date;not null;index" json:"effective_date"`
	BookValue               float64   `gorm:"type:decimal(18,2);not null;default:0" json:"book_value"`
	AcquisitionValue        float64   `gorm:"type:decimal(18,2);not null;default:0" json:"acquisition_value"`
	AccumulatedDepreciation float64   `gorm:"type:decimal(18,2);not null;default:0" json:"accumulated_depreciation"`
	Condition               *string   `gorm:"column:condition;size:50" json:"condition"` // FIX: explicit column name karena reserved keyword
	PhysicalStatus          *string   `gorm:"size:50" json:"physical_status"`
	AssetStatus             *string   `gorm:"size:50" json:"asset_status"`
	IsActive                bool      `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`

	Asset *Asset `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (AssetValue) TableName() string { return "asset_values" }

const (
	ConditionGood   = "GOOD"
	ConditionFair   = "FAIR"
	ConditionPoor   = "POOR"
	ConditionBroken = "BROKEN"
)

const (
	PhysicalStatusExists   = "EXISTS"
	PhysicalStatusMissing  = "MISSING"
	PhysicalStatusDamaged  = "DAMAGED"
	PhysicalStatusObsolete = "OBSOLETE"
)
