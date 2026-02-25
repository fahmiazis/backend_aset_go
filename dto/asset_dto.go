package dto

import "time"

type AssetResponse struct {
	ID            uint                `json:"id"`
	AssetNumber   string              `json:"asset_number"`
	AssetName     string              `json:"asset_name"`
	Description   *string             `json:"description"`
	Brand         *string             `json:"brand"`
	UnitOfMeasure *string             `json:"unit_of_measure"`
	UnitQuantity  *float64            `json:"unit_quantity"`
	Location      *string             `json:"location"`
	Grouping      *string             `json:"grouping"`
	CategoryID    *uint               `json:"category_id"`
	CategoryName  *string             `json:"category_name,omitempty"`
	BranchCode    *string             `json:"branch_code"`
	IONumber      *string             `json:"io_number"`
	RecordType    *string             `json:"record_type"`
	AssetStatus   string              `json:"asset_status"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	CurrentValue  *AssetValueResponse `json:"current_value,omitempty"`
}

type AssetValueResponse struct {
	ID                      uint      `json:"id"`
	AssetID                 uint      `json:"asset_id"`
	EffectiveDate           time.Time `json:"effective_date"`
	BookValue               float64   `json:"book_value"`
	AcquisitionValue        float64   `json:"acquisition_value"`
	AccumulatedDepreciation float64   `json:"accumulated_depreciation"`
	Condition               *string   `json:"condition"`
	PhysicalStatus          *string   `json:"physical_status"`
	AssetStatus             *string   `json:"asset_status"`
	IsActive                bool      `json:"is_active"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type AssetListFilter struct {
	BranchCode  *string `form:"branch_code"`
	CategoryID  *uint   `form:"category_id"`
	AssetStatus *string `form:"asset_status"`
	Search      *string `form:"search"`
	Page        int     `form:"page" binding:"min=1"`
	Limit       int     `form:"limit" binding:"min=1,max=100"`
}
