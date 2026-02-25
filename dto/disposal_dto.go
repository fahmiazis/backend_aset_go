package dto

import "time"

type CreateDisposalRequest struct {
	TransactionDate string                      `json:"transaction_date" binding:"required"`
	Notes           *string                     `json:"notes"`
	Items           []CreateDisposalItemRequest `json:"items" binding:"required,min=1"`
}

type CreateDisposalItemRequest struct {
	AssetID        uint    `json:"asset_id" binding:"required"`
	AssetNumber    string  `json:"asset_number" binding:"required"`
	DisposalMethod string  `json:"disposal_method" binding:"required,oneof=SALE SCRAP DONATE WRITE_OFF"`
	DisposalValue  float64 `json:"disposal_value" binding:"min=0"`
	DisposalReason *string `json:"disposal_reason"`
	Notes          *string `json:"notes"`
}

type DisposalResponse struct {
	Transaction TransactionHeaderResponse `json:"transaction"`
	Items       []DisposalItemResponse    `json:"items"`
}

type DisposalItemResponse struct {
	ID                uint      `json:"id"`
	TransactionID     uint      `json:"transaction_id"`
	TransactionNumber string    `json:"transaction_number"`
	AssetID           uint      `json:"asset_id"`
	AssetNumber       string    `json:"asset_number"`
	AssetName         *string   `json:"asset_name,omitempty"`
	DisposalMethod    string    `json:"disposal_method"`
	DisposalValue     float64   `json:"disposal_value"`
	DisposalReason    *string   `json:"disposal_reason"`
	Notes             *string   `json:"notes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
