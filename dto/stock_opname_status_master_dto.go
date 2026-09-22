package dto

import "time"

// ============================================================
// MASTER DATA STATUS FISIK & KONDISI STOCK OPNAME
//
// Sengaja CREATE + LIST + SOFT DELETE saja (tidak ada UPDATE) — flag
// perilaku (requires_borrow_document dkk) ditentukan sekali saat create
// supaya tidak berubah di tengah jalan transaksi yang sudah pakai kode itu.
// Baris bawaan sistem (is_system=true) tidak bisa dihapus.
// ============================================================

type CreateStockOpnamePhysicalStatusMasterRequest struct {
	Code                           string `json:"code" binding:"required,min=2,max=50"`
	Label                          string `json:"label" binding:"required,min=1,max=100"`
	RequiresBorrowDocument         bool   `json:"requires_borrow_document"`
	RequiresNotApplicableCondition bool   `json:"requires_not_applicable_condition"`
	CountsAsMissing                bool   `json:"counts_as_missing"`
}

type StockOpnamePhysicalStatusMasterResponse struct {
	ID                             uint      `json:"id"`
	Code                           string    `json:"code"`
	Label                          string    `json:"label"`
	RequiresBorrowDocument         bool      `json:"requires_borrow_document"`
	RequiresNotApplicableCondition bool      `json:"requires_not_applicable_condition"`
	CountsAsMissing                bool      `json:"counts_as_missing"`
	IsSystem                       bool      `json:"is_system"`
	CreatedAt                      time.Time `json:"created_at"`
}

// ReportBucket cuma boleh kosong (tidak dihitung), "BAIK", atau "RUSAK".
type CreateStockOpnameConditionMasterRequest struct {
	Code                 string `json:"code" binding:"required,min=2,max=50"`
	Label                string `json:"label" binding:"required,min=1,max=100"`
	IsNotApplicableValue bool   `json:"is_not_applicable_value"`
	ReportBucket         string `json:"report_bucket" binding:"omitempty,oneof=BAIK RUSAK"`
}

type StockOpnameConditionMasterResponse struct {
	ID                   uint      `json:"id"`
	Code                 string    `json:"code"`
	Label                string    `json:"label"`
	IsNotApplicableValue bool      `json:"is_not_applicable_value"`
	ReportBucket         string    `json:"report_bucket"`
	IsSystem             bool      `json:"is_system"`
	CreatedAt            time.Time `json:"created_at"`
}
