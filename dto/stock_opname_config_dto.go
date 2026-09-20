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
	SubmissionStartDay int       `json:"submission_start_day"`
	SubmissionEndDay   int       `json:"submission_end_day"`
	UpdatedBy          *string   `json:"updated_by"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type UpdateStockOpnameConfigRequest struct {
	SubmissionStartDay int `json:"submission_start_day" binding:"required,min=1,max=31"`
	SubmissionEndDay   int `json:"submission_end_day" binding:"required,min=1,max=31"`
}
