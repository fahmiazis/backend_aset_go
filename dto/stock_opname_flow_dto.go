package dto

import "time"

// ============================================================
// CREATE DRAFT
// ============================================================

type CreateStockOpnameDraftRequest struct {
	TransactionDate string  `json:"transaction_date" binding:"required"`
	Notes           *string `json:"notes"`
}

// ============================================================
// ADD ASSET KE DRAFT
// ============================================================

type AddStockOpnameAssetRequest struct {
	AssetID     uint   `json:"asset_id" binding:"required"`
	AssetNumber string `json:"asset_number" binding:"required"`
}

type RemoveStockOpnameAssetRequest struct {
	AssetID uint `json:"asset_id" binding:"required"`
}

// ============================================================
// INPUT HASIL TEMUAN FISIK PER ASSET (masih draft)
// ============================================================

type UpdateStockOpnameFindingRequest struct {
	AssetID        uint    `json:"asset_id" binding:"required"`
	PhysicalStatus string  `json:"physical_status" binding:"required,oneof=EXISTS MISSING DAMAGED OBSOLETE"`
	Condition      string  `json:"condition" binding:"required,oneof=GOOD FAIR POOR BROKEN"`
	AssetStatus    *string `json:"asset_status" binding:"omitempty,oneof=ACTIVE INACTIVE MAINTENANCE RETIRED"`
	Notes          *string `json:"notes"`
}

// ============================================================
// SUBMIT
// ============================================================

type SubmitStockOpnameRequest struct {
	Notes *string `json:"notes"`
}

// ============================================================
// EKSEKUSI
// ============================================================

type ExecuteStockOpnameRequest struct {
	Notes *string `json:"notes"`
}

// ============================================================
// REJECT
// ============================================================

type RejectStockOpnameRequest struct {
	Reason string `json:"reason" binding:"required,min=10"`
}

// ============================================================
// RESPONSES
// ============================================================

// StockOpnameFlowItemResponse menampilkan temuan auditor berdampingan
// dengan data sistem (aktif) saat ini — inti dari stock opname.
type StockOpnameFlowItemResponse struct {
	ID                uint    `json:"id"`
	TransactionID     uint    `json:"transaction_id"`
	TransactionNumber string  `json:"transaction_number"`
	AssetID           uint    `json:"asset_id"`
	AssetNumber       string  `json:"asset_number"`
	AssetName         *string `json:"asset_name,omitempty"`
	CategoryName      *string `json:"category_name,omitempty"`
	BranchCode        *string `json:"branch_code,omitempty"`

	// Hasil temuan fisik (diisi/diupdate selama DRAFT)
	FoundPhysicalStatus *string `json:"found_physical_status"`
	FoundCondition      *string `json:"found_condition"`
	FoundAssetStatus    *string `json:"found_asset_status"`
	Notes               *string `json:"notes"`

	// Data sistem saat ini (live, untuk perbandingan)
	SystemPhysicalStatus *string `json:"system_physical_status,omitempty"`
	SystemCondition      *string `json:"system_condition,omitempty"`
	SystemAssetStatus    string  `json:"system_asset_status,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StockOpnameFlowDetailResponse struct {
	Transaction TransactionHeaderResponse     `json:"transaction"`
	Items       []StockOpnameFlowItemResponse `json:"items"`
	Stages      []TransactionStageResponse    `json:"stages"`
}
