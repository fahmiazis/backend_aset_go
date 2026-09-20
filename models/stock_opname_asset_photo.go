package models

import "time"

// StockOpnameAssetPhoto adalah 1 foto bukti fisik per asset di sebuah stock
// opname. Satu asset cuma boleh punya 1 foto aktif per transaksi — re-upload
// menimpa (replace) foto lama, bukan nambah baris baru. Dikunci lewat
// (transaction_id, asset_id), BUKAN lewat row item stock opname-nya sendiri
// — item itu berpindah antar tabel seiring stage (draft -> aktif -> history),
// sementara pasangan (transaction_id, asset_id) tetap stabil sepanjang umur
// transaksi.
type StockOpnameAssetPhoto struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TransactionID uint      `gorm:"not null;index" json:"transaction_id"`
	AssetID       uint      `gorm:"not null;index" json:"asset_id"`
	FileName      string    `gorm:"size:255" json:"file_name"`
	FilePath      string    `gorm:"size:500" json:"-"`
	FileSize      int64     `json:"file_size"`
	FileHash      string    `gorm:"size:64;index" json:"-"`
	CapturedAt    time.Time `json:"captured_at"`
	UploadedBy    string    `gorm:"size:100" json:"uploaded_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (StockOpnameAssetPhoto) TableName() string { return "stock_opname_asset_photos" }
