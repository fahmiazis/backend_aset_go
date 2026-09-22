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
// PhysicalStatus & Condition SEKARANG master data (lihat
// models.StockOpnamePhysicalStatusMaster / StockOpnameConditionMaster,
// bisa ditambah lewat /transactions/stock-opname/status-master/*) — jadi
// TIDAK ada lagi binding:"oneof=..." statis di sini, validitas kode +
// aturan silangnya (mis. status yang RequiresNotApplicableCondition wajib
// pasangan condition yang IsNotApplicableValue) dicek di service terhadap
// data master saat itu. Khusus status yang RequiresBorrowDocument (dulu
// cuma "BORROWED"), dokumen peminjaman (PDF) wajib sudah diupload lebih
// dulu lewat endpoint upload terpisah — juga divalidasi di service.
// ============================================================

type UpdateStockOpnameFindingRequest struct {
	AssetID        uint    `json:"asset_id" binding:"required"`
	PhysicalStatus string  `json:"physical_status" binding:"required"`
	Condition      string  `json:"condition" binding:"required"`
	AssetStatus    *string `json:"asset_status" binding:"omitempty,oneof=ACTIVE INACTIVE MAINTENANCE RETIRED"`
	Notes          *string `json:"notes"`
}

// ============================================================
// BULK UPDATE TEMUAN (autosave dari grid "Lengkapi Data")
//
// Beda dari UpdateStockOpnameFindingRequest: field di sini SEMUA opsional
// (pointer) karena tiap cell di grid bisa disimpan sendiri-sendiri sambil
// user masih ngisi cell lain — field yang gak dikirim (nil) berarti "belum
// berubah", nilai lama di DB dipertahankan. Validasi pasangan
// physical_status/condition dilakukan di service terhadap nilai EFEKTIF
// (gabungan nilai baru + nilai lama yang belum diubah).
//
// SENGAJA gak pakai binding:"oneof=..." di sini (beda dari
// UpdateStockOpnameFindingRequest) — pointer non-nil ke string KOSONG
// harus tetap valid lolos binding, karena itu representasi "field ini
// sengaja dikosongkan lagi" (mis. saat physical_status dibalik dari
// MISSING ke EXISTS, condition ikut direset ke "" di FE supaya user pilih
// ulang). Validasi non-empty-tapi-invalid ditangani manual di service.
// ============================================================

type BulkUpdateStockOpnameFindingItem struct {
	AssetID        uint    `json:"asset_id" binding:"required"`
	PhysicalStatus *string `json:"physical_status"`
	Condition      *string `json:"condition"`
	AssetStatus    *string `json:"asset_status"`
	Notes          *string `json:"notes"`
}

type BulkUpdateStockOpnameFindingRequest struct {
	Items []BulkUpdateStockOpnameFindingItem `json:"items" binding:"required,min=1,dive"`
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

	// Foto bukti fisik — wajib diisi sebelum submit (lihat SubmitStockOpname)
	PhotoID         *uint      `json:"photo_id,omitempty"`
	PhotoURL        *string    `json:"photo_url,omitempty"`
	PhotoCapturedAt *time.Time `json:"photo_captured_at,omitempty"`

	// Dokumen peminjaman (PDF) — wajib diisi kalau FoundPhysicalStatus =
	// BORROWED, divalidasi juga sebelum submit (lihat SubmitStockOpname)
	BorrowDocumentID       *uint   `json:"borrow_document_id,omitempty"`
	BorrowDocumentURL      *string `json:"borrow_document_url,omitempty"`
	BorrowDocumentFileName *string `json:"borrow_document_file_name,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StockOpnameFlowDetailResponse struct {
	Transaction TransactionHeaderResponse     `json:"transaction"`
	Items       []StockOpnameFlowItemResponse `json:"items"`
	Stages      []TransactionStageResponse    `json:"stages"`

	// IsSubmissive: null kalau belum pernah di-submit, kalau udah diisi
	// TRUE/FALSE tergantung submit-nya masuk jendela StockOpnameConfig atau
	// enggak (lihat SubmitStockOpname). Cuma penanda kepatuhan jadwal —
	// tidak pernah memblokir submit.
	IsSubmissive *bool `json:"is_submissive"`
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
