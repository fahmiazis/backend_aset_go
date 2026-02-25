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

func CreateStockOpname(userID string, req dto.CreateStockOpnameRequest) (*dto.StockOpnameResponse, error) {
	transactionNumber, err := GenerateTransactionNumber(userID, TxStockOpname)
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
		TransactionType:   TxStockOpname,
		TransactionDate:   transactionDate,
		Status:            models.TransactionStatusDraft, // FIX: pakai models konstanta
		Notes:             req.Notes,
		CreatedBy:         userID,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, item := range req.Items {
		var asset models.Asset
		// FIX: item.AssetID sudah uint, langsung pass
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

		// FIX: AssetStatus di model adalah string, pakai asset.AssetStatus langsung
		assetStatus := asset.AssetStatus
		if item.AssetStatus != nil {
			assetStatus = *item.AssetStatus
		}

		stockOpname := models.TransactionStockOpname{
			TransactionID:     transaction.ID,
			TransactionNumber: transactionNumber,
			AssetID:           asset.ID,
			AssetNumber:       item.AssetNumber,
			PhysicalStatus:    item.PhysicalStatus,
			Condition:         item.Condition,
			AssetStatus:       assetStatus,
			Notes:             item.Notes,
		}

		if err := tx.Create(&stockOpname).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetStockOpnameByTransactionNumber(transactionNumber)
}

func GetStockOpnameByTransactionNumber(transactionNumber string) (*dto.StockOpnameResponse, error) {
	var transaction models.Transaction
	// FIX: hapus Preload("Creator") & Preload("Approver")
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxStockOpname).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("stock opname transaction not found")
		}
		return nil, err
	}

	var stockOpnames []models.TransactionStockOpname
	if err := config.DB.
		Preload("Asset").
		Where("transaction_id = ?", transaction.ID).
		Find(&stockOpnames).Error; err != nil {
		return nil, err
	}

	return &dto.StockOpnameResponse{
		Transaction: mapTransactionHeaderToResponse(transaction),
		Items:       mapStockOpnameItemsToResponse(stockOpnames),
	}, nil
}

func GetAllStockOpnames(filter dto.TransactionListFilter) ([]dto.StockOpnameResponse, int64, error) {
	query := config.DB.Model(&models.Transaction{}).Where("transaction_type = ?", TxStockOpname)

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

	var transactions []models.Transaction
	// FIX: hapus Preload("Creator") & Preload("Approver"), pindah Offset/Limit sebelum Find
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(filter.Limit).
		Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	responses := make([]dto.StockOpnameResponse, len(transactions))
	for i, t := range transactions {
		var stockOpnames []models.TransactionStockOpname
		config.DB.Preload("Asset").Where("transaction_id = ?", t.ID).Find(&stockOpnames)

		responses[i] = dto.StockOpnameResponse{
			Transaction: mapTransactionHeaderToResponse(t),
			Items:       mapStockOpnameItemsToResponse(stockOpnames),
		}
	}

	return responses, total, nil
}

func UpdateStockOpname(transactionNumber string, userID string, req dto.CreateStockOpnameRequest) (*dto.StockOpnameResponse, error) {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxStockOpname).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("stock opname transaction not found")
		}
		return nil, err
	}

	if transaction.Status != models.TransactionStatusDraft { // FIX: pakai models konstanta
		return nil, errors.New("can only update DRAFT transactions")
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only update your own transactions")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("transaction_id = ?", transaction.ID).Delete(&models.TransactionStockOpname{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// FIX: tambah error handling untuk time.Parse
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
		// FIX: item.AssetID sudah uint
		if err := tx.First(&asset, item.AssetID).Error; err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("asset not found: %d", item.AssetID)
			}
			return nil, err
		}

		assetStatus := asset.AssetStatus
		if item.AssetStatus != nil {
			assetStatus = *item.AssetStatus
		}

		stockOpname := models.TransactionStockOpname{
			TransactionID:     transaction.ID,
			TransactionNumber: transaction.TransactionNumber,
			AssetID:           asset.ID,
			AssetNumber:       item.AssetNumber,
			PhysicalStatus:    item.PhysicalStatus,
			Condition:         item.Condition,
			AssetStatus:       assetStatus,
			Notes:             item.Notes,
		}

		if err := tx.Create(&stockOpname).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetStockOpnameByTransactionNumber(transaction.TransactionNumber)
}

func DeleteStockOpname(transactionNumber string, userID string) error {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxStockOpname).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("stock opname transaction not found")
		}
		return err
	}

	if transaction.Status != models.TransactionStatusDraft { // FIX: pakai models konstanta
		return errors.New("can only delete DRAFT transactions")
	}

	if transaction.CreatedBy != userID {
		return errors.New("you can only delete your own transactions")
	}

	MarkTransactionAsExpired(transactionNumber)
	return config.DB.Delete(&transaction).Error
}
