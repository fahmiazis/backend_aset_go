package models

import "time"

type MonthlyDepreciationCalculation struct {
	ID                               uint      `gorm:"primaryKey" json:"id"`
	AssetID                          uint      `gorm:"not null;index" json:"asset_id"`
	Period                           string    `gorm:"size:7;not null;uniqueIndex:idx_asset_period" json:"period"` // FIX: "YYYY-MM"
	CalculationDate                  time.Time `gorm:"type:date;not null" json:"calculation_date"`                 // FIX: tambah
	BeginningBookValue               float64   `gorm:"type:decimal(18,2);not null;default:0" json:"beginning_book_value"`
	DepreciationAmount               float64   `gorm:"type:decimal(18,2);not null;default:0" json:"depreciation_amount"`
	BeginningAccumulatedDepreciation float64   `gorm:"type:decimal(18,2);not null;default:0" json:"beginning_accumulated_depreciation"` // FIX: tambah
	EndingAccumulatedDepreciation    float64   `gorm:"type:decimal(18,2);not null;default:0" json:"ending_accumulated_depreciation"`    // FIX: tambah
	EndingBookValue                  float64   `gorm:"type:decimal(18,2);not null;default:0" json:"ending_book_value"`
	CalculationMethod                *string   `gorm:"size:50" json:"calculation_method"`    // FIX: nullable sesuai migration
	DepreciationSettingID            *uint     `gorm:"index" json:"depreciation_setting_id"` // FIX: tambah
	IsLocked                         bool      `gorm:"not null;default:false;index" json:"is_locked"`
	CreatedAt                        time.Time `json:"created_at"`
	UpdatedAt                        time.Time `json:"updated_at"`

	Asset               *Asset               `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
	DepreciationSetting *DepreciationSetting `gorm:"foreignKey:DepreciationSettingID" json:"depreciation_setting,omitempty"`
}

func (MonthlyDepreciationCalculation) TableName() string { return "monthly_depreciation_calculations" }
