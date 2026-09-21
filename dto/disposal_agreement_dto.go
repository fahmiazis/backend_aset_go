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
	Items           []DisposalAgreementItemResponse `json:"items,omitempty"`
	CreatedAt       time.Time                       `json:"created_at"`
	UpdatedAt       time.Time                       `json:"updated_at"`
}

type DisposalAgreementListFilter struct {
	Stage  *string `form:"stage"`
	Search *string `form:"search"`
	Page   int     `form:"page"`
	Limit  int     `form:"limit"`
}
