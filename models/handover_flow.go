package models

import "time"

// ============================================================
// Serah Terima Aset (handover)
//
// Menyerahkan aset di cabang homebase pengaju ke seorang user (HANDOVER), atau
// mengembalikan aset yang sedang dipegang user ke cabang (RETURN). Hanya
// dalam satu cabang — aset lintas cabang wajib dimutasi dulu.
// ============================================================

const (
	StageHandoverDraft     = "DRAFT"
	StageHandoverApproval  = "APPROVAL"
	StageHandoverReceiving = "HANDOVER_RECEIVING" // penerima konfirmasi + upload BAST
	StageHandoverFinished  = "FINISHED"
	StageHandoverRejected  = "REJECTED"
	StageHandoverCancelled = "CANCELLED"
)

const (
	HandoverTypeHandover = "HANDOVER" // cabang/pemegang lama → user penerima
	HandoverTypeReturn   = "RETURN"   // user → cabang (pemegang dikosongkan)
)

const (
	HandoverAssetStatusPending   = "PENDING"
	HandoverAssetStatusCompleted = "COMPLETED"
	HandoverAssetStatusCancelled = "CANCELLED"
)

// Status aset selama ikut ajuan serah terima aktif
const AssetStatusInHandover = "IN_HANDOVER"

const FlowAssetHandoverApproval = "ASSET_HANDOVER_APPROVAL"

type TransactionHandoverAsset struct {
	ID                uint   `gorm:"primaryKey" json:"id"`
	TransactionID     uint   `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber string `gorm:"size:100;not null;index" json:"transaction_number"`
	AssetID           uint   `gorm:"not null;index" json:"asset_id"`
	AssetNumber       string `gorm:"size:100;not null" json:"asset_number"`
	// pemegang sebelum serah terima (NULL = dipegang cabang)
	FromUserID *string `gorm:"type:char(36)" json:"from_user_id"`
	// status aset sebelum dikunci IN_HANDOVER — dipulihkan saat selesai/batal
	PreviousAssetStatus string  `gorm:"size:50;not null" json:"previous_asset_status"`
	Notes               *string `gorm:"type:text" json:"notes"`
	Status              string  `gorm:"type:enum('PENDING','COMPLETED','CANCELLED');not null;default:PENDING;index" json:"status"`
	NeedsRevision       bool    `gorm:"not null;default:false" json:"needs_revision"`
	RevisionNotes       *string `gorm:"type:text" json:"revision_notes"`
	CreatedAt           time.Time
	UpdatedAt           time.Time

	Asset *Asset `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (TransactionHandoverAsset) TableName() string { return "transaction_handover_assets" }
