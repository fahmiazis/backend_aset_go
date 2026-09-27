package dto

import "time"

// ============================================================
// MASTER DATA STATUS FISIK & KONDISI STOCK OPNAME
//
// Status fisik & kondisi sendiri sengaja CREATE + LIST + SOFT DELETE saja
// (tidak ada UPDATE) — flag perilaku (requires_borrow_document dkk)
// ditentukan sekali saat create supaya tidak berubah di tengah jalan
// transaksi yang sudah pakai kode itu. Baris bawaan sistem (is_system=true)
// tidak bisa dihapus.
//
// Yang BOLEH diubah kapan aja: relasi status fisik -> kondisi yang boleh
// dipilih (PUT .../physical-status/:id/conditions).
// ============================================================

type CreateStockOpnamePhysicalStatusMasterRequest struct {
	Code                   string `json:"code" binding:"required,min=2,max=50"`
	Label                  string `json:"label" binding:"required,min=1,max=100"`
	RequiresBorrowDocument bool   `json:"requires_borrow_document"`
	CountsAsMissing        bool   `json:"counts_as_missing"`
	// Kondisi yang boleh dipilih buat status ini — minimal 1.
	ConditionIDs []uint `json:"condition_ids" binding:"required,min=1"`
}

type UpdateStockOpnamePhysicalStatusConditionsRequest struct {
	ConditionIDs []uint `json:"condition_ids" binding:"required,min=1"`
}

type StockOpnameConditionRef struct {
	ID    uint   `json:"id"`
	Code  string `json:"code"`
	Label string `json:"label"`
}

type StockOpnamePhysicalStatusMasterResponse struct {
	ID                     uint      `json:"id"`
	Code                   string    `json:"code"`
	Label                  string    `json:"label"`
	RequiresBorrowDocument bool      `json:"requires_borrow_document"`
	CountsAsMissing        bool      `json:"counts_as_missing"`
	IsSystem               bool      `json:"is_system"`
	CreatedAt              time.Time `json:"created_at"`
	// Kondisi yang boleh dipasangkan dengan status fisik ini (urut id).
	AllowedConditions []StockOpnameConditionRef `json:"allowed_conditions"`
}

// ReportBucket cuma boleh kosong (tidak dihitung), "BAIK", atau "RUSAK".
type CreateStockOpnameConditionMasterRequest struct {
	Code         string `json:"code" binding:"required,min=2,max=50"`
	Label        string `json:"label" binding:"required,min=1,max=100"`
	ReportBucket string `json:"report_bucket" binding:"omitempty,oneof=BAIK RUSAK"`
	// Opsional: langsung dipasangkan ke status fisik ini. Bisa diatur juga
	// belakangan dari sisi status fisik.
	PhysicalStatusIDs []uint `json:"physical_status_ids"`
}

type StockOpnameConditionMasterResponse struct {
	ID           uint      `json:"id"`
	Code         string    `json:"code"`
	Label        string    `json:"label"`
	ReportBucket string    `json:"report_bucket"`
	IsSystem     bool      `json:"is_system"`
	CreatedAt    time.Time `json:"created_at"`
}
