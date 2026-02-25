package models

import "time"

type AssetCategory struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	CategoryCode string     `gorm:"size:50;uniqueIndex;not null" json:"category_code"`
	CategoryName string     `gorm:"size:255;not null" json:"category_name"`
	Description  *string    `gorm:"type:text" json:"description"`
	IsActive     bool       `gorm:"not null;default:true" json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `gorm:"index" json:"deleted_at"` // ada di migration tapi tidak di model sebelumnya

	Assets []Asset `gorm:"foreignKey:CategoryID" json:"assets,omitempty"`
}

func (AssetCategory) TableName() string { return "asset_categories" }
