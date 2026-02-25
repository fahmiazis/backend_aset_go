package dto

import "time"

// --- Depreciation Settings ---

type CreateDepreciationSettingRequest struct {
	SettingType        string   `json:"setting_type" binding:"required,oneof=CATEGORY ASSET"`
	ReferenceID        *uint    `json:"reference_id"`    // FIX: category_id atau asset_id tergantung setting_type
	ReferenceValue     *string  `json:"reference_value"` // FIX: category_code atau asset_number
	CalculationMethod  string   `json:"calculation_method" binding:"required,oneof=STRAIGHT_LINE DECLINING_BALANCE"`
	DepreciationPeriod string   `json:"depreciation_period" binding:"required,oneof=MONTHLY DAILY"`
	UsefulLifeMonths   int      `json:"useful_life_months" binding:"required,min=1"`
	DepreciationRate   *float64 `json:"depreciation_rate" binding:"omitempty,min=0"` // FIX: nullable
	StartDate          string   `json:"start_date" binding:"required"`               // FIX: tambah, format YYYY-MM-DD
	EndDate            *string  `json:"end_date"`                                    // FIX: tambah, optional
	IsActive           bool     `json:"is_active"`
}

type UpdateDepreciationSettingRequest struct {
	CalculationMethod  *string  `json:"calculation_method" binding:"omitempty,oneof=STRAIGHT_LINE DECLINING_BALANCE"`
	DepreciationPeriod *string  `json:"depreciation_period" binding:"omitempty,oneof=MONTHLY DAILY"`
	UsefulLifeMonths   *int     `json:"useful_life_months" binding:"omitempty,min=1"`
	DepreciationRate   *float64 `json:"depreciation_rate" binding:"omitempty,min=0"`
	EndDate            *string  `json:"end_date"` // FIX: tambah
	IsActive           *bool    `json:"is_active"`
}

type DepreciationSettingResponse struct {
	ID                 uint       `json:"id"`
	SettingType        string     `json:"setting_type"`
	ReferenceID        *uint      `json:"reference_id"`    // FIX: ganti dari CategoryID/AssetID
	ReferenceValue     *string    `json:"reference_value"` // FIX: ganti dari CategoryName/AssetNumber
	CalculationMethod  string     `json:"calculation_method"`
	DepreciationPeriod string     `json:"depreciation_period"`
	UsefulLifeMonths   int        `json:"useful_life_months"`
	DepreciationRate   *float64   `json:"depreciation_rate"` // FIX: nullable
	StartDate          time.Time  `json:"start_date"`        // FIX: tambah
	EndDate            *time.Time `json:"end_date"`          // FIX: tambah
	IsActive           bool       `json:"is_active"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// --- Monthly Depreciation ---

type MonthlyDepreciationResponse struct {
	ID                               uint      `json:"id"`
	AssetID                          uint      `json:"asset_id"`
	AssetNumber                      string    `json:"asset_number,omitempty"`
	AssetName                        string    `json:"asset_name,omitempty"`
	Period                           string    `json:"period"`           // FIX: "YYYY-MM", ganti dari CalculationMonth+CalculationYear
	CalculationDate                  time.Time `json:"calculation_date"` // FIX: tambah
	BeginningBookValue               float64   `json:"beginning_book_value"`
	DepreciationAmount               float64   `json:"depreciation_amount"`
	BeginningAccumulatedDepreciation float64   `json:"beginning_accumulated_depreciation"` // FIX: tambah
	EndingAccumulatedDepreciation    float64   `json:"ending_accumulated_depreciation"`    // FIX: tambah
	EndingBookValue                  float64   `json:"ending_book_value"`
	CalculationMethod                *string   `json:"calculation_method"`      // FIX: nullable
	DepreciationSettingID            *uint     `json:"depreciation_setting_id"` // FIX: tambah
	IsLocked                         bool      `json:"is_locked"`
	CreatedAt                        time.Time `json:"created_at"`
	UpdatedAt                        time.Time `json:"updated_at"`
}

type CalculateDepreciationRequest struct {
	Period string `json:"period" binding:"required"` // FIX: format "YYYY-MM", ganti dari Month+Year terpisah
}
