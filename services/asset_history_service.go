package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"

	"gorm.io/gorm"
)

func GetAssetHistory(assetNumber string) ([]dto.AssetHistoryResponse, error) {
	var asset models.Asset
	if err := config.DB.Where("asset_number = ?", assetNumber).First(&asset).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("asset not found")
		}
		return nil, err
	}

	var histories []models.AssetHistory
	if err := config.DB.
		Preload("ChangedByUser").
		Where("asset_id = ?", asset.ID).
		Order("created_at DESC").
		Find(&histories).Error; err != nil {
		return nil, err
	}

	return mapAssetHistoriesToResponse(histories), nil
}

func GetAllAssetHistories(filter dto.AssetHistoryFilter) ([]dto.AssetHistoryResponse, int64, error) {
	query := config.DB.Model(&models.AssetHistory{})

	if filter.AssetID != nil {
		query = query.Where("asset_id = ?", *filter.AssetID)
	}
	if filter.TransactionType != nil {
		query = query.Where("transaction_type = ?", *filter.TransactionType)
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

	var histories []models.AssetHistory
	if err := query.
		Preload("Asset").
		Preload("ChangedByUser").
		Order("created_at DESC").
		Find(&histories).Error; err != nil {
		return nil, 0, err
	}

	return mapAssetHistoriesToResponse(histories), total, nil
}

// CreateAssetHistory inserts an audit row. Accepts an explicit *gorm.DB so
// callers running inside a transaction (e.g. stock opname execute) can keep
// the write atomic with their other changes; pass config.DB otherwise.
func CreateAssetHistory(db *gorm.DB, history models.AssetHistory) error {
	return db.Create(&history).Error
}
