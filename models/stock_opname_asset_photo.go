package models

import "time"

// StockOpnameAssetPhoto adalah 1 foto bukti fisik per baris asset di draft
// stock opname (TransactionStockOpname). Satu asset cuma boleh punya 1 foto
// aktif — re-upload menimpa (replace) foto lama, bukan nambah baris baru.
type StockOpnameAssetPhoto struct {
	ID                       uint      `gorm:"primaryKey" json:"id"`
	TransactionStockOpnameID uint      `gorm:"not null;uniqueIndex" json:"transaction_stock_opname_id"`
	TransactionID            uint      `gorm:"not null;index" json:"transaction_id"`
	AssetID                  uint      `gorm:"not null;index" json:"asset_id"`
	FileName                 string    `gorm:"size:255" json:"file_name"`
	FilePath                 string    `gorm:"size:500" json:"-"`
	FileSize                 int64     `json:"file_size"`
	FileHash                 string    `gorm:"size:64;index" json:"-"`
	CapturedAt               time.Time `json:"captured_at"`
	UploadedBy               string    `gorm:"size:100" json:"uploaded_by"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

func (StockOpnameAssetPhoto) TableName() string { return "stock_opname_asset_photos" }
