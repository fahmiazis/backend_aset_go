package models

import (
	"time"

	"gorm.io/gorm"
)

// StockOpnameConditionMaster adalah master data kondisi aset stock opname
// (dulu hardcode GOOD/FAIR/POOR/BROKEN/NOT_APPLICABLE). Lihat catatan
// IsSystem di StockOpnamePhysicalStatusMaster — sama perlakuannya di sini.
type StockOpnameConditionMaster struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Code  string `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Label string `gorm:"size:100;not null" json:"label"`

	// IsNotApplicableValue: menandai kondisi ini sebagai representasi
	// "Tidak Ada"/N.A (dulu hardcode "NOT_APPLICABLE") — dipakai validasi
	// silang terhadap StockOpnamePhysicalStatusMaster.RequiresNotApplicableCondition.
	IsNotApplicableValue bool `gorm:"not null;default:false" json:"is_not_applicable_value"`

	// ReportBucket: "BAIK" | "RUSAK" | "" (kosong = tidak dihitung di bucket
	// manapun, dulu berlaku utk NOT_APPLICABLE). Dipakai laporan stock opname.
	ReportBucket string `gorm:"size:20;not null;default:''" json:"report_bucket"`

	IsSystem bool `gorm:"not null;default:false" json:"is_system"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (StockOpnameConditionMaster) TableName() string {
	return "stock_opname_condition_masters"
}
