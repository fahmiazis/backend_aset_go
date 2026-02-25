package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"strconv"

	"gorm.io/gorm"
)

// ============================================================================
// ASSET CATEGORY SERVICES
// ============================================================================

func GetAllAssetCategories() ([]dto.AssetCategoryResponse, error) {
	var categories []models.AssetCategory

	if err := config.DB.Order("category_name ASC").Find(&categories).Error; err != nil {
		return nil, err
	}

	return mapAssetCategoriesToResponse(categories), nil
}

func GetAssetCategoryByID(id string) (*dto.AssetCategoryResponse, error) {
	var category models.AssetCategory

	if err := config.DB.First(&category, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("asset category not found")
		}
		return nil, err
	}

	response := mapAssetCategoryToResponse(category)
	return &response, nil
}

func GetAssetCategoryByCode(code string) (*dto.AssetCategoryResponse, error) {
	var category models.AssetCategory

	if err := config.DB.First(&category, "category_code = ?", code).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("asset category not found")
		}
		return nil, err
	}

	response := mapAssetCategoryToResponse(category)
	return &response, nil
}

func CreateAssetCategory(req dto.CreateAssetCategoryRequest) (*dto.AssetCategoryResponse, error) {
	// Check if category code already exists
	var existing models.AssetCategory
	if err := config.DB.Where("category_code = ?", req.CategoryCode).First(&existing).Error; err == nil {
		return nil, errors.New("category code already exists")
	}

	category := models.AssetCategory{
		CategoryCode: req.CategoryCode,
		CategoryName: req.CategoryName,
		Description:  req.Description,
		IsActive:     req.IsActive,
	}

	if err := config.DB.Create(&category).Error; err != nil {
		return nil, err
	}

	// FIXED: Convert uint to string
	return GetAssetCategoryByID(strconv.FormatUint(uint64(category.ID), 10))
}

func UpdateAssetCategory(id string, req dto.UpdateAssetCategoryRequest) (*dto.AssetCategoryResponse, error) {
	var category models.AssetCategory
	if err := config.DB.First(&category, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("asset category not found")
		}
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.CategoryCode != nil && *req.CategoryCode != "" {
		// Check if new code already exists
		var existing models.AssetCategory
		if err := config.DB.Where("category_code = ? AND id != ?", *req.CategoryCode, id).First(&existing).Error; err == nil {
			return nil, errors.New("category code already exists")
		}
		updates["category_code"] = *req.CategoryCode
	}
	if req.CategoryName != nil && *req.CategoryName != "" {
		updates["category_name"] = *req.CategoryName
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := config.DB.Model(&category).Updates(updates).Error; err != nil {
		return nil, err
	}

	return GetAssetCategoryByID(id)
}

func DeleteAssetCategory(id string) error {
	var category models.AssetCategory
	if err := config.DB.First(&category, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("asset category not found")
		}
		return err
	}

	// Check if category is used by any assets
	var count int64
	config.DB.Model(&models.Asset{}).Where("category_id = ?", id).Count(&count)
	if count > 0 {
		return errors.New("cannot delete category that is used by assets")
	}

	return config.DB.Delete(&category).Error
}

// ============================================================================
// ALTERNATIVE: Return response directly without calling GetByID again
// ============================================================================

func CreateAssetCategory_Alternative(req dto.CreateAssetCategoryRequest) (*dto.AssetCategoryResponse, error) {
	// Check if category code already exists
	var existing models.AssetCategory
	if err := config.DB.Where("category_code = ?", req.CategoryCode).First(&existing).Error; err == nil {
		return nil, errors.New("category code already exists")
	}

	category := models.AssetCategory{
		CategoryCode: req.CategoryCode,
		CategoryName: req.CategoryName,
		Description:  req.Description,
		IsActive:     req.IsActive,
	}

	if err := config.DB.Create(&category).Error; err != nil {
		return nil, err
	}

	// ALTERNATIVE: Map directly without additional DB call
	response := mapAssetCategoryToResponse(category)
	return &response, nil
}
