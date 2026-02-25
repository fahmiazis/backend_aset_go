package dto

import "time"

type CreateProcurementRequest struct {
	TransactionDate string                         `json:"transaction_date" binding:"required"`
	Notes           *string                        `json:"notes"`
	Items           []CreateProcurementItemRequest `json:"items" binding:"required,min=1"`
}

type CreateProcurementItemRequest struct {
	ItemName   string                           `json:"item_name" binding:"required"`
	CategoryID uint                             `json:"category_id" binding:"required"`
	Quantity   int                              `json:"quantity" binding:"required,min=1"`
	UnitPrice  float64                          `json:"unit_price" binding:"required,min=0"`
	BranchCode *string                          `json:"branch_code"`
	Notes      *string                          `json:"notes"`
	Details    []CreateProcurementDetailRequest `json:"details"`
}

type CreateProcurementDetailRequest struct {
	BranchCode    string  `json:"branch_code" binding:"required"`
	Quantity      int     `json:"quantity" binding:"required,min=1"`
	RequesterName *string `json:"requester_name"`
	Notes         *string `json:"notes"`
}

type ProcurementResponse struct {
	Transaction TransactionHeaderResponse `json:"transaction"`
	Items       []ProcurementItemResponse `json:"items"`
}

type ProcurementItemResponse struct {
	ID                uint                        `json:"id"`
	TransactionID     uint                        `json:"transaction_id"`
	TransactionNumber string                      `json:"transaction_number"`
	ItemName          string                      `json:"item_name"`
	CategoryID        *uint                       `json:"category_id"`
	CategoryName      *string                     `json:"category_name,omitempty"`
	Quantity          int                         `json:"quantity"`
	UnitPrice         float64                     `json:"unit_price"`
	TotalPrice        float64                     `json:"total_price"`
	BranchCode        string                      `json:"branch_code"`
	Notes             *string                     `json:"notes"`
	CreatedAt         time.Time                   `json:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at"`
	Details           []ProcurementDetailResponse `json:"details,omitempty"`
}

type ProcurementDetailResponse struct {
	ID                       uint      `json:"id"`
	TransactionProcurementID uint      `json:"transaction_procurement_id"`
	BranchCode               string    `json:"branch_code"`
	Quantity                 int       `json:"quantity"`
	RequesterName            *string   `json:"requester_name"`
	Notes                    *string   `json:"notes"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}
