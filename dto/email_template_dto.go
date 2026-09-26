package dto

import "time"

// ============================================================
// EMAIL TEMPLATE DTOs
// ============================================================

type CreateEmailTemplateRequest struct {
	TransactionType string   `json:"transaction_type" binding:"required,oneof=procurement mutation disposal disposal_agreement handover"`
	Stage           string   `json:"stage" binding:"required"`
	Action          string   `json:"action" binding:"required,oneof=proceed reject revise cancel"`
	Subject         string   `json:"subject" binding:"required,max=255"`
	Body            string   `json:"body" binding:"required"`
	CCRoleIDs       []string `json:"cc_role_ids"`
	IsActive        bool     `json:"is_active"`
}

// Jenis transaksi, stage, dan aksi adalah kunci template — tidak bisa diubah.
// Kalau salah pilih, hapus lalu buat baru.
type UpdateEmailTemplateRequest struct {
	Subject   *string   `json:"subject" binding:"omitempty,max=255"`
	Body      *string   `json:"body"`
	CCRoleIDs *[]string `json:"cc_role_ids"`
	IsActive  *bool     `json:"is_active"`
}

type EmailTemplateRoleResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type EmailTemplateResponse struct {
	ID              uint                        `json:"id"`
	TransactionType string                      `json:"transaction_type"`
	Stage           string                      `json:"stage"`
	Action          string                      `json:"action"`
	Subject         string                      `json:"subject"`
	Body            string                      `json:"body"`
	IsActive        bool                        `json:"is_active"`
	CCRoles         []EmailTemplateRoleResponse `json:"cc_roles"`
	CreatedBy       *string                     `json:"created_by"`
	CreatedAt       time.Time                   `json:"created_at"`
	UpdatedAt       time.Time                   `json:"updated_at"`
}

// ============================================================
// DIALOG EMAIL (preview & kirim)
// ============================================================

type EmailRecipient struct {
	UserID string `json:"user_id,omitempty"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

// Isi dialog sebelum aksi dijalankan. HasTemplate = false berarti stage/aksi
// ini tidak dikonfigurasi dan aksinya langsung dijalankan tanpa dialog.
type EmailPreviewResponse struct {
	HasTemplate       bool   `json:"has_template"`
	TemplateID        uint   `json:"template_id,omitempty"`
	TransactionNumber string `json:"transaction_number"`
	TransactionType   string `json:"transaction_type"`
	Stage             string `json:"stage"`
	NextStage         string `json:"next_stage"`
	Action            string `json:"action"`
	Subject           string `json:"subject"`
	Body              string `json:"body"`
	// HTML email lengkap (layout + info + daftar aset) tanpa pesan tambahan,
	// untuk pratinjau di dialog
	HTML string           `json:"html"`
	To   []EmailRecipient `json:"to"`
	CC   []EmailRecipient `json:"cc"`
	// true kalau SMTP belum dikonfigurasi di .env — dialog tetap tampil tapi
	// memberi tahu bahwa email akan tercatat gagal.
	MailerConfigured bool `json:"mailer_configured"`
}

// Dikirim SETELAH aksinya berhasil. Subject tidak ikut dikirim: selalu
// dirender ulang dari template di server, karena memang dikunci.
type SendTransactionEmailRequest struct {
	TransactionType   string   `json:"transaction_type" binding:"required"`
	TransactionNumber string   `json:"transaction_number" binding:"required"`
	TemplateID        uint     `json:"template_id" binding:"required"`
	To                []string `json:"to" binding:"required,min=1,dive,required,email"`
	CC                []string `json:"cc" binding:"required,min=1,dive,required,email"`
	AdditionalMessage string   `json:"additional_message"`
}

type TransactionEmailLogResponse struct {
	ID                uint       `json:"id"`
	EmailTemplateID   *uint      `json:"email_template_id"`
	TransactionNumber string     `json:"transaction_number"`
	TransactionType   string     `json:"transaction_type"`
	Stage             string     `json:"stage"`
	Action            string     `json:"action"`
	Subject           string     `json:"subject"`
	To                []string   `json:"to"`
	CC                []string   `json:"cc"`
	Status            string     `json:"status"`
	ErrorMessage      *string    `json:"error_message"`
	Attempts          int        `json:"attempts"`
	SentBy            string     `json:"sent_by"`
	SentByName        *string    `json:"sent_by_name"`
	SentAt            *time.Time `json:"sent_at"`
	CreatedAt         time.Time  `json:"created_at"`
}
