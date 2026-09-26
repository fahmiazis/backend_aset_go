package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"

	"gorm.io/gorm"
)

// AssetViewer — siapa yang meminta data aset. Non-admin hanya boleh melihat
// aset di cabang yang dia punya (homebase + assignment/temporary di
// user_branchs), sama dengan aturan akses cabang di approval.
type AssetViewer struct {
	UserID  string
	IsAdmin bool
}

// AssetBranchScope — kode cabang yang boleh dilihat. all=true untuk admin.
func AssetBranchScope(viewer AssetViewer) (codes []string, all bool) {
	if viewer.IsAdmin {
		return nil, true
	}
	return userBranchCodes(viewer.UserID), false
}

// CanViewAsset — dipakai endpoint detail aset.
func CanViewAsset(viewer AssetViewer, asset *dto.AssetResponse) bool {
	codes, all := AssetBranchScope(viewer)
	if all {
		return true
	}
	if asset.BranchCode == nil {
		return false
	}
	for _, code := range codes {
		if code == *asset.BranchCode {
			return true
		}
	}
	return false
}

// GetViewableAssetBranches — pilihan dropdown cabang di halaman aset.
func GetViewableAssetBranches(viewer AssetViewer) ([]dto.BranchOption, error) {
	codes, all := AssetBranchScope(viewer)
	query := config.DB.Model(&models.Branch{}).Order("branch_code")
	if !all {
		if len(codes) == 0 {
			return []dto.BranchOption{}, nil
		}
		query = query.Where("branch_code IN ?", codes)
	}
	var branches []models.Branch
	if err := query.Find(&branches).Error; err != nil {
		return nil, err
	}
	result := make([]dto.BranchOption, 0, len(branches))
	for _, b := range branches {
		result = append(result, dto.BranchOption{BranchCode: b.BranchCode, BranchName: b.BranchName})
	}
	return result, nil
}

func GetAllAssets(filter dto.AssetListFilter, viewer AssetViewer) ([]dto.AssetResponse, int64, error) {
	query := config.DB.Model(&models.Asset{}).Where("deleted_at IS NULL")

	// Batasi ke cabang milik user. Berlaku untuk SEMUA pemakai GET /assets
	// (halaman aset, pemilih aset di mutasi/disposal/stock opname, dashboard).
	if codes, all := AssetBranchScope(viewer); !all {
		if len(codes) == 0 {
			return []dto.AssetResponse{}, 0, nil
		}
		query = query.Where("branch_code IN ?", codes)
	}

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
		// FIX: ILIKE itu sintaks PostgreSQL — MariaDB menolaknya dengan syntax
		// error, jadi SELURUH request /assets yang memakai `search` selalu gagal.
		// LIKE di MySQL/MariaDB sudah case-insensitive untuk collation utf8mb4_*_ci.
		// Tanda kurung dipasang eksplisit supaya OR tidak melebar ke filter lain
		// (branch_code, asset_status, category_id).
		query = query.Where("(asset_number LIKE ? OR asset_name LIKE ?)", search, search)
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
