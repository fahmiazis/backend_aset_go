package models

import "time"

// StockOpnamePhysicalConditionRule: kondisi mana aja yang boleh dipilih
// buat satu status fisik (misal "Tidak Ada" cuma boleh kondisi "Tidak Ada").
// Diatur lewat master data status — menggantikan flag lama
// RequiresNotApplicableCondition / IsNotApplicableValue.
type StockOpnamePhysicalConditionRule struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	PhysicalStatusID uint      `gorm:"not null;index" json:"physical_status_id"`
	ConditionID      uint      `gorm:"not null;index" json:"condition_id"`
	CreatedAt        time.Time `json:"created_at"`
}

func (StockOpnamePhysicalConditionRule) TableName() string {
	return "stock_opname_physical_condition_rules"
}
