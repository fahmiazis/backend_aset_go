package dto

import "time"

// ============================================================
// TRANSACTION STAGE DTOs
// ============================================================

type TransactionStageResponse struct {
	ID                uint      `json:"id"`
	TransactionID     uint      `json:"transaction_id"`
	TransactionNumber string    `json:"transaction_number"`
	FromStage         *string   `json:"from_stage"`
	ToStage           string    `json:"to_stage"`
	Action            string    `json:"action"`
	ActorID           string    `json:"actor_id"`
	ActorName         *string   `json:"actor_name"`
	Notes             *string   `json:"notes"`
	CreatedAt         time.Time `json:"created_at"`
}

// ============================================================
// SUBMIT PROCUREMENT
// DRAFT → VERIFIKASI_ASET
// ============================================================

type SubmitProcurementRequest struct {
	Notes *string `json:"notes"`
}

// ============================================================
// VERIFIKASI ASET
// VERIFIKASI_ASET → APPROVAL
// PIC Asset marking setiap item: ASSET atau NON_ASSET
// ============================================================

type VerifyProcurementItemRequest struct {
	TransactionProcurementID uint    `json:"transaction_procurement_id" binding:"required"`
	ItemType                 string  `json:"item_type" binding:"required,oneof=ASSET NON_ASSET"`
	Notes                    *string `json:"notes"`
}

type VerifyProcurementRequest struct {
	Items []VerifyProcurementItemRequest `json:"items" binding:"required,min=1"`
	Notes *string                        `json:"notes"`
}

type TransactionItemVerificationResponse struct {
	ID                       uint      `json:"id"`
	TransactionID            uint      `json:"transaction_id"`
	TransactionProcurementID uint      `json:"transaction_procurement_id"`
	ItemName                 *string   `json:"item_name,omitempty"`
	ItemType                 string    `json:"item_type"`
	IsActive                 bool      `json:"is_active"`
	VerifiedBy               string    `json:"verified_by"`
	VerifiedAt               time.Time `json:"verified_at"`
	Notes                    *string   `json:"notes"`
}

// ============================================================
// APPROVAL
// APPROVAL → PROSES_BUDGET (jika approved) atau REJECTED
// Pakai system TransactionApproval yang sudah ada
// ============================================================

type InitiateApprovalRequest struct {
	FlowID   string  `json:"flow_id" binding:"required"`
	Notes    *string `json:"notes"`
	Metadata *string `json:"metadata"`
}

// ============================================================
// PROSES BUDGET
// PROSES_BUDGET → EKSEKUSI_ASET
// PIC Budget input/generate nomor IO
// ============================================================

type ProcessBudgetRequest struct {
	Notes *string `json:"notes"`
}

type ProcessBudgetResponse struct {
	TransactionNumber string    `json:"transaction_number"`
	IONumber          string    `json:"io_number"`
	GeneratedAt       time.Time `json:"generated_at"`
	GeneratedBy       string    `json:"generated_by"`
}

// ============================================================
// EKSEKUSI ASET
// EKSEKUSI_ASET → GR
// PIC Asset generate nomor asset & create asset records
// ============================================================

type ExecuteAssetRequest struct {
	Notes *string `json:"notes"`
}

type ExecuteAssetItemResponse struct {
	TransactionProcurementID uint      `json:"transaction_procurement_id"`
	ItemName                 string    `json:"item_name"`
	AssetID                  uint      `json:"asset_id"`
	AssetNumber              string    `json:"asset_number"`
	CategoryCode             string    `json:"category_code"`
	GeneratedAt              time.Time `json:"generated_at"`
}

type ExecuteAssetResponse struct {
	TransactionNumber string                     `json:"transaction_number"`
	Assets            []ExecuteAssetItemResponse `json:"assets"`
	ExecutedBy        string                     `json:"executed_by"`
	ExecutedAt        time.Time                  `json:"executed_at"`
}

