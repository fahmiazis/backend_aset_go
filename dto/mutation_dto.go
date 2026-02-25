package dto

import "time"

type CreateMutationRequest struct {
	TransactionDate string                      `json:"transaction_date" binding:"required"`
	Notes           *string                     `json:"notes"`
	Items           []CreateMutationItemRequest `json:"items" binding:"required,min=1"`
}

type CreateMutationItemRequest struct {
	AssetID        uint    `json:"asset_id" binding:"required"`
	AssetNumber    string  `json:"asset_number" binding:"required"`
	FromBranchCode string  `json:"from_branch_code" binding:"required"`
	ToBranchCode   string  `json:"to_branch_code" binding:"required"`
	FromLocation   *string `json:"from_location"`
	ToLocation     *string `json:"to_location"`
	Notes          *string `json:"notes"`
}

type MutationResponse struct {
	Transaction TransactionHeaderResponse `json:"transaction"`
	Items       []MutationItemResponse    `json:"items"`
}

type MutationItemResponse struct {
	ID                uint      `json:"id"`
	TransactionID     uint      `json:"transaction_id"`
	TransactionNumber string    `json:"transaction_number"`
	AssetID           uint      `json:"asset_id"`
	AssetNumber       string    `json:"asset_number"`
	AssetName         *string   `json:"asset_name,omitempty"`
	FromBranchCode    string    `json:"from_branch_code"`
	ToBranchCode      string    `json:"to_branch_code"`
	FromLocation      *string   `json:"from_location"`
	ToLocation        *string   `json:"to_location"`
	Notes             *string   `json:"notes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
