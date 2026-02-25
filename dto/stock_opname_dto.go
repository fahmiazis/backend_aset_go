package dto

import "time"

type CreateStockOpnameRequest struct {
	TransactionDate string                         `json:"transaction_date" binding:"required"`
	Notes           *string                        `json:"notes"`
	Items           []CreateStockOpnameItemRequest `json:"items" binding:"required,min=1"`
}

type CreateStockOpnameItemRequest struct {
	AssetID        uint    `json:"asset_id" binding:"required"`
	AssetNumber    string  `json:"asset_number" binding:"required"`
	PhysicalStatus string  `json:"physical_status" binding:"required,oneof=EXISTS MISSING DAMAGED OBSOLETE"`
	Condition      string  `json:"condition" binding:"required,oneof=GOOD FAIR POOR BROKEN"`
	AssetStatus    *string `json:"asset_status" binding:"omitempty,oneof=ACTIVE INACTIVE MAINTENANCE RETIRED DISPOSED"`
	Notes          *string `json:"notes"`
}

type StockOpnameResponse struct {
	Transaction TransactionHeaderResponse `json:"transaction"`
	Items       []StockOpnameItemResponse `json:"items"`
}

type StockOpnameItemResponse struct {
	ID                uint      `json:"id"`
	TransactionID     uint      `json:"transaction_id"`
	TransactionNumber string    `json:"transaction_number"`
	AssetID           uint      `json:"asset_id"`
	AssetNumber       string    `json:"asset_number"`
	AssetName         *string   `json:"asset_name,omitempty"`
	PhysicalStatus    string    `json:"physical_status"`
	Condition         string    `json:"condition"`
	AssetStatus       *string   `json:"asset_status"`
	Notes             *string   `json:"notes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
