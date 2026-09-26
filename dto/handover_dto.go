package dto

import "time"

// ============================================================
// SERAH TERIMA ASET
// ============================================================

type CreateHandoverRequest struct {
	HandoverType string `json:"handover_type" binding:"required,oneof=HANDOVER RETURN"`
	// wajib untuk HANDOVER, diabaikan untuk RETURN
	ToUserID        *string `json:"to_user_id"`
	TransactionDate string  `json:"transaction_date"` // YYYY-MM-DD, kosong = hari ini
	Notes           *string `json:"notes"`
	AssetIDs        []uint  `json:"asset_ids" binding:"required,min=1"`
}

// Draft diganti utuh: penerima, catatan, dan daftar aset. Jenis serah terima
// tidak bisa diubah — buat ajuan baru kalau salah pilih.
type UpdateHandoverDraftRequest struct {
	ToUserID *string `json:"to_user_id"`
	Notes    *string `json:"notes"`
	AssetIDs []uint  `json:"asset_ids" binding:"required,min=1"`
}

type HandoverActionRequest struct {
	Notes *string `json:"notes"`
}

type RejectHandoverReceivingRequest struct {
	Reason string `json:"reason" binding:"required,min=10"`
}

type HandoverListFilter struct {
	// beberapa stage dipisah koma
	CurrentStage *string `form:"current_stage"`
	HandoverType *string `form:"handover_type"`
	Search       *string `form:"search"`
	StartDate    *string `form:"start_date"`
	EndDate      *string `form:"end_date"`
	WaitingForMe bool    `form:"waiting_for_me"`
	Page         int     `form:"page"`
	Limit        int     `form:"limit"`

	ViewerUserID string `form:"-"`
}

type HandoverAssetResponse struct {
	ID                  uint    `json:"id"`
	AssetID             uint    `json:"asset_id"`
	AssetNumber         string  `json:"asset_number"`
	AssetName           string  `json:"asset_name"`
	CategoryName        *string `json:"category_name,omitempty"`
	BranchCode          *string `json:"branch_code"`
	FromUserID          *string `json:"from_user_id"`
	FromUserName        *string `json:"from_user_name,omitempty"`
	PreviousAssetStatus string  `json:"previous_asset_status"`
	Status              string  `json:"status"`
	NeedsRevision       bool    `json:"needs_revision"`
	RevisionNotes       *string `json:"revision_notes"`
}

type HandoverDetailResponse struct {
	Transaction   TransactionHeaderResponse  `json:"transaction"`
	HandoverType  string                     `json:"handover_type"`
	ToUserID      *string                    `json:"to_user_id"`
	ToUserName    *string                    `json:"to_user_name,omitempty"`
	BranchCode    string                     `json:"branch_code"`
	Assets        []HandoverAssetResponse    `json:"assets"`
	Stages        []TransactionStageResponse `json:"stages"`
	TotalAssets   int                        `json:"total_assets"`
	NeedsRevision bool                       `json:"needs_revision"`
	WaitingForMe  bool                       `json:"waiting_for_me"`
	// true kalau user yang melihat adalah pihak yang berhak konfirmasi terima
	CanConfirmReceiving bool `json:"can_confirm_receiving"`
}

// Aset yang bisa dipilih saat membuat/mengubah ajuan
type HandoverEligibleAsset struct {
	AssetID          uint    `json:"asset_id"`
	AssetNumber      string  `json:"asset_number"`
	AssetName        string  `json:"asset_name"`
	CategoryName     *string `json:"category_name,omitempty"`
	AssetStatus      string  `json:"asset_status"`
	Location         *string `json:"location"`
	AssignedUserID   *string `json:"assigned_user_id"`
	AssignedUserName *string `json:"assigned_user_name,omitempty"`
}

type HandoverRecipient struct {
	UserID   string `json:"user_id"`
	Fullname string `json:"fullname"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type HandoverListItem struct {
	TransactionNumber string    `json:"transaction_number"`
	TransactionDate   time.Time `json:"transaction_date"`
	CurrentStage      string    `json:"current_stage"`
	Status            string    `json:"status"`
	HandoverType      string    `json:"handover_type"`
	ToUserName        *string   `json:"to_user_name,omitempty"`
	CreatedBy         string    `json:"created_by"`
	CreatedByName     *string   `json:"created_by_name,omitempty"`
	BranchCode        string    `json:"branch_code"`
	TotalAssets       int       `json:"total_assets"`
	NeedsRevision     bool      `json:"needs_revision"`
	CreatedAt         time.Time `json:"created_at"`
}
