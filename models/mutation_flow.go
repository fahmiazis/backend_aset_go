package models

import "time"

// ============================================================
// Constants
// ============================================================

const (
	StageMutationDraft     = "DRAFT"
	StageMutationApproval  = "APPROVAL"
	StageMutationReceiving = "MUTATION_RECEIVING" // branch tujuan upload dok serah terima
	StageMutationExecute   = "EXECUTE_MUTATION"   // PIC Asset eksekusi perpindahan
	StageMutationFinished  = "FINISHED"
	StageMutationRejected  = "REJECTED"
)

const (
	MutationAssetStatusPending   = "PENDING"
	MutationAssetStatusExecuted  = "EXECUTED"
	MutationAssetStatusCancelled = "CANCELLED"
)

const AssetStatusInMutation = "IN_MUTATION"

// ============================================================
// TransactionMutationAsset
// Asset yang dimasukkan ke dalam draft mutasi
// ============================================================

type TransactionMutationAsset struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	TransactionID     uint      `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber string    `gorm:"size:100;not null;index" json:"transaction_number"`
	AssetID           uint      `gorm:"not null;index" json:"asset_id"`
	AssetNumber       string    `gorm:"size:100;not null" json:"asset_number"`
	FromBranchCode    string    `gorm:"size:50;not null" json:"from_branch_code"`
	ToBranchCode      string    `gorm:"size:50;not null" json:"to_branch_code"`
	FromLocation      *string   `gorm:"size:255" json:"from_location"`
	ToLocation        *string   `gorm:"size:255" json:"to_location"`
	DocumentNumber    *string   `gorm:"size:50" json:"document_number"` // generated saat eksekusi
	Notes             *string   `gorm:"type:text" json:"notes"`
	Status            string    `gorm:"type:enum('PENDING','EXECUTED','CANCELLED');not null;default:PENDING;index" json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// Relations
	Transaction *Transaction                    `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	Asset       *Asset                          `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
	Attachments []TransactionMutationAttachment `gorm:"foreignKey:TransactionMutationAssetID" json:"attachments,omitempty"`
}

func (TransactionMutationAsset) TableName() string { return "transaction_mutation_assets" }

// ============================================================
// TransactionMutationAttachment
// Attachment per asset di mutasi
// ============================================================

type TransactionMutationAttachment struct {
	ID                         uint       `gorm:"primaryKey" json:"id"`
	TransactionID              uint       `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber          string     `gorm:"size:100;not null;index" json:"transaction_number"`
	TransactionMutationAssetID uint       `gorm:"not null;index" json:"transaction_mutation_asset_id"`
	AssetID                    uint       `gorm:"not null;index" json:"asset_id"`
	AssetNumber                string     `gorm:"size:100;not null" json:"asset_number"`
	AttachmentConfigID         uint       `gorm:"not null;index" json:"attachment_config_id"`
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
	TransactionMutationAsset *TransactionMutationAsset `gorm:"foreignKey:TransactionMutationAssetID" json:"mutation_asset,omitempty"`
	AttachmentConfig         *AttachmentConfig         `gorm:"foreignKey:AttachmentConfigID" json:"attachment_config,omitempty"`
}

func (TransactionMutationAttachment) TableName() string { return "transaction_mutation_attachments" }

// ============================================================
// Update Transaction model — tambah mutation fields
// ============================================================
// Tambahkan ke struct Transaction yang sudah ada:
// MutationCategoryID   *uint   `gorm:"index" json:"mutation_category_id"`
// MutationToBranchCode *string `gorm:"size:50;index" json:"mutation_to_branch_code"`
