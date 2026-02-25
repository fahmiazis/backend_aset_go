package dto

import "time"

// ============================================================================
// USER BRANCH / HOMEBASE DTOs
// ============================================================================

type HomebaseBranchResponse struct {
	ID         string                `json:"id"`
	UserID     string                `json:"user_id"`
	BranchID   string                `json:"branch_id"`
	BranchType string                `json:"branch_type"`
	IsActive   bool                  `json:"is_active"`
	CreatedAt  time.Time             `json:"created_at"`
	Branch     *HomebaseBranchDetail `json:"branch,omitempty"`
}

type HomebaseBranchDetail struct {
	ID         string `json:"id"`
	BranchCode string `json:"branch_code"`
	BranchName string `json:"branch_name"`
	BranchType string `json:"branch_type"`
	Status     string `json:"status"`
}

type SetActiveHomebaseRequest struct {
	BranchID string `json:"branch_id" binding:"required"`
}

// ============================================================================
// TRANSACTION NUMBER DTOs
// ============================================================================

type GenerateTransactionNumberRequest struct {
	TransactionType string `json:"transaction_type" binding:"required,oneof=procurement disposal mutation stock_opname"`
}

type GenerateTransactionNumberResponse struct {
	TransactionNumber string `json:"transaction_number"`
	BranchCode        string `json:"branch_code"`
	BranchName        string `json:"branch_name"`
	TransactionType   string `json:"transaction_type"`
	Status            string `json:"status"`
}

type ReservoirResponse struct {
	ID          string    `json:"id"`
	NoTransaksi string    `json:"no_transaksi"`
	KodePlant   string    `json:"kode_plant"`
	Transaksi   string    `json:"transaksi"`
	Tipe        string    `json:"tipe"`
	Status      string    `json:"status"`
	BranchID    *string   `json:"branch_id"`
	UserID      *string   `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
}
