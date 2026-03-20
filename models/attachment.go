package models

import "time"

// ============================================================
// Constants
// ============================================================

const (
	AttachmentStageAll           = "ALL"
	AttachmentBranchAll          = "ALL"
	AttachmentTransactionTypeAll = "ALL"

	AttachmentStatusPending  = "PENDING"
	AttachmentStatusApproved = "APPROVED"
	AttachmentStatusRejected = "REJECTED"
)

// ============================================================
// AttachmentConfig — Master konfigurasi attachment
// ============================================================

type AttachmentConfig struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TransactionType string    `gorm:"size:50;not null;index" json:"transaction_type"` // procurement, mutation, ALL, dll
	Stage           string    `gorm:"size:50;not null;index" json:"stage"`            // stage nama atau ALL
	BranchCode      string    `gorm:"size:50;not null;index" json:"branch_code"`      // branch spesifik atau ALL
	AttachmentType  string    `gorm:"size:100;not null" json:"attachment_type"`       // SURAT_PENGAJUAN, KTP, dll
	Description     *string   `gorm:"type:text" json:"description"`
	IsRequired      bool      `gorm:"not null;default:true" json:"is_required"`
	IsActive        bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedBy       *string   `gorm:"size:100" json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (AttachmentConfig) TableName() string { return "attachment_configs" }

// ============================================================
// TransactionAttachment — Attachment per transaksi
// ============================================================

type TransactionAttachment struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	TransactionID      uint       `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber  string     `gorm:"size:100;not null;index" json:"transaction_number"`
	TransactionType    string     `gorm:"size:50;not null" json:"transaction_type"`
	Stage              string     `gorm:"size:50;not null;index" json:"stage"`
	AttachmentConfigID uint       `gorm:"not null;index" json:"attachment_config_id"`
	FileName           string     `gorm:"size:255;not null" json:"file_name"`
	FilePath           string     `gorm:"size:500;not null" json:"file_path"`
	FileSize           *int64     `json:"file_size"`
	MimeType           *string    `gorm:"size:100" json:"mime_type"`
	Status             string     `gorm:"type:enum('PENDING','APPROVED','REJECTED');not null;default:PENDING;index" json:"status"`
	UploadedBy         string     `gorm:"size:100;not null" json:"uploaded_by"`
	UploadedAt         time.Time  `gorm:"not null" json:"uploaded_at"`
	ReviewedBy         *string    `gorm:"size:100" json:"reviewed_by"`
	ReviewedAt         *time.Time `json:"reviewed_at"`
	RejectionReason    *string    `gorm:"type:text" json:"rejection_reason"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	// Relations
	Transaction      *Transaction      `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	AttachmentConfig *AttachmentConfig `gorm:"foreignKey:AttachmentConfigID" json:"attachment_config,omitempty"`
}

func (TransactionAttachment) TableName() string { return "transaction_attachments" }
