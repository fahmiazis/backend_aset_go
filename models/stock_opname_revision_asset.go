package models

import "time"

// StockOpnameRevisionAsset adalah asset yang dichecklist approver/eksekutor
// waktu mentrigger revisi. Selama transaksi ada di DRAFT dan masih punya
// baris di sini, cuma asset-asset ini yang boleh diubah — asset lain dikunci.
// Dihapus semua begitu draft revisi di-submit ulang. Dikunci lewat
// (transaction_id, asset_id), sama kayak foto/dokumen peminjaman.
type StockOpnameRevisionAsset struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TransactionID uint      `gorm:"not null;index" json:"transaction_id"`
	AssetID       uint      `gorm:"not null;index" json:"asset_id"`
	CreatedBy     string    `gorm:"size:100" json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

func (StockOpnameRevisionAsset) TableName() string { return "stock_opname_revision_assets" }
