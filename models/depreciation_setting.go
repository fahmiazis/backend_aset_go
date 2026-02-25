package models

import "time"

type DepreciationSetting struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	SettingType        string     `gorm:"size:50;not null;index" json:"setting_type"`
	ReferenceID        *uint      `gorm:"column:reference_id;index" json:"reference_id"`          // FIX: ganti dari CategoryID/AssetID
	ReferenceValue     *string    `gorm:"column:reference_value;size:255" json:"reference_value"` // FIX: tambah
	CalculationMethod  string     `gorm:"size:50;not null" json:"calculation_method"`
	DepreciationPeriod string     `gorm:"size:50;not null" json:"depreciation_period"`
	UsefulLifeMonths   int        `gorm:"not null" json:"useful_life_months"`
	DepreciationRate   *float64   `gorm:"type:decimal(10,4)" json:"depreciation_rate"` // FIX: decimal(10,4) & nullable
	StartDate          time.Time  `gorm:"type:date;not null" json:"start_date"`        // FIX: tambah
	EndDate            *time.Time `gorm:"type:date" json:"end_date"`                   // FIX: tambah
	IsActive           bool       `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (DepreciationSetting) TableName() string { return "depreciation_settings" }

const (
	SettingTypeCategory = "CATEGORY"
	SettingTypeAsset    = "ASSET"
)

const (
	CalculationMethodStraightLine     = "STRAIGHT_LINE"
	CalculationMethodDecliningBalance = "DECLINING_BALANCE"
)

const (
	DepreciationPeriodMonthly = "MONTHLY"
	DepreciationPeriodDaily   = "DAILY"
)
