package dto

import "time"

// ============================================================
// ATTACHMENT CONFIG DTOs
// ============================================================

type CreateAttachmentConfigRequest struct {
	TransactionType string  `json:"transaction_type" binding:"required"`
	Stage           string  `json:"stage" binding:"required"`       // stage spesifik atau ALL
	BranchCode      string  `json:"branch_code" binding:"required"` // branch spesifik atau ALL
	AttachmentType  string  `json:"attachment_type" binding:"required"`
	Description     *string `json:"description"`
	IsRequired      bool    `json:"is_required"`
	IsActive        bool    `json:"is_active"`
}

type UpdateAttachmentConfigRequest struct {
	Description *string `json:"description"`
	IsRequired  *bool   `json:"is_required"`
	IsActive    *bool   `json:"is_active"`
}

type AttachmentConfigResponse struct {
	ID              uint      `json:"id"`
	TransactionType string    `json:"transaction_type"`
	Stage           string    `json:"stage"`
	BranchCode      string    `json:"branch_code"`
	AttachmentType  string    `json:"attachment_type"`
	Description     *string   `json:"description"`
	IsRequired      bool      `json:"is_required"`
	IsActive        bool      `json:"is_active"`
	CreatedBy       *string   `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ============================================================
// TRANSACTION ATTACHMENT DTOs
// ============================================================

type UploadAttachmentRequest struct {
	AttachmentConfigID uint `form:"attachment_config_id" binding:"required"`
	// File dihandle via multipart form
}

type ReviewAttachmentRequest struct {
	Status          string  `json:"status" binding:"required,oneof=APPROVED REJECTED"`
	RejectionReason *string `json:"rejection_reason"` // wajib kalau REJECTED
}

type TransactionAttachmentResponse struct {
	ID                 uint       `json:"id"`
	TransactionID      uint       `json:"transaction_id"`
	TransactionNumber  string     `json:"transaction_number"`
	TransactionType    string     `json:"transaction_type"`
	Stage              string     `json:"stage"`
	AttachmentConfigID uint       `json:"attachment_config_id"`
	AttachmentType     *string    `json:"attachment_type,omitempty"`
	IsRequired         *bool      `json:"is_required,omitempty"`
	FileName           string     `json:"file_name"`
	FilePath           string     `json:"file_path"`
	FileSize           *int64     `json:"file_size"`
	MimeType           *string    `json:"mime_type"`
	Status             string     `json:"status"`
	UploadedBy         string     `json:"uploaded_by"`
	UploadedAt         time.Time  `json:"uploaded_at"`
	ReviewedBy         *string    `json:"reviewed_by"`
	ReviewedAt         *time.Time `json:"reviewed_at"`
	RejectionReason    *string    `json:"rejection_reason"`
	CreatedAt          time.Time  `json:"created_at"`
}

// ============================================================
// ATTACHMENT STATUS SUMMARY
// Dipakai untuk cek apakah semua required attachment sudah APPROVED
// sebelum bisa lanjut ke stage berikutnya
// ============================================================

type AttachmentStatusSummary struct {
	TransactionNumber string                          `json:"transaction_number"`
	Stage             string                          `json:"stage"`
	CanProceed        bool                            `json:"can_proceed"` // true kalau semua required sudah APPROVED
	TotalRequired     int                             `json:"total_required"`
	TotalApproved     int                             `json:"total_approved"`
	TotalPending      int                             `json:"total_pending"`
	TotalRejected     int                             `json:"total_rejected"`
	MissingRequired   []AttachmentConfigResponse      `json:"missing_required"` // config yang belum diupload
	Attachments       []TransactionAttachmentResponse `json:"attachments"`
}
