package models

import "time"

type Asset struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	AssetNumber   string     `gorm:"size:100;uniqueIndex;not null" json:"asset_number"`
	AssetName     string     `gorm:"size:255;not null" json:"asset_name"`
	Description   *string    `gorm:"type:text" json:"description"`
	Brand         *string    `gorm:"size:100" json:"brand"`
	UnitOfMeasure *string    `gorm:"size:50" json:"unit_of_measure"`
	UnitQuantity  *float64   `gorm:"type:decimal(15,2)" json:"unit_quantity"` // FIX: 15,2 sesuai migration
	Location      *string    `gorm:"size:255" json:"location"`
	Grouping      *string    `gorm:"size:100" json:"grouping"`
	CategoryID    *uint      `gorm:"index" json:"category_id"`
	BranchCode    *string    `gorm:"size:50;index" json:"branch_code"`
	IONumber      *string    `gorm:"size:100" json:"io_number"`
	RecordType    *string    `gorm:"size:50" json:"record_type"`
	AssetStatus   string     `gorm:"size:50;not null;default:ACTIVE;index" json:"asset_status"`
	DeletedAt     *time.Time `gorm:"index" json:"deleted_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	Category    *AssetCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	AssetValues []AssetValue   `gorm:"foreignKey:AssetID" json:"asset_values,omitempty"`
}

func (Asset) TableName() string { return "assets" }

const (
	AssetStatusInactive    = "INACTIVE"
	AssetStatusMaintenance = "MAINTENANCE"
	AssetStatusRetired     = "RETIRED"
	AssetStatusDisposed    = "DISPOSED"
)
