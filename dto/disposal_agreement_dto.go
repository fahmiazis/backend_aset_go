package dto

import "time"

// ============================================================
// DISPOSAL AGREEMENT
// ============================================================

type CreateDisposalAgreementRequest struct {
	// nomor transaksi disposal yang digabungkan, minimal satu
	TransactionNumbers []string `json:"transaction_numbers" binding:"required,min=1"`
	Notes              *string  `json:"notes"`
}

type DisposalAgreementItemResponse struct {
	TransactionID     uint     `json:"transaction_id"`
	TransactionNumber string   `json:"transaction_number"`
	DisposalType      *string  `json:"disposal_type"`
	CurrentStage      string   `json:"current_stage"`
	BranchCode        string   `json:"branch_code"`
	CreatedBy         string   `json:"created_by"`
	CreatedByName     *string  `json:"created_by_name,omitempty"`
	TotalAssets       int      `json:"total_assets"`
	TotalSaleValue    *float64 `json:"total_sale_value,omitempty"`
}

// DisposalAgreementAssetResponse — aset dari seluruh transaksi anggota,
// diratakan jadi satu daftar. Yang ditimbang manajemen saat menyetujui
// kesepakatan adalah asetnya, bukan nomor transaksinya.
type DisposalAgreementAssetResponse struct {
	DisposalAssetID   uint     `json:"disposal_asset_id"`
	AssetID           uint     `json:"asset_id"`
	AssetNumber       string   `json:"asset_number"`
	AssetName         *string  `json:"asset_name,omitempty"`
	CategoryName      *string  `json:"category_name,omitempty"`
	BranchCode        *string  `json:"branch_code,omitempty"`
	DisposalType      string   `json:"disposal_type"`
	DisposalReason    *string  `json:"disposal_reason"`
	SaleValue         *float64 `json:"sale_value"`
	TransactionNumber string   `json:"transaction_number"`
}

type DisposalAgreementResponse struct {
	ID              uint                            `json:"id"`
	AgreementNumber string                          `json:"agreement_number"`
	CurrentStage    string                          `json:"current_stage"`
	Status          string                          `json:"status"`
	Notes           *string                         `json:"notes"`
	RejectionReason *string                         `json:"rejection_reason"`
	CreatedBy       string                          `json:"created_by"`
	CreatedByName   *string                         `json:"created_by_name,omitempty"`
	TotalItems      int                             `json:"total_items"`
	TotalAssets     int                             `json:"total_assets"`
	Items           []DisposalAgreementItemResponse `json:"items,omitempty"`
	Assets          []DisposalAgreementAssetResponse `json:"assets,omitempty"`
	CreatedAt       time.Time                       `json:"created_at"`
	UpdatedAt       time.Time                       `json:"updated_at"`
}

type DisposalAgreementListFilter struct {
	Stage  *string `form:"stage"`
	Search *string `form:"search"`
	// rentang tanggal pembuatan agreement, format YYYY-MM-DD
	StartDate *string `form:"start_date"`
	EndDate   *string `form:"end_date"`
	Page      int     `form:"page"`
	Limit     int     `form:"limit"`
}
