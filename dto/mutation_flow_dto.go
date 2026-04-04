package dto

import "time"

// ============================================================
// CREATE MUTATION DRAFT
// ============================================================

type CreateMutationDraftRequest struct {
	TransactionDate string  `json:"transaction_date" binding:"required"`
	CategoryID      uint    `json:"category_id" binding:"required"`
	ToBranchCode    string  `json:"to_branch_code" binding:"required"`
	Notes           *string `json:"notes"`
}

// ============================================================
// ADD / REMOVE ASSET KE DRAFT
// ============================================================

type AddMutationAssetRequest struct {
	AssetID      uint    `json:"asset_id" binding:"required"`
	AssetNumber  string  `json:"asset_number" binding:"required"`
	FromLocation *string `json:"from_location"`
	ToLocation   *string `json:"to_location"`
	Notes        *string `json:"notes"`
}

type RemoveMutationAssetRequest struct {
	AssetID uint `json:"asset_id" binding:"required"`
}

// ============================================================
// SUBMIT
// ============================================================

type SubmitMutationRequest struct {
	Notes *string `json:"notes"`
}

// ============================================================
// EKSEKUSI
// ============================================================

type ExecuteMutationRequest struct {
	Notes *string `json:"notes"`
}

// ============================================================
// REJECT
// ============================================================

type RejectMutationRequest struct {
	Reason string `json:"reason" binding:"required,min=10"`
}

// ============================================================
// UPLOAD ATTACHMENT PER ASSET
// ============================================================

type UploadMutationAttachmentRequest struct {
	TransactionMutationAssetID uint `form:"transaction_mutation_asset_id" binding:"required"`
	AttachmentConfigID         uint `form:"attachment_config_id" binding:"required"`
}

type ReviewMutationAttachmentRequest struct {
	Status          string  `json:"status" binding:"required,oneof=APPROVED REJECTED"`
	RejectionReason *string `json:"rejection_reason"`
}

// ============================================================
// RESPONSES
// ============================================================

type MutationAssetResponse struct {
	ID                uint                         `json:"id"`
	TransactionID     uint                         `json:"transaction_id"`
	TransactionNumber string                       `json:"transaction_number"`
	AssetID           uint                         `json:"asset_id"`
	AssetNumber       string                       `json:"asset_number"`
	AssetName         *string                      `json:"asset_name,omitempty"`
	CategoryID        *uint                        `json:"category_id,omitempty"`
	CategoryName      *string                      `json:"category_name,omitempty"`
	FromBranchCode    string                       `json:"from_branch_code"`
	ToBranchCode      string                       `json:"to_branch_code"`
	FromLocation      *string                      `json:"from_location"`
	ToLocation        *string                      `json:"to_location"`
	DocumentNumber    *string                      `json:"document_number"`
	Notes             *string                      `json:"notes"`
	Status            string                       `json:"status"`
	Attachments       []MutationAttachmentResponse `json:"attachments,omitempty"`
	CreatedAt         time.Time                    `json:"created_at"`
	UpdatedAt         time.Time                    `json:"updated_at"`
}

type MutationAttachmentResponse struct {
	ID                         uint       `json:"id"`
	TransactionMutationAssetID uint       `json:"transaction_mutation_asset_id"`
	AssetID                    uint       `json:"asset_id"`
	AssetNumber                string     `json:"asset_number"`
	AttachmentConfigID         uint       `json:"attachment_config_id"`
	AttachmentType             *string    `json:"attachment_type,omitempty"`
	IsRequired                 *bool      `json:"is_required,omitempty"`
	FileName                   string     `json:"file_name"`
	FilePath                   string     `json:"file_path"`
	FileSize                   *int64     `json:"file_size"`
	MimeType                   *string    `json:"mime_type"`
	Status                     string     `json:"status"`
	UploadedBy                 string     `json:"uploaded_by"`
	UploadedAt                 time.Time  `json:"uploaded_at"`
	ReviewedBy                 *string    `json:"reviewed_by"`
	ReviewedAt                 *time.Time `json:"reviewed_at"`
	RejectionReason            *string    `json:"rejection_reason"`
	CreatedAt                  time.Time  `json:"created_at"`
}

type MutationTransactionResponse struct {
	ID                uint      `json:"id"`
	TransactionNumber string    `json:"transaction_number"`
	TransactionType   string    `json:"transaction_type"`
	TransactionDate   time.Time `json:"transaction_date"`
	Status            string    `json:"status"`
	CurrentStage      string    `json:"current_stage"`
	CategoryID        *uint     `json:"category_id"`
	CategoryName      *string   `json:"category_name,omitempty"`
	ToBranchCode      *string   `json:"to_branch_code"`
	Notes             *string   `json:"notes"`
	CreatedBy         string    `json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type MutationDetailResponse struct {
	Transaction MutationTransactionResponse `json:"transaction"`
	Assets      []MutationAssetResponse     `json:"assets"`
	Stages      []TransactionStageResponse  `json:"stages"`
}

// ============================================================
// ATTACHMENT STATUS SUMMARY PER ASSET
// ============================================================

type MutationAttachmentStatusSummary struct {
	TransactionNumber string                       `json:"transaction_number"`
	AssetID           uint                         `json:"asset_id"`
	AssetNumber       string                       `json:"asset_number"`
	CanProceed        bool                         `json:"can_proceed"`
	TotalRequired     int                          `json:"total_required"`
	TotalApproved     int                          `json:"total_approved"`
	TotalPending      int                          `json:"total_pending"`
	TotalRejected     int                          `json:"total_rejected"`
	Attachments       []MutationAttachmentResponse `json:"attachments"`
}

type MutationAllAttachmentStatus struct {
	TransactionNumber string                            `json:"transaction_number"`
	AllCanProceed     bool                              `json:"all_can_proceed"` // true kalau semua asset attachmentnya OK
	Assets            []MutationAttachmentStatusSummary `json:"assets"`
}
