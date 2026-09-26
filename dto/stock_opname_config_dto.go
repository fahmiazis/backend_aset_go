package dto

import "time"

// ============================================================
// CONFIG STOCK OPNAME
//
// Jendela submit HANYA nentuin apakah sebuah submit dianggap "on schedule"
// (IsSubmissive) — tidak pernah memblokir submit itu sendiri. Semua stock
// opname tetap bisa di-submit kapan pun.
// ============================================================

type StockOpnameConfigResponse struct {
	SubmissionStartDay int `json:"submission_start_day"`
	SubmissionEndDay   int `json:"submission_end_day"`

	BorrowDocAllowPDF   bool `json:"borrow_doc_allow_pdf"`
	BorrowDocAllowWord  bool `json:"borrow_doc_allow_word"`
	BorrowDocAllowPhoto bool `json:"borrow_doc_allow_photo"`
	BorrowDocIsRequired bool `json:"borrow_doc_is_required"`

	UpdatedBy *string   `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateStockOpnameConfigRequest — BorrowDocAllow* sengaja bukan pointer:
// kalau semuanya false berarti gak ada format yang diterima sama sekali,
// itu tanggung jawab admin yang ngisi form, bukan divalidasi di sini (biar
// gak keluar error yang membingungkan buat kasus valid tapi jarang kepake).
type UpdateStockOpnameConfigRequest struct {
	SubmissionStartDay int `json:"submission_start_day" binding:"required,min=1,max=31"`
	SubmissionEndDay   int `json:"submission_end_day" binding:"required,min=1,max=31"`

	BorrowDocAllowPDF   bool `json:"borrow_doc_allow_pdf"`
	BorrowDocAllowWord  bool `json:"borrow_doc_allow_word"`
	BorrowDocAllowPhoto bool `json:"borrow_doc_allow_photo"`
	BorrowDocIsRequired bool `json:"borrow_doc_is_required"`
}
