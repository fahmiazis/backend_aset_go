package models

import "time"

// ============================================================
// Constants — Stage
// ============================================================

const (
	StageDisposalDraft             = "DRAFT"
	StageDisposalSubmitted         = "SUBMITTED"
	StageDisposalPurchasing        = "PURCHASING" // SELL only
	StageDisposalApprovalRequest   = "APPROVAL_REQUEST"
	StageDisposalApprovalAgreement = "APPROVAL_AGREEMENT"
	StageDisposalExecute           = "EXECUTE"
	StageDisposalFinance           = "FINANCE" // SELL only
	StageDisposalTax               = "TAX"     // SELL only
	StageDisposalAssetDeletion     = "ASSET_DELETION"
	StageDisposalFinished          = "FINISHED"
	StageDisposalRejected          = "REJECTED"
)

// ============================================================
// Constants — Disposal Type
// ============================================================

const (
	DisposalTypeDispose = "DISPOSE"
	DisposalTypeSell    = "SELL"
)

// ============================================================
// Constants — Disposal Asset Status
// ============================================================

const (
	DisposalAssetStatusPending   = "PENDING"
	DisposalAssetStatusDeleted   = "DELETED"
	DisposalAssetStatusCancelled = "CANCELLED"
)

// ============================================================
// Constants — Asset Status (tambahan untuk disposal)
// ============================================================

const AssetStatusInDisposal = "IN_DISPOSAL"

// ============================================================
// Constants — Approval Flow Code
// ============================================================

const (
	FlowDisposalApprovalRequest   = "DISPOSAL_APPROVAL_REQUEST"
	FlowDisposalApprovalAgreement = "DISPOSAL_APPROVAL_AGREEMENT"
)

// ============================================================
// TransactionDisposalAsset
// Asset yang dimasukkan ke dalam draft disposal
// ============================================================

type TransactionDisposalAsset struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	TransactionID     uint      `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber string    `gorm:"size:100;not null;index" json:"transaction_number"`
	AssetID           uint      `gorm:"not null;index" json:"asset_id"`
	AssetNumber       string    `gorm:"size:100;not null" json:"asset_number"`
	DisposalType      string    `gorm:"size:20;not null" json:"disposal_type"`
	DisposalReason    *string   `gorm:"type:text" json:"disposal_reason"`
	SaleValue         *float64  `gorm:"type:decimal(18,2)" json:"sale_value"` // diisi purchasing (SELL only)
	DocumentNumber    *string   `gorm:"size:50" json:"document_number"`       // generated saat asset deletion
	Notes             *string   `gorm:"type:text" json:"notes"`
	Status            string    `gorm:"type:enum('PENDING','DELETED','CANCELLED');not null;default:PENDING;index" json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// Relations
	Transaction *Transaction                    `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	Asset       *Asset                          `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
	Attachments []TransactionDisposalAttachment `gorm:"foreignKey:TransactionDisposalAssetID" json:"attachments,omitempty"`
}

func (TransactionDisposalAsset) TableName() string { return "transaction_disposal_assets" }

// ============================================================
// TransactionDisposalAttachment
// Attachment per asset per stage di disposal
// ============================================================

type TransactionDisposalAttachment struct {
	ID                         uint       `gorm:"primaryKey" json:"id"`
	TransactionID              uint       `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber          string     `gorm:"size:100;not null;index" json:"transaction_number"`
	TransactionDisposalAssetID uint       `gorm:"not null;index" json:"transaction_disposal_asset_id"`
	AssetID                    uint       `gorm:"not null;index" json:"asset_id"`
	AssetNumber                string     `gorm:"size:100;not null" json:"asset_number"`
	AttachmentConfigID         uint       `gorm:"not null;index" json:"attachment_config_id"`
	Stage                      string     `gorm:"size:50;not null;index" json:"stage"` // stage saat diupload
	FileName                   string     `gorm:"size:255;not null" json:"file_name"`
	FilePath                   string     `gorm:"size:500;not null" json:"file_path"`
	FileSize                   *int64     `json:"file_size"`
	MimeType                   *string    `gorm:"size:100" json:"mime_type"`
	Status                     string     `gorm:"type:enum('PENDING','APPROVED','REJECTED');not null;default:PENDING;index" json:"status"`
	UploadedBy                 string     `gorm:"size:100;not null" json:"uploaded_by"`
	UploadedAt                 time.Time  `gorm:"not null" json:"uploaded_at"`
	ReviewedBy                 *string    `gorm:"size:100" json:"reviewed_by"`
	ReviewedAt                 *time.Time `json:"reviewed_at"`
	RejectionReason            *string    `gorm:"type:text" json:"rejection_reason"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`

	// Relations
	Transaction              *Transaction              `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	TransactionDisposalAsset *TransactionDisposalAsset `gorm:"foreignKey:TransactionDisposalAssetID" json:"disposal_asset,omitempty"`
	AttachmentConfig         *AttachmentConfig         `gorm:"foreignKey:AttachmentConfigID" json:"attachment_config,omitempty"`
}

func (TransactionDisposalAttachment) TableName() string { return "transaction_disposal_attachments" }

// ============================================================
// Tambahan field di struct Transaction (reference — tidak ditulis ulang):
//
// DisposalType              *string  `gorm:"size:20;index" json:"disposal_type"`
// SaleValue                 *float64 `gorm:"type:decimal(18,2)" json:"sale_value"`
// ApprovalRequestNumber     *string  `gorm:"size:100" json:"approval_request_number"`
// ApprovalAgreementNumber   *string  `gorm:"size:100" json:"approval_agreement_number"`
// ============================================================
