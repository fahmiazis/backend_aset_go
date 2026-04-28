package dto

import "time"

type TransactionHeaderResponse struct {
	ID                uint       `json:"id"`
	TransactionNumber string     `json:"transaction_number"`
	TransactionType   string     `json:"transaction_type"`
	TransactionDate   time.Time  `json:"transaction_date"`
	CurrentStage      string     `json:"current_stage"`
	Status            string     `json:"status"`
	Notes             *string    `json:"notes"`
	CreatedBy         string     `json:"created_by"`
	ApprovedBy        *string    `json:"approved_by"`
	ApprovedAt        *time.Time `json:"approved_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type TransactionListFilter struct {
	TransactionType *string `form:"transaction_type" binding:"omitempty,oneof=PROCUREMENT MUTATION DISPOSAL STOCK_OPNAME"`
	Status          *string `form:"status"`
	CurrentStage    *string `form:"current_stage"` // ADD: filter by stage
	CreatedBy       *string `form:"created_by"`    // ADD: filter by creator (UUID)
	StartDate       *string `form:"start_date"`
	EndDate         *string `form:"end_date"`
	Page            int     `form:"page" binding:"min=1"`
	Limit           int     `form:"limit" binding:"min=1,max=100"`
}
