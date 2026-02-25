package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"

	"gorm.io/gorm"
)

// Transaction status constants
const (
	TransactionStatusDraft    = "DRAFT"
	TransactionStatusApproved = "APPROVED"
	TransactionStatusRejected = "REJECTED"
)

// GetTransactionByNumber gets transaction by number
func GetTransactionByNumber(transactionNumber string) (*dto.TransactionHeaderResponse, error) {
	var transaction models.Transaction

	if err := config.DB.
		Preload("Creator").
		Preload("Approver").
		Where("transaction_number = ?", transactionNumber).
		First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("transaction not found")
		}
		return nil, err
	}

	response := mapTransactionHeaderToResponse(transaction)
	return &response, nil
}

// GetAllTransactions gets all transactions with filters
func GetAllTransactions(filter dto.TransactionListFilter) ([]dto.TransactionHeaderResponse, int64, error) {
	query := config.DB.Model(&models.Transaction{})

	// Apply filters
	if filter.TransactionType != nil {
		query = query.Where("transaction_type = ?", *filter.TransactionType)
	}

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	if filter.StartDate != nil {
		query = query.Where("transaction_date >= ?", *filter.StartDate)
	}

	if filter.EndDate != nil {
		query = query.Where("transaction_date <= ?", *filter.EndDate)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.Limit
	query = query.Offset(offset).Limit(filter.Limit)

	// Get transactions
	var transactions []models.Transaction
	if err := query.
		Preload("Creator").
		Preload("Approver").
		Order("created_at DESC").
		Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return mapTransactionHeadersToResponse(transactions), total, nil
}

// GetTransactionsByUser gets transactions created by specific user
func GetTransactionsByUser(userID string, filter dto.TransactionListFilter) ([]dto.TransactionHeaderResponse, int64, error) {
	query := config.DB.Model(&models.Transaction{}).
		Where("created_by = ?", userID)

	// Apply filters (same as GetAllTransactions)
	if filter.TransactionType != nil {
		query = query.Where("transaction_type = ?", *filter.TransactionType)
	}

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	if filter.StartDate != nil {
		query = query.Where("transaction_date >= ?", *filter.StartDate)
	}

	if filter.EndDate != nil {
		query = query.Where("transaction_date <= ?", *filter.EndDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.Limit
	query = query.Offset(offset).Limit(filter.Limit)

	var transactions []models.Transaction
	if err := query.
		Preload("Creator").
		Preload("Approver").
		Order("created_at DESC").
		Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return mapTransactionHeadersToResponse(transactions), total, nil
}
