package dto

import "time"

type TransactionHeaderResponse struct {
	ID                uint       `json:"id"`
	TransactionNumber string     `json:"transaction_number"`
	TransactionType   string     `json:"transaction_type"`
	TransactionDate   time.Time  `json:"transaction_date"`
	CurrentStage      string     `json:"current_stage"`
	Status            string     `json:"status"`
	Notes             *string    `json:"notes"`
	CreatedBy         string     `json:"created_by"`
	CreatedByName     *string    `json:"created_by_name"`
	ApprovedBy        *string    `json:"approved_by"`
	ApprovedAt        *time.Time `json:"approved_at"`

	// true kalau ada baris yang ditandai approver perlu diperbaiki. Ditaruh di
	// level transaksi supaya daftar bisa menandainya tanpa memuat seluruh
	// barisnya — pengajuan yang dikembalikan revisi tampak persis seperti draft
	// biasa kalau tidak ditandai.
	NeedsRevision bool `json:"needs_revision"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TransactionListFilter struct {
	TransactionType *string `form:"transaction_type" binding:"omitempty,oneof=PROCUREMENT MUTATION DISPOSAL STOCK_OPNAME"`
	Status          *string `form:"status"`
	CurrentStage    *string `form:"current_stage"` // ADD: filter by stage
	CreatedBy       *string `form:"created_by"`    // ADD: filter by creator (UUID)
	StartDate       *string `form:"start_date"`
	EndDate         *string `form:"end_date"`
	Page            int     `form:"page" binding:"min=1"`
	Limit           int     `form:"limit" binding:"min=1,max=100"`

	// Hanya pengajuan yang menunggu tindakan user yang sedang login.
	WaitingForMe bool `form:"waiting_for_me"`
	// Diisi controller dari token, bukan dari query — kalau boleh dikirim
	// client, siapa pun bisa mengintip daftar tugas orang lain.
	ViewerUserID string `form:"-"`
}

// ============================================================
// REVISI & PEMBATALAN — bentuknya sama untuk procurement dan mutation
// ============================================================

// ReturnForRevisionRequest dikirim APPROVER step berjalan untuk
// mengembalikan pengajuan ke DRAFT.
type ReturnForRevisionRequest struct {
	RevisionNotes string `json:"revision_notes" binding:"required,min=10"`
	// Baris yang perlu diperbaiki: item procurement / aset mutasi. Wajib diisi
	// supaya pengaju tahu bagian mana yang salah, bukan sekadar "ada yang salah".
	RowIDs []uint `json:"row_ids" binding:"required,min=1"`
}

// CancelTransactionRequest dikirim PEMBUAT pengajuan untuk membatalkan
// pengajuannya sendiri.
type CancelTransactionRequest struct {
	Reason string `json:"reason" binding:"required,min=10"`
}
