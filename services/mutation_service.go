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

func CreateMutation(userID string, req dto.CreateMutationRequest) (*dto.MutationResponse, error) {
	transactionNumber, err := GenerateTransactionNumber(userID, TxMutation)
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
		TransactionType:   TxMutation,
		TransactionDate:   transactionDate,
		Status:            models.TransactionStatusDraft,
		Notes:             req.Notes,
		CreatedBy:         userID,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, item := range req.Items {
		var asset models.Asset
		// FIX: item.AssetID sudah uint, langsung pass tanpa "id = ?"
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

		var existingMutation models.TransactionMutation
		err := tx.Joins("JOIN transactions ON transactions.id = transaction_mutations.transaction_id").
			Where("transaction_mutations.asset_id = ? AND transactions.status = ?", item.AssetID, models.TransactionStatusDraft).
			First(&existingMutation).Error

		if err == nil {
			tx.Rollback()
			return nil, fmt.Errorf("asset is already in a pending mutation: %s", item.AssetNumber)
		}

		if asset.BranchCode == nil || *asset.BranchCode != item.FromBranchCode {
			tx.Rollback()
			return nil, fmt.Errorf("asset current branch does not match from_branch_code: %s", item.AssetNumber)
		}

		if item.FromBranchCode == item.ToBranchCode {
			tx.Rollback()
			return nil, fmt.Errorf("from_branch and to_branch must be different: %s", item.AssetNumber)
		}

		mutation := models.TransactionMutation{
			TransactionID:     transaction.ID,
			TransactionNumber: transactionNumber,
			AssetID:           asset.ID,
			AssetNumber:       item.AssetNumber,
			FromBranchCode:    item.FromBranchCode,
			ToBranchCode:      item.ToBranchCode,
			FromLocation:      item.FromLocation,
			ToLocation:        item.ToLocation,
			Notes:             item.Notes,
		}

		if err := tx.Create(&mutation).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetMutationByTransactionNumber(transactionNumber)
}

func GetMutationByTransactionNumber(transactionNumber string) (*dto.MutationResponse, error) {
	var transaction models.Transaction
	// FIX: hapus Preload("Creator") & Preload("Approver")
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxMutation).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("mutation transaction not found")
		}
		return nil, err
	}

	var mutations []models.TransactionMutation
	if err := config.DB.
		Preload("Asset").
		Where("transaction_id = ?", transaction.ID).
		Find(&mutations).Error; err != nil {
		return nil, err
	}

	return &dto.MutationResponse{
		Transaction: mapTransactionHeaderToResponse(transaction),
		Items:       mapMutationItemsToResponse(mutations),
	}, nil
}

func GetAllMutations(filter dto.TransactionListFilter) ([]dto.MutationResponse, int64, error) {
	query := config.DB.Model(&models.Transaction{}).Where("transaction_type = ?", TxMutation)

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

	responses := make([]dto.MutationResponse, len(transactions))
	for i, t := range transactions {
		var mutations []models.TransactionMutation
		config.DB.Preload("Asset").Where("transaction_id = ?", t.ID).Find(&mutations)

		responses[i] = dto.MutationResponse{
			Transaction: mapTransactionHeaderToResponse(t),
			Items:       mapMutationItemsToResponse(mutations),
		}
	}

	return responses, total, nil
}

func UpdateMutation(transactionNumber string, userID string, req dto.CreateMutationRequest) (*dto.MutationResponse, error) {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxMutation).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("mutation transaction not found")
		}
		return nil, err
	}

	if transaction.Status != models.TransactionStatusDraft {
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

	if err := tx.Where("transaction_id = ?", transaction.ID).Delete(&models.TransactionMutation{}).Error; err != nil {
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
		// FIX: item.AssetID sudah uint
		if err := tx.First(&asset, item.AssetID).Error; err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("asset not found: %d", item.AssetID)
			}
			return nil, err
		}

		mutation := models.TransactionMutation{
			TransactionID:     transaction.ID,
			TransactionNumber: transaction.TransactionNumber,
			AssetID:           asset.ID,
			AssetNumber:       item.AssetNumber,
			FromBranchCode:    item.FromBranchCode,
			ToBranchCode:      item.ToBranchCode,
			FromLocation:      item.FromLocation,
			ToLocation:        item.ToLocation,
			Notes:             item.Notes,
		}

		if err := tx.Create(&mutation).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetMutationByTransactionNumber(transaction.TransactionNumber)
}

func DeleteMutation(transactionNumber string, userID string) error {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxMutation).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("mutation transaction not found")
		}
		return err
	}

	if transaction.Status != models.TransactionStatusDraft {
		return errors.New("can only delete DRAFT transactions")
	}

	if transaction.CreatedBy != userID {
		return errors.New("you can only delete your own transactions")
	}

	MarkTransactionAsExpired(transactionNumber)
	return config.DB.Delete(&transaction).Error
}
