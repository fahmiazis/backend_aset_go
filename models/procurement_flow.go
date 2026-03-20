package models

import "time"

// ============================================================
// Constants
// ============================================================

// Transaction Stages
const (
	StageDraft          = "DRAFT"
	StageVerifikasiAset = "VERIFIKASI_ASET"
	StageApproval       = "APPROVAL"
	StageProsesBudget   = "PROSES_BUDGET"
	StageEksekusiAset   = "EKSEKUSI_ASET"
	StageGR             = "GR"
	StageSelesai        = "SELESAI"
	StageRejected       = "REJECTED"
)

// Stage Actions
const (
	ActionSubmit        = "SUBMIT"
	ActionVerify        = "VERIFY"
	ActionApprove       = "APPROVE"
	ActionReject        = "REJECT"
	ActionProcessBudget = "PROCESS_BUDGET"
	ActionExecute       = "EXECUTE"
	ActionGR            = "GR"
	ActionRevise        = "REVISE"
)

// Item Verification Types
const (
	ItemTypeAsset    = "ASSET"
	ItemTypeNonAsset = "NON_ASSET"
)

// Document Sequence Types
const (
	SeqTypeIO    = "IO"
	SeqTypeAsset = "ASSET"
)

// Asset Status tambahan
const (
	AssetStatusPendingReceipt = "PENDING_RECEIPT"
	AssetStatusAvailable      = "AVAILABLE"
)

// ============================================================
// TransactionStage
// History setiap perpindahan stage per transaksi
// ============================================================
type TransactionStage struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	TransactionID     uint      `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber string    `gorm:"size:100;not null;index" json:"transaction_number"`
	FromStage         *string   `gorm:"size:50" json:"from_stage"` // NULL jika stage pertama
	ToStage           string    `gorm:"size:50;not null;index" json:"to_stage"`
	Action            string    `gorm:"size:50;not null" json:"action"`
	ActorID           string    `gorm:"size:100;not null;index" json:"actor_id"` // UUID user
	ActorName         *string   `gorm:"size:100" json:"actor_name"`
	Notes             *string   `gorm:"type:text" json:"notes"`
	Metadata          *string   `gorm:"type:json" json:"metadata"`
	CreatedAt         time.Time `json:"created_at"`

	// Relations
	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
}

func (TransactionStage) TableName() string { return "transaction_stages" }

// ============================================================
// TransactionItemVerification
// Marking ASSET / NON_ASSET per item procurement
// ============================================================
type TransactionItemVerification struct {
	ID                       uint      `gorm:"primaryKey" json:"id"`
	TransactionID            uint      `gorm:"not null;index" json:"transaction_id"`
	TransactionProcurementID uint      `gorm:"not null;uniqueIndex" json:"transaction_procurement_id"`
	ItemType                 string    `gorm:"type:enum('ASSET','NON_ASSET');not null" json:"item_type"`
	IsActive                 bool      `gorm:"not null;default:true" json:"is_active"` // false jika NON_ASSET dikeluarkan dari list
	VerifiedBy               string    `gorm:"size:100;not null" json:"verified_by"`   // UUID PIC Asset
	VerifiedAt               time.Time `gorm:"not null" json:"verified_at"`
	Notes                    *string   `gorm:"type:text" json:"notes"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`

	// Relations
	Transaction            *Transaction            `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	TransactionProcurement *TransactionProcurement `gorm:"foreignKey:TransactionProcurementID" json:"transaction_procurement,omitempty"`
}

func (TransactionItemVerification) TableName() string { return "transaction_item_verifications" }

// ============================================================
// DocumentNumberSequence
// Global sequence untuk nomor IO dan nomor Asset
// ============================================================
type DocumentNumberSequence struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	SequenceType  string    `gorm:"type:enum('IO','ASSET');not null;uniqueIndex:uq_sequence" json:"sequence_type"`
	ReferenceCode string    `gorm:"size:50;not null;uniqueIndex:uq_sequence" json:"reference_code"` // branch_code untuk IO, category_code untuk ASSET
	LastSequence  uint      `gorm:"not null;default:0" json:"last_sequence"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (DocumentNumberSequence) TableName() string { return "document_number_sequences" }

// ============================================================
// AssetGR (Good Receipt)
// Tracking GR per asset — dilakukan oleh user branch tujuan
// ============================================================
type AssetGR struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	TransactionID     uint      `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber string    `gorm:"size:100;not null;index" json:"transaction_number"`
	AssetID           uint      `gorm:"not null;uniqueIndex" json:"asset_id"` // unique: satu asset hanya bisa GR sekali
	AssetNumber       string    `gorm:"size:100;not null" json:"asset_number"`
	BranchCode        string    `gorm:"size:50;not null" json:"branch_code"` // branch tujuan
	GRDate            time.Time `gorm:"type:date;not null" json:"gr_date"`
	GRBy              string    `gorm:"size:100;not null" json:"gr_by"` // UUID user branch tujuan
	GRAt              time.Time `gorm:"not null" json:"gr_at"`
	Notes             *string   `gorm:"type:text" json:"notes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// Relations
	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	Asset       *Asset       `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (AssetGR) TableName() string { return "asset_gr" }
