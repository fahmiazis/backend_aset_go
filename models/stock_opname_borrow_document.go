package models

import "time"

// StockOpnameBorrowDocument adalah dokumen bukti peminjaman (PDF) per baris
// asset di draft stock opname (TransactionStockOpname) yang statusnya
// BORROWED ("Dipinjam"). Satu asset cuma boleh punya 1 dokumen aktif —
// re-upload menimpa (replace) dokumen lama, bukan nambah baris baru.
type StockOpnameBorrowDocument struct {
	ID                       uint      `gorm:"primaryKey" json:"id"`
	TransactionStockOpnameID uint      `gorm:"not null;uniqueIndex" json:"transaction_stock_opname_id"`
	TransactionID            uint      `gorm:"not null;index" json:"transaction_id"`
	AssetID                  uint      `gorm:"not null;index" json:"asset_id"`
	FileName                 string    `gorm:"size:255" json:"file_name"`
	FilePath                 string    `gorm:"size:500" json:"-"`
	FileSize                 int64     `json:"file_size"`
	UploadedBy               string    `gorm:"size:100" json:"uploaded_by"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

func (StockOpnameBorrowDocument) TableName() string { return "stock_opname_borrow_documents" }
