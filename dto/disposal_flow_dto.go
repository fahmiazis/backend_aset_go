package dto

import "time"

// ============================================================
// CREATE DISPOSAL DRAFT
// ============================================================

type CreateDisposalDraftRequest struct {
	TransactionDate string  `json:"transaction_date" binding:"required"`
	DisposalType    string  `json:"disposal_type" binding:"required,oneof=DISPOSE SELL"`
	Notes           *string `json:"notes"`
}

// ============================================================
// ADD / REMOVE ASSET
// ============================================================

type AddDisposalAssetRequest struct {
	AssetID        uint    `json:"asset_id" binding:"required"`
	AssetNumber    string  `json:"asset_number" binding:"required"`
	DisposalReason *string `json:"disposal_reason"`
	Notes          *string `json:"notes"`
}

type RemoveDisposalAssetRequest struct {
	AssetID uint `json:"asset_id" binding:"required"`
}

// ============================================================
// SUBMIT (DRAFT → SUBMITTED)
// ============================================================

type SubmitDisposalRequest struct {
	Notes *string `json:"notes"`
}

// ============================================================
// PURCHASING — upload sale_value per asset (SELL only)
// ============================================================

type SetDisposalSaleValueRequest struct {
	Assets []DisposalAssetSaleValue `json:"assets" binding:"required,min=1,dive"`
	Notes  *string                  `json:"notes"`
}

type DisposalAssetSaleValue struct {
	DisposalAssetID uint    `json:"disposal_asset_id" binding:"required"`
	SaleValue       float64 `json:"sale_value" binding:"required,gt=0"`
}

// ============================================================
// INITIATE APPROVAL REQUEST / APPROVAL AGREEMENT
// ============================================================

type InitiateDisposalApprovalRequest struct {
	Metadata map[string]interface{} `json:"metadata"`
}

// ============================================================
// EXECUTE (creator upload dok penghapusan/hasil jual)
// ============================================================

type ExecuteDisposalRequest struct {
	Notes *string `json:"notes"`
}

// ============================================================
// FINANCE — validasi + upload
// ============================================================

type ConfirmDisposalFinanceRequest struct {
	Notes *string `json:"notes"`
}

// ============================================================
// TAX — validasi + upload
// ============================================================

type ConfirmDisposalTaxRequest struct {
	Notes *string `json:"notes"`
}

// ============================================================
// ASSET DELETION — tim asset hapus asset + generate doc number
// ============================================================

type ConfirmDisposalAssetDeletionRequest struct {
	Notes *string `json:"notes"`
}

// ============================================================
// REJECT
// ============================================================

type RejectDisposalRequest struct {
	Reason string `json:"reason" binding:"required,min=10"`
}

// ============================================================
// UPLOAD ATTACHMENT
// ============================================================

type UploadDisposalAttachmentRequest struct {
	TransactionDisposalAssetID uint   `form:"transaction_disposal_asset_id" binding:"required"`
	AttachmentConfigID         uint   `form:"attachment_config_id" binding:"required"`
	Stage                      string `form:"stage" binding:"required"`
}

// ============================================================
// REVIEW ATTACHMENT
// ============================================================

type ReviewDisposalAttachmentRequest struct {
	Status          string  `json:"status" binding:"required,oneof=APPROVED REJECTED"`
	RejectionReason *string `json:"rejection_reason"`
}

// ============================================================
// RESPONSES
// ============================================================

type DisposalTransactionResponse struct {
	ID                      uint      `json:"id"`
	TransactionNumber       string    `json:"transaction_number"`
	TransactionType         string    `json:"transaction_type"`
	TransactionDate         time.Time `json:"transaction_date"`
	Status                  string    `json:"status"`
	CurrentStage            string    `json:"current_stage"`
	DisposalType            *string   `json:"disposal_type"`
	SaleValue               *float64  `json:"sale_value"`
	ApprovalRequestNumber   *string   `json:"approval_request_number"`
	ApprovalAgreementNumber *string   `json:"approval_agreement_number"`
	Notes                   *string   `json:"notes"`
	CreatedBy               string    `json:"created_by"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type DisposalAssetResponse struct {
	ID                uint                         `json:"id"`
	TransactionID     uint                         `json:"transaction_id"`
	TransactionNumber string                       `json:"transaction_number"`
	AssetID           uint                         `json:"asset_id"`
	AssetNumber       string                       `json:"asset_number"`
	AssetName         *string                      `json:"asset_name,omitempty"`
	CategoryID        *uint                        `json:"category_id,omitempty"`
	CategoryName      *string                      `json:"category_name,omitempty"`
	BranchCode        *string                      `json:"branch_code,omitempty"`
	DisposalType      string                       `json:"disposal_type"`
	DisposalReason    *string                      `json:"disposal_reason"`
	SaleValue         *float64                     `json:"sale_value"`
	DocumentNumber    *string                      `json:"document_number"`
	Notes             *string                      `json:"notes"`
	Status            string                       `json:"status"`
	Attachments       []DisposalAttachmentResponse `json:"attachments,omitempty"`
	CreatedAt         time.Time                    `json:"created_at"`
	UpdatedAt         time.Time                    `json:"updated_at"`
}

type DisposalAttachmentResponse struct {
	ID                         uint       `json:"id"`
	TransactionDisposalAssetID uint       `json:"transaction_disposal_asset_id"`
	AssetID                    uint       `json:"asset_id"`
	AssetNumber                string     `json:"asset_number"`
	AttachmentConfigID         uint       `json:"attachment_config_id"`
	AttachmentType             *string    `json:"attachment_type,omitempty"`
	IsRequired                 *bool      `json:"is_required,omitempty"`
	Stage                      string     `json:"stage"`
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

type DisposalDetailResponse struct {
	Transaction DisposalTransactionResponse `json:"transaction"`
	Assets      []DisposalAssetResponse     `json:"assets"`
	Stages      []TransactionStageResponse  `json:"stages"`
}

// ============================================================
// ATTACHMENT STATUS SUMMARY
// ============================================================

type DisposalAttachmentStatusSummary struct {
	AssetID       uint                         `json:"asset_id"`
	AssetNumber   string                       `json:"asset_number"`
	Stage         string                       `json:"stage"`
	CanProceed    bool                         `json:"can_proceed"`
	TotalRequired int                          `json:"total_required"`
	TotalApproved int                          `json:"total_approved"`
	TotalPending  int                          `json:"total_pending"`
	TotalRejected int                          `json:"total_rejected"`
	Attachments   []DisposalAttachmentResponse `json:"attachments"`
}

type DisposalAllAttachmentStatus struct {
	TransactionNumber string                            `json:"transaction_number"`
	Stage             string                            `json:"stage"`
	AllCanProceed     bool                              `json:"all_can_proceed"`
	Assets            []DisposalAttachmentStatusSummary `json:"assets"`
}

// ============================================================
// LIST FILTER
// ============================================================

type DisposalListFilter struct {
	DisposalType *string `form:"disposal_type"`
	Status       *string `form:"status"`
	CurrentStage *string `form:"current_stage"`
	CreatedBy    *string `form:"created_by"`
	StartDate    *string `form:"start_date"`
	EndDate      *string `form:"end_date"`
	Page         int     `form:"page"`
	Limit        int     `form:"limit"`
}