// ============================================================
// GOOD RECEIPT (GR)
// GR per item — dilakukan user branch tujuan
// ============================================================

type CreateGRRequest struct {
	AssetID     uint    `json:"asset_id" binding:"required"`
	AssetNumber string  `json:"asset_number" binding:"required"`
	GRDate      string  `json:"gr_date" binding:"required"` // format YYYY-MM-DD
	Notes       *string `json:"notes"`
}

type AssetGRResponse struct {
	ID                uint      `json:"id"`
	TransactionID     uint      `json:"transaction_id"`
	TransactionNumber string    `json:"transaction_number"`
	AssetID           uint      `json:"asset_id"`
	AssetNumber       string    `json:"asset_number"`
	BranchCode        string    `json:"branch_code"`
	GRDate            time.Time `json:"gr_date"`
	GRBy              string    `json:"gr_by"`
	GRAt              time.Time `json:"gr_at"`
	Notes             *string   `json:"notes"`
	CreatedAt         time.Time `json:"created_at"`
}

// ============================================================
// REJECT
// Bisa dilakukan di semua stage kecuali DRAFT & SELESAI
// ============================================================

type RejectProcurementRequest struct {
	Reason string `json:"reason" binding:"required,min=10"`
}

// ============================================================
// REVISI
// Bisa dilakukan di semua stage kecuali DRAFT, SELESAI, REJECTED
// Kalau stage = APPROVAL → ulang dari APPROVAL
// Kalau stage lain → langsung ke stage tersebut
// ============================================================

type ReviseProcurementRequest struct {
	TransactionDate *string                        `json:"transaction_date"`
	Notes           *string                        `json:"notes"`
	Items           []CreateProcurementItemRequest `json:"items" binding:"required,min=1"`
	RevisionNotes   string                         `json:"revision_notes" binding:"required,min=5"`
}

// ============================================================
// PROCUREMENT DETAIL RESPONSE (dengan stage info)
// ============================================================

type ProcurementDetailWithStageResponse struct {
	Transaction ProcurementTransactionResponse            `json:"transaction"`
	Items       []ProcurementItemWithVerificationResponse `json:"items"`
	Stages      []TransactionStageResponse                `json:"stages"`
	GRStatus    []AssetGRResponse                         `json:"gr_status,omitempty"`
}

type ProcurementTransactionResponse struct {
	ID                uint       `json:"id"`
	TransactionNumber string     `json:"transaction_number"`
	TransactionType   string     `json:"transaction_type"`
	TransactionDate   time.Time  `json:"transaction_date"`
	Status            string     `json:"status"`
	CurrentStage      string     `json:"current_stage"`
	IONumber          *string    `json:"io_number"`
	Notes             *string    `json:"notes"`
	CreatedBy         string     `json:"created_by"`
	ApprovedBy        *string    `json:"approved_by"`
	ApprovedAt        *time.Time `json:"approved_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type ProcurementItemWithVerificationResponse struct {
	ProcurementItemResponse
	Verification *TransactionItemVerificationResponse `json:"verification,omitempty"`
	Asset        *AssetBriefResponse                  `json:"asset,omitempty"` // terisi setelah EKSEKUSI_ASET
}

type AssetBriefResponse struct {
	ID          uint    `json:"id"`
	AssetNumber string  `json:"asset_number"`
	AssetName   string  `json:"asset_name"`
	AssetStatus string  `json:"asset_status"`
	GRStatus    *string `json:"gr_status"` // "BELUM_GR" atau "AVAILABLE"
}

// ============================================================
// DOCUMENT NUMBER SEQUENCE RESPONSE
// ============================================================

type GeneratedNumberResponse struct {
	SequenceType  string    `json:"sequence_type"`
	ReferenceCode string    `json:"reference_code"`
	Number        string    `json:"number"`
	GeneratedAt   time.Time `json:"generated_at"`
}
