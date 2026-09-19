package dto

import "time"

// ============================================================
// CREATE DRAFT
//
// TransactionDate SENGAJA gak ada di sini — tanggal opname selalu
// otomatis diisi tanggal hari ini (server time) di service layer.
// ============================================================

type CreateStockOpnameDraftRequest struct {
	Notes *string `json:"notes"`
}

// ============================================================
// INPUT HASIL TEMUAN FISIK PER ASSET (masih draft)
//
// PhysicalStatus hanya 2 opsi: EXISTS ("Ada") / MISSING ("Tidak Ada").
// Condition punya opsi tambahan NOT_APPLICABLE ("Tidak Ada") yang WAJIB
// dipakai ketika PhysicalStatus = MISSING (dan hanya boleh dipakai saat
// itu) — divalidasi silang di service, bukan cuma lewat oneof di sini.
// ============================================================

type UpdateStockOpnameFindingRequest struct {
	AssetID        uint    `json:"asset_id" binding:"required"`
	PhysicalStatus string  `json:"physical_status" binding:"required,oneof=EXISTS MISSING"`
	Condition      string  `json:"condition" binding:"required,oneof=GOOD FAIR POOR BROKEN NOT_APPLICABLE"`
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

// ============================================================
// UPLOAD TEMPLATE EXCEL (bulk update temuan)
// ============================================================

type StockOpnameTemplateRowError struct {
	Row         int    `json:"row"`
	AssetNumber string `json:"asset_number"`
	Message     string `json:"message"`
}

type StockOpnameTemplateUploadResponse struct {
	UpdatedCount int                            `json:"updated_count"`
	FailedCount  int                            `json:"failed_count"`
	Errors       []StockOpnameTemplateRowError  `json:"errors"`
	Detail       *StockOpnameFlowDetailResponse `json:"detail"`
}
