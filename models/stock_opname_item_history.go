package models

import "time"

// StockOpnameItemHistory adalah baris temuan per-asset setelah stock opname
// selesai (FINISHED). Diisi dengan cara memindahkan baris dari
// transaction_stock_opnames (insert ke sini + delete dari sana) — lihat
// ExecuteStockOpname. Terpisah dari asset_histories (audit trail JSON
// generik before/after) — tabel ini menyimpan bentuk relasional penuh biar
// report & detail tetap bisa nampilin foto/notes dst untuk opname yang udah
// selesai.
type StockOpnameItemHistory struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	TransactionID     uint      `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber string    `gorm:"size:100;not null;index" json:"transaction_number"`
	AssetID           uint      `gorm:"not null;index" json:"asset_id"`
	AssetNumber       string    `gorm:"size:100;not null" json:"asset_number"`
	PhysicalStatus    string    `gorm:"size:50" json:"physical_status"`
	Condition         string    `gorm:"column:condition;size:50" json:"condition"`
	AssetStatus       string    `gorm:"size:50" json:"asset_status"`
	Notes             *string   `gorm:"type:text" json:"notes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	Asset       *Asset       `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (StockOpnameItemHistory) TableName() string { return "stock_opname_item_history" }
