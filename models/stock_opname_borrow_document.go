package models

import "time"

// StockOpnameBorrowDocument adalah dokumen bukti peminjaman (PDF) per asset
// di sebuah stock opname yang statusnya BORROWED ("Dipinjam"). Satu asset
// cuma boleh punya 1 dokumen aktif per transaksi — re-upload menimpa
// (replace) dokumen lama, bukan nambah baris baru. Dikunci lewat
// (transaction_id, asset_id), BUKAN lewat row item stock opname-nya sendiri
// — item itu berpindah antar tabel seiring stage (draft -> aktif -> history),
// sementara pasangan (transaction_id, asset_id) tetap stabil sepanjang umur
// transaksi.
type StockOpnameBorrowDocument struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TransactionID uint      `gorm:"not null;index" json:"transaction_id"`
	AssetID       uint      `gorm:"not null;index" json:"asset_id"`
	FileName      string    `gorm:"size:255" json:"file_name"`
	FilePath      string    `gorm:"size:500" json:"-"`
	FileSize      int64     `json:"file_size"`
	UploadedBy    string    `gorm:"size:100" json:"uploaded_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (StockOpnameBorrowDocument) TableName() string { return "stock_opname_borrow_documents" }
