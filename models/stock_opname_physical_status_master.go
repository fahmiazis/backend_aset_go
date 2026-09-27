package models

import (
	"time"

	"gorm.io/gorm"
)

// StockOpnamePhysicalStatusMaster adalah master data status fisik stock
// opname (dulu hardcode EXISTS/MISSING/BORROWED). Status bawaan sistem
// (IsSystem=true) di-seed lewat migration dan tidak bisa dihapus karena
// terikat ke logic inti (lihat services/stock_opname_status_master_service.go)
// — status baru yang ditambah admin lewat master data ini murni data,
// perilakunya ditentukan lewat flag di bawah (tidak ada "update" sengaja,
// biar flag-nya gak berubah di tengah jalan transaksi yang udah pakai kode itu).
// Kondisi yang boleh dipasangkan diatur terpisah lewat
// StockOpnamePhysicalConditionRule (itu yang boleh diubah).
type StockOpnamePhysicalStatusMaster struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Code  string `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Label string `gorm:"size:100;not null" json:"label"`

	// RequiresBorrowDocument: status ini butuh dokumen peminjaman (PDF/dll,
	// lihat StockOpnameConfig) sebelum bisa disimpan/submit — dulu hanya
	// berlaku utk "BORROWED".
	RequiresBorrowDocument bool `gorm:"not null;default:false" json:"requires_borrow_document"`

	// CountsAsMissing: dipakai laporan buat bucket "Hilang"/"SAP ADA FISIK
	// TIDAK" (dulu hanya "MISSING"). BORROWED sengaja TIDAK masuk sini karena
	// aset dianggap masih "ada" (cuma lagi dipinjam).
	CountsAsMissing bool `gorm:"not null;default:false" json:"counts_as_missing"`

	// IsSystem: status bawaan (EXISTS/MISSING/BORROWED) yang di-seed migration
	// — tidak bisa dihapus lewat API, dicek di service.
	IsSystem bool `gorm:"not null;default:false" json:"is_system"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (StockOpnamePhysicalStatusMaster) TableName() string {
	return "stock_opname_physical_status_masters"
}
