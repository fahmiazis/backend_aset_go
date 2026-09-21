package models

import (
	"time"

	"gorm.io/gorm"
)

// ============================================================
// Constants — Disposal Agreement
// ============================================================

const (
	StageAgreementApproval = "APPROVAL_AGREEMENT"
	StageAgreementFinished = "FINISHED"
	StageAgreementRejected = "REJECTED"
)

const (
	AgreementStatusPending  = "PENDING"
	AgreementStatusApproved = "APPROVED"
	AgreementStatusRejected = "REJECTED"
)

// DisposalAgreement — kesepakatan yang menggabungkan beberapa transaksi
// disposal yang sudah lolos APPROVAL_REQUEST, untuk disetujui sekaligus oleh
// manajemen puncak.
//
// Punya nomor sendiri (0001/IX/2026-DPSL-AGMNT) sehingga satu aset disposal
// berakhir dengan dua nomor: nomor transaksi request dan nomor agreement.
// Setelah agreement disetujui, tiap transaksi anggota lanjut ke stage
// berikutnya memakai nomor transaksinya sendiri.
type DisposalAgreement struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	AgreementNumber string         `gorm:"size:100;not null;uniqueIndex" json:"agreement_number"`
	CurrentStage    string         `gorm:"size:50;not null;default:APPROVAL_AGREEMENT;index" json:"current_stage"`
	Status          string         `gorm:"size:50;not null;default:PENDING" json:"status"`
	Notes           *string        `gorm:"type:text" json:"notes"`
	RejectionReason *string        `gorm:"type:text" json:"rejection_reason"`
	CreatedBy       string         `gorm:"size:36;not null;index" json:"created_by"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Items []DisposalAgreementItem `gorm:"foreignKey:AgreementID" json:"items,omitempty"`
}

func (DisposalAgreement) TableName() string { return "disposal_agreements" }

// DisposalAgreementItem — transaksi disposal yang menjadi anggota agreement.
type DisposalAgreementItem struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	AgreementID       uint      `gorm:"not null;index" json:"agreement_id"`
	TransactionID     uint      `gorm:"not null" json:"transaction_id"`
	TransactionNumber string    `gorm:"size:100;not null;index" json:"transaction_number"`
	CreatedAt         time.Time `json:"created_at"`

	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
}

func (DisposalAgreementItem) TableName() string { return "disposal_agreement_items" }
