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

// validateBranchExists cek apakah branch_code terdaftar di tabel branches
func validateBranchExists(branchCode string) error {
	var count int64
	if err := config.DB.Model(&models.Branch{}).
		Where("branch_code = ?", branchCode).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("branch not found: %s", branchCode)
	}
	return nil
}

func CreateProcurement(userID string, req dto.CreateProcurementRequest) (*dto.ProcurementResponse, error) {
	// ---- VALIDASI DULU SEBELUM GENERATE NOMOR TRANSAKSI ----
	// Supaya nomor tidak ter-reserve kalau input tidak valid

	transactionDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		return nil, errors.New("invalid transaction date format, use YYYY-MM-DD")
	}

	homebase, err := GetUserActiveHomebase(userID)
	if err != nil {
		return nil, err
	}

	isHO := homebase.Branch.BranchType == "HO"
	for _, item := range req.Items {
		// Validasi category exist
		var category models.AssetCategory
		if err := config.DB.First(&category, item.CategoryID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("category not found: %d", item.CategoryID)
			}
			return nil, err
		}

		// Validasi items.branch_code
		if item.BranchCode != nil && *item.BranchCode != "" {
			if !isHO {
				if *item.BranchCode != homebase.Branch.BranchCode {
					return nil, fmt.Errorf("item '%s': branch_code must match your homebase (%s)", item.ItemName, homebase.Branch.BranchCode)
				}
			} else {
				if err := validateBranchExists(*item.BranchCode); err != nil {
					return nil, fmt.Errorf("item '%s': %w", item.ItemName, err)
				}
			}
		}

		// Validasi details.branch_code
		for _, detail := range item.Details {
			if err := validateBranchExists(detail.BranchCode); err != nil {
				return nil, fmt.Errorf("item '%s' detail: %w", item.ItemName, err)
			}
			if !isHO && detail.BranchCode != homebase.Branch.BranchCode {
				return nil, fmt.Errorf("item '%s' detail: only HO users can add details for other branches (your homebase: %s)", item.ItemName, homebase.Branch.BranchCode)
			}
		}
	}

	// ---- SEMUA VALID, BARU GENERATE NOMOR TRANSAKSI ----
	transactionNumber, err := GenerateTransactionNumber(userID, TxProcurement)
	if err != nil {
		return nil, err
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	transaction := models.Transaction{
		TransactionNumber: transactionNumber,
		TransactionType:   TxProcurement,
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
		var category models.AssetCategory
		if err := tx.First(&category, item.CategoryID).Error; err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("category not found: %d", item.CategoryID)
			}
			return nil, err
		}

		totalPrice := float64(item.Quantity) * item.UnitPrice

		branchCode := homebase.Branch.BranchCode
		if item.BranchCode != nil && *item.BranchCode != "" {
			branchCode = *item.BranchCode
		}

		procurement := models.TransactionProcurement{
			TransactionID:     transaction.ID,
			TransactionNumber: transactionNumber,
			ItemName:          item.ItemName,
			CategoryID:        &category.ID,
			Quantity:          item.Quantity,
			UnitPrice:         item.UnitPrice,
			TotalPrice:        totalPrice,
			BranchCode:        branchCode,
			Notes:             item.Notes,
		}

		if err := tx.Create(&procurement).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		if len(item.Details) > 0 {
			totalDetailQty := 0
			for _, detail := range item.Details {
				procDetail := models.TransactionProcurementDetail{
					TransactionProcurementID: procurement.ID,
					BranchCode:               detail.BranchCode,
					Quantity:                 detail.Quantity,
					RequesterName:            detail.RequesterName,
					Notes:                    detail.Notes,
				}

				if err := tx.Create(&procDetail).Error; err != nil {
					tx.Rollback()
					return nil, err
				}

				totalDetailQty += detail.Quantity
			}

			if totalDetailQty != item.Quantity {
				tx.Rollback()
				return nil, fmt.Errorf("total detail quantity must match item quantity for item: %s", item.ItemName)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetProcurementByTransactionNumber(transactionNumber)
}

func GetProcurementByTransactionNumber(transactionNumber string) (*dto.ProcurementResponse, error) {
	var transaction models.Transaction
	// FIX: hapus Preload("Creator") & Preload("Approver")
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxProcurement).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("procurement transaction not found")
		}
		return nil, err
	}

	var procurements []models.TransactionProcurement
	if err := config.DB.
		Preload("Category").
		Preload("TransactionProcurementDetails").
		Where("transaction_id = ?", transaction.ID).
		Find(&procurements).Error; err != nil {
		return nil, err
	}

	return &dto.ProcurementResponse{
		Transaction: mapTransactionHeaderToResponse(transaction),
		Items:       mapProcurementItemsToResponse(procurements),
	}, nil
}

