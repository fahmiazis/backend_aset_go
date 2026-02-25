package dto

import "time"

type AssetHistoryResponse struct {
	ID              uint       `json:"id"`
	AssetID         uint       `json:"asset_id"`
	AssetNumber     *string    `json:"asset_number,omitempty"`
	TransactionType string     `json:"transaction_type"`
	TransactionID   *uint      `json:"transaction_id"` // FIX: *string -> *uint
	DocumentNumber  *string    `json:"document_number"`
	TransactionDate *time.Time `json:"transaction_date"`
	BeforeData      *string    `json:"before_data"`
	AfterData       *string    `json:"after_data"`
	ChangedBy       *string    `json:"changed_by"`
	CreatedAt       time.Time  `json:"created_at"`
}

type AssetHistoryFilter struct {
	AssetID         *uint   `form:"asset_id"`
	TransactionType *string `form:"transaction_type"`
	StartDate       *string `form:"start_date"`
	EndDate         *string `form:"end_date"`
	Page            int     `form:"page" binding:"min=1"`
	Limit           int     `form:"limit" binding:"min=1,max=100"`
}
