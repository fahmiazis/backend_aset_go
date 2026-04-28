package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

func CreateDisposal(userID string, req dto.CreateDisposalRequest) (*dto.DisposalResponse, error) {
	transactionNumber, err := GenerateTransactionNumber(userID, TxDisposal)
	if err != nil {
		return nil, err
	}

	transactionDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		return nil, errors.New("invalid transaction date format, use YYYY-MM-DD")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	transaction := models.Transaction{
		TransactionNumber: transactionNumber,
		TransactionType:   TxDisposal,
		TransactionDate:   transactionDate,
		Status:            models.TransactionStatusDraft,
		Notes:             req.Notes,
		CreatedBy:         userID, // FIX: langsung assign, userID sudah string (UUID)
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, item := range req.Items {
		var asset models.Asset
		if err := tx.First(&asset, item.AssetID).Error; err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("asset not found: %d", item.AssetID)
			}
			return nil, err
		}

		if asset.AssetNumber != item.AssetNumber {
			tx.Rollback()
			return nil, fmt.Errorf("asset number mismatch for asset ID: %d", item.AssetID)
		}

		if asset.AssetStatus == models.AssetStatusDisposed {
			tx.Rollback()
			return nil, fmt.Errorf("asset is already disposed: %s", item.AssetNumber)
		}

		var existingDisposal models.TransactionDisposal
		err := tx.Joins("JOIN transactions ON transactions.id = transaction_disposals.transaction_id").
			Where("transaction_disposals.asset_id = ? AND transactions.status = ?", item.AssetID, models.TransactionStatusDraft).
			First(&existingDisposal).Error

		if err == nil {
			tx.Rollback()
			return nil, fmt.Errorf("asset is already in a pending disposal: %s", item.AssetNumber)
		}

		disposal := models.TransactionDisposal{
			TransactionID:     transaction.ID,
			TransactionNumber: transactionNumber,
			AssetID:           asset.ID,
			AssetNumber:       item.AssetNumber,
			DisposalMethod:    item.DisposalMethod,
			DisposalValue:     item.DisposalValue,
			DisposalReason:    item.DisposalReason,
			Notes:             item.Notes,
		}

		if err := tx.Create(&disposal).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalByTransactionNumber(transactionNumber)
}

func GetDisposalByTransactionNumber(transactionNumber string) (*dto.DisposalResponse, error) {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxDisposal).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("disposal transaction not found")
		}
		return nil, err
	}

	var disposals []models.TransactionDisposal
	if err := config.DB.
		Preload("Asset").
		Where("transaction_id = ?", transaction.ID).
		Find(&disposals).Error; err != nil {
		return nil, err
	}

	return &dto.DisposalResponse{
		Transaction: mapTransactionHeaderToResponse(transaction),
		Items:       mapDisposalItemsToResponse(disposals),
	}, nil
}

func UpdateDisposal(transactionNumber string, userID string, req dto.CreateDisposalRequest) (*dto.DisposalResponse, error) {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxDisposal).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("disposal transaction not found")
		}
		return nil, err
	}

	if transaction.Status != models.TransactionStatusDraft {
		return nil, errors.New("can only update DRAFT transactions")
	}

	// FIX: langsung compare string, tidak perlu fmt.Sprintf
	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only update your own transactions")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("transaction_id = ?", transaction.ID).Delete(&models.TransactionDisposal{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	transactionDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		tx.Rollback()
		return nil, errors.New("invalid transaction date format, use YYYY-MM-DD")
	}

	if err := tx.Model(&transaction).Updates(map[string]interface{}{
		"transaction_date": transactionDate,
		"notes":            req.Notes,
	}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, item := range req.Items {
		var asset models.Asset
		if err := tx.First(&asset, item.AssetID).Error; err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("asset not found: %d", item.AssetID)
			}
			return nil, err
		}

		disposal := models.TransactionDisposal{
			TransactionID:     transaction.ID,
			TransactionNumber: transaction.TransactionNumber,
			AssetID:           asset.ID,
			AssetNumber:       item.AssetNumber,
			DisposalMethod:    item.DisposalMethod,
			DisposalValue:     item.DisposalValue,
			DisposalReason:    item.DisposalReason,
			Notes:             item.Notes,
		}

		if err := tx.Create(&disposal).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetDisposalByTransactionNumber(transaction.TransactionNumber)
}

func DeleteDisposal(transactionNumber string, userID string) error {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxDisposal).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("disposal transaction not found")
		}
		return err
	}

	if transaction.Status != models.TransactionStatusDraft {
		return errors.New("can only delete DRAFT transactions")
	}

	// FIX: langsung compare string
	if transaction.CreatedBy != userID {
		return errors.New("you can only delete your own transactions")
	}

	MarkTransactionAsExpired(transactionNumber)
	return config.DB.Delete(&transaction).Error
}
