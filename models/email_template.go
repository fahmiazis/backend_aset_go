package models

import "time"

// ============================================================
// Constants
// ============================================================

// Aksi yang memicu email. Template dibedakan per aksi karena penerima dan
// isinya berbeda: "proceed" diteruskan ke pengerja stage berikutnya, sementara
// reject/revise dikembalikan ke pengaju.
const (
	EmailActionProceed = "proceed"
	EmailActionReject  = "reject"
	EmailActionRevise  = "revise"
	EmailActionCancel  = "cancel"

	EmailStatusSent   = "SENT"
	EmailStatusFailed = "FAILED"
)

// ============================================================
// EmailTemplate — template email per transaksi, stage, dan aksi
// ============================================================

type EmailTemplate struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TransactionType string    `gorm:"size:50;not null" json:"transaction_type"`
	Stage           string    `gorm:"size:50;not null" json:"stage"` // stage asal saat aksi dilakukan
	Action          string    `gorm:"type:enum('proceed','reject','revise','cancel');not null;default:proceed" json:"action"`
	Subject         string    `gorm:"size:255;not null" json:"subject"`
	Body            string    `gorm:"type:text;not null" json:"body"`
	IsActive        bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedBy       *string   `gorm:"size:100" json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	CCRoles []EmailTemplateCCRole `gorm:"foreignKey:EmailTemplateID" json:"cc_roles,omitempty"`
}

func (EmailTemplate) TableName() string { return "email_templates" }

type EmailTemplateCCRole struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	EmailTemplateID uint      `gorm:"not null" json:"email_template_id"`
	RoleID          string    `gorm:"type:char(36);not null" json:"role_id"`
	CreatedAt       time.Time `json:"created_at"`
}

func (EmailTemplateCCRole) TableName() string { return "email_template_cc_roles" }

// ============================================================
// TransactionEmailLog — jejak email yang dikirim dari dialog aksi
// ============================================================

type TransactionEmailLog struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	EmailTemplateID   *uint      `json:"email_template_id"`
	TransactionNumber string     `gorm:"size:100;not null" json:"transaction_number"`
	TransactionType   string     `gorm:"size:50;not null" json:"transaction_type"`
	Stage             string     `gorm:"size:50;not null" json:"stage"`
	Action            string     `gorm:"size:20;not null" json:"action"`
	Subject           string     `gorm:"size:255;not null" json:"subject"`
	Body              string     `gorm:"type:mediumtext;not null" json:"body"`
	ToEmails          string     `gorm:"type:text;not null" json:"to_emails"` // JSON array
	CCEmails          string     `gorm:"type:text;not null" json:"cc_emails"` // JSON array
	Status            string     `gorm:"type:enum('SENT','FAILED');not null" json:"status"`
	ErrorMessage      *string    `gorm:"type:text" json:"error_message"`
	Attempts          int        `gorm:"not null;default:1" json:"attempts"`
	SentBy            string     `gorm:"size:100;not null" json:"sent_by"`
	SentAt            *time.Time `json:"sent_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (TransactionEmailLog) TableName() string { return "transaction_email_logs" }
