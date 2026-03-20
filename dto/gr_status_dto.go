package dto

// ============================================================
// GR STATUS RESPONSE
// ============================================================

type AssetGRStatusResponse struct {
	TransactionNumber string           `json:"transaction_number"`
	CurrentStage      string           `json:"current_stage"`
	TotalAssets       int              `json:"total_assets"`
	GRDone            int              `json:"gr_done"`
	GRPending         int              `json:"gr_pending"`
	Items             []GRItemResponse `json:"items"`
}

type GRItemResponse struct {
	ProcurementItemID uint            `json:"procurement_item_id"`
	ItemName          string          `json:"item_name"`
	Quantity          int             `json:"quantity"`
	BranchCode        string          `json:"branch_code"`
	Assets            []AssetGRDetail `json:"assets"`
}

type AssetGRDetail struct {
	AssetID     uint    `json:"asset_id"`
	AssetNumber string  `json:"asset_number"`
	AssetName   string  `json:"asset_name"`
	BranchCode  string  `json:"branch_code"`
	IONumber    string  `json:"io_number"`
	GRStatus    string  `json:"gr_status"` // BELUM_GR atau AVAILABLE
	GRDate      *string `json:"gr_date"`   // terisi kalau sudah GR
	GRBy        *string `json:"gr_by"`     // terisi kalau sudah GR
}
