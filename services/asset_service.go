package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"

	"gorm.io/gorm"
)

func GetAllAssets(filter dto.AssetListFilter) ([]dto.AssetResponse, int64, error) {
	query := config.DB.Model(&models.Asset{}).Where("deleted_at IS NULL")

	if filter.BranchCode != nil {
		query = query.Where("branch_code = ?", *filter.BranchCode)
	}
	if filter.CategoryID != nil {
		query = query.Where("category_id = ?", *filter.CategoryID)
	}
	if filter.AssetStatus != nil {
		query = query.Where("asset_status = ?", *filter.AssetStatus)
	}
	if filter.Search != nil && *filter.Search != "" {
		search := "%" + *filter.Search + "%"
		query = query.Where("asset_number ILIKE ? OR asset_name ILIKE ?", search, search)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.Limit
	query = query.Offset(offset).Limit(filter.Limit)

	var assets []models.Asset
	if err := query.
		Preload("Category").
		Order("created_at DESC").
		Find(&assets).Error; err != nil {
		return nil, 0, err
	}

	// Get current values for all assets
	responses := make([]dto.AssetResponse, len(assets))
	for i, asset := range assets {
		responses[i] = mapAssetToResponse(asset)

		// Get current active value
		var currentValue models.AssetValue
		if err := config.DB.
			Where("asset_id = ? AND is_active = ?", asset.ID, true).
			First(&currentValue).Error; err == nil {
			value := mapAssetValueToResponse(currentValue)
			responses[i].CurrentValue = &value
		}
	}

	return responses, total, nil
}

func GetAssetByNumber(assetNumber string) (*dto.AssetResponse, error) {
	var asset models.Asset

	if err := config.DB.
		Preload("Category").
		Where("asset_number = ? AND deleted_at IS NULL", assetNumber).
		First(&asset).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("asset not found")
		}
		return nil, err
	}

	response := mapAssetToResponse(asset)

	// Get current active value
	var currentValue models.AssetValue
	if err := config.DB.
		Where("asset_id = ? AND is_active = ?", asset.ID, true).
		First(&currentValue).Error; err == nil {
		value := mapAssetValueToResponse(currentValue)
		response.CurrentValue = &value
	}

	return &response, nil
}

func GetAssetByID(assetID string) (*dto.AssetResponse, error) {
	var asset models.Asset

	if err := config.DB.
		Preload("Category").
		Where("id = ? AND deleted_at IS NULL", assetID).
		First(&asset).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("asset not found")
		}
		return nil, err
	}

	response := mapAssetToResponse(asset)

	// Get current active value
	var currentValue models.AssetValue
	if err := config.DB.
		Where("asset_id = ? AND is_active = ?", asset.ID, true).
		First(&currentValue).Error; err == nil {
		value := mapAssetValueToResponse(currentValue)
		response.CurrentValue = &value
	}

	return &response, nil
}

func GetAssetValueHistory(assetNumber string) ([]dto.AssetValueResponse, error) {
	var asset models.Asset
	if err := config.DB.
		Where("asset_number = ? AND deleted_at IS NULL", assetNumber).
		First(&asset).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("asset not found")
		}
		return nil, err
	}

	var values []models.AssetValue
	if err := config.DB.
		Where("asset_id = ?", asset.ID).
		Order("effective_date DESC").
		Find(&values).Error; err != nil {
		return nil, err
	}

	return mapAssetValuesToResponse(values), nil
}