func GetAllProcurements(filter dto.TransactionListFilter) ([]dto.ProcurementResponse, int64, error) {
	query := config.DB.Model(&models.Transaction{}).Where("transaction_type = ?", TxProcurement)

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

	responses := make([]dto.ProcurementResponse, len(transactions))
	for i, t := range transactions {
		var procurements []models.TransactionProcurement
		config.DB.
			Preload("Category").
			Preload("TransactionProcurementDetails").
			Where("transaction_id = ?", t.ID).
			Find(&procurements)

		responses[i] = dto.ProcurementResponse{
			Transaction: mapTransactionHeaderToResponse(t),
			Items:       mapProcurementItemsToResponse(procurements),
		}
	}

	return responses, total, nil
}

func UpdateProcurement(transactionNumber string, userID string, req dto.CreateProcurementRequest) (*dto.ProcurementResponse, error) {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxProcurement).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("procurement transaction not found")
		}
		return nil, err
	}

	if transaction.Status != models.TransactionStatusDraft { // FIX: pakai models konstanta
		return nil, errors.New("can only update DRAFT transactions")
	}

	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only update your own transactions")
	}

	homebase, err := GetUserActiveHomebase(userID)
	if err != nil {
		return nil, err
	}

	// ---- VALIDASI DULU SEBELUM MULAI TX ----
	isHO := homebase.Branch.BranchType == "HO"
	for _, item := range req.Items {
		// Validasi category exist
		var category models.AssetCategory
		if err := config.DB.First(&category, item.CategoryID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("category not found: %d", item.CategoryID)
			}
			return nil, err
		}

		// Validasi items.branch_code
		if item.BranchCode != nil && *item.BranchCode != "" {
			if !isHO {
				if *item.BranchCode != homebase.Branch.BranchCode {
					return nil, fmt.Errorf("item '%s': branch_code must match your homebase (%s)", item.ItemName, homebase.Branch.BranchCode)
				}
			} else {
				if err := validateBranchExists(*item.BranchCode); err != nil {
					return nil, fmt.Errorf("item '%s': %w", item.ItemName, err)
				}
			}
		}

		// Validasi details.branch_code
		for _, detail := range item.Details {
			if err := validateBranchExists(detail.BranchCode); err != nil {
				return nil, fmt.Errorf("item '%s' detail: %w", item.ItemName, err)
			}
			if !isHO && detail.BranchCode != homebase.Branch.BranchCode {
				return nil, fmt.Errorf("item '%s' detail: only HO users can add details for other branches (your homebase: %s)", item.ItemName, homebase.Branch.BranchCode)
			}
		}
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("transaction_id = ?", transaction.ID).Delete(&models.TransactionProcurement{}).Error; err != nil {
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
		var category models.AssetCategory
		// FIX: item.CategoryID sudah uint
		if err := tx.First(&category, item.CategoryID).Error; err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("category not found: %d", item.CategoryID)
			}
			return nil, err
		}

		totalPrice := float64(item.Quantity) * item.UnitPrice
		branchCode := homebase.Branch.BranchCode
		if item.BranchCode != nil && *item.BranchCode != "" {
			branchCode = *item.BranchCode
		}

		procurement := models.TransactionProcurement{
			TransactionID:     transaction.ID,
			TransactionNumber: transaction.TransactionNumber,
			ItemName:          item.ItemName,
			CategoryID:        &category.ID,
			Quantity:          item.Quantity,
			UnitPrice:         item.UnitPrice,
			TotalPrice:        totalPrice,
			BranchCode:        branchCode,
			Notes:             item.Notes,
		}

		if err := tx.Create(&procurement).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		if len(item.Details) > 0 {
			totalDetailQty := 0
			for _, detail := range item.Details {
				procDetail := models.TransactionProcurementDetail{
					TransactionProcurementID: procurement.ID,
					BranchCode:               detail.BranchCode,
					Quantity:                 detail.Quantity,
					RequesterName:            detail.RequesterName,
					Notes:                    detail.Notes,
				}

				if err := tx.Create(&procDetail).Error; err != nil {
					tx.Rollback()
					return nil, err
				}

				totalDetailQty += detail.Quantity
			}

			if totalDetailQty != item.Quantity {
				tx.Rollback()
				return nil, fmt.Errorf("total detail quantity must match item quantity for item: %s", item.ItemName)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return GetProcurementByTransactionNumber(transaction.TransactionNumber)
}

func DeleteProcurement(transactionNumber string, userID string) error {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxProcurement).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("procurement transaction not found")
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
