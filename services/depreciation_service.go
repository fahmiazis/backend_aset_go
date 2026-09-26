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

// ============================================================================
// Depreciation Settings
// ============================================================================

func CreateDepreciationSetting(req dto.CreateDepreciationSettingRequest) (*dto.DepreciationSettingResponse, error) {
	if req.SettingType == models.SettingTypeDefault {
		// DEFAULT berlaku untuk semua aset tanpa setting lain, tidak menunjuk
		// ke kategori/aset tertentu
		req.ReferenceID = nil
		req.ReferenceValue = nil
		if req.IsActive {
			if err := ensureSingleActiveDefault(0); err != nil {
				return nil, err
			}
		}
	} else if req.ReferenceID == nil {
		// FIX: Validasi pakai ReferenceID bukan CategoryID/AssetID terpisah
		return nil, errors.New("reference_id is required")
	}

	// Validasi reference exists berdasarkan setting_type
	if req.SettingType == models.SettingTypeCategory {
		var category models.AssetCategory
		if err := config.DB.First(&category, "id = ?", *req.ReferenceID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("category not found")
			}
			return nil, err
		}
		// Set reference_value dari category_code
		if req.ReferenceValue == nil {
			req.ReferenceValue = &category.CategoryCode
		}
	} else if req.SettingType == models.SettingTypeAsset {
		var asset models.Asset
		if err := config.DB.First(&asset, "id = ?", *req.ReferenceID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("asset not found")
			}
			return nil, err
		}
		// Set reference_value dari asset_number
		if req.ReferenceValue == nil {
			req.ReferenceValue = &asset.AssetNumber
		}
	}

	// Parse StartDate
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, errors.New("invalid start_date format, use YYYY-MM-DD")
	}

	// Parse EndDate (optional)
	var endDate *time.Time
	if req.EndDate != nil && *req.EndDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return nil, errors.New("invalid end_date format, use YYYY-MM-DD")
		}
		endDate = &parsed
	}

	// FIX: pakai ReferenceID & ReferenceValue, hapus CategoryID/AssetID/ResidualValue
	setting := models.DepreciationSetting{
		SettingType:        req.SettingType,
		ReferenceID:        req.ReferenceID,
		ReferenceValue:     req.ReferenceValue,
		CalculationMethod:  req.CalculationMethod,
		DepreciationPeriod: req.DepreciationPeriod,
		UsefulLifeMonths:   req.UsefulLifeMonths,
		DepreciationRate:   req.DepreciationRate,
		StartDate:          startDate,
		EndDate:            endDate,
		IsActive:           req.IsActive,
	}

	if err := config.DB.Create(&setting).Error; err != nil {
		return nil, err
	}

	return GetDepreciationSettingByID(setting.ID)
}

// ensureSingleActiveDefault — setting DEFAULT yang aktif hanya boleh satu,
// kalau tidak aset tanpa setting akan disusutkan dengan aturan yang acak.
func ensureSingleActiveDefault(excludeID uint) error {
	var count int64
	config.DB.Model(&models.DepreciationSetting{}).
		Where("setting_type = ? AND is_active = ? AND id <> ?", models.SettingTypeDefault, true, excludeID).
		Count(&count)
	if count > 0 {
		return errors.New("an active DEFAULT depreciation setting already exists, deactivate it first")
	}
	return nil
}

func UpdateDepreciationSetting(id uint, req dto.UpdateDepreciationSettingRequest) (*dto.DepreciationSettingResponse, error) {
	var setting models.DepreciationSetting
	if err := config.DB.First(&setting, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("depreciation setting not found")
		}
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.CalculationMethod != nil {
		updates["calculation_method"] = *req.CalculationMethod
	}
	if req.DepreciationPeriod != nil {
		updates["depreciation_period"] = *req.DepreciationPeriod
	}
	if req.UsefulLifeMonths != nil {
		updates["useful_life_months"] = *req.UsefulLifeMonths
	}
	if req.DepreciationRate != nil {
		updates["depreciation_rate"] = *req.DepreciationRate
	}
	if req.IsActive != nil {
		if *req.IsActive && setting.SettingType == models.SettingTypeDefault {
			if err := ensureSingleActiveDefault(setting.ID); err != nil {
				return nil, err
			}
		}
		updates["is_active"] = *req.IsActive
	}
	// FIX: tambah EndDate update
	if req.EndDate != nil {
		if *req.EndDate == "" {
			updates["end_date"] = nil
		} else {
			parsed, err := time.Parse("2006-01-02", *req.EndDate)
			if err != nil {
				return nil, errors.New("invalid end_date format, use YYYY-MM-DD")
			}
			updates["end_date"] = parsed
		}
	}

	if err := config.DB.Model(&setting).Updates(updates).Error; err != nil {
		return nil, err
	}

	return GetDepreciationSettingByID(id)
}

func GetDepreciationSettingByID(id uint) (*dto.DepreciationSettingResponse, error) {
	var setting models.DepreciationSetting

	// FIX: hapus Preload("Category") & Preload("Asset") karena model tidak punya relasi itu lagi
	if err := config.DB.First(&setting, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("depreciation setting not found")
		}
		return nil, err
	}

	response := mapDepreciationSettingToResponse(setting)
	return &response, nil
}

func GetAllDepreciationSettings() ([]dto.DepreciationSettingResponse, error) {
	var settings []models.DepreciationSetting

	// FIX: hapus Preload
	if err := config.DB.
		Order("created_at DESC").
		Find(&settings).Error; err != nil {
		return nil, err
	}

	return mapDepreciationSettingsToResponse(settings), nil
}

func DeleteDepreciationSetting(id uint) error {
	var setting models.DepreciationSetting
	if err := config.DB.First(&setting, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("depreciation setting not found")
		}
		return err
	}

	return config.DB.Delete(&setting).Error
}

// ============================================================================
// Monthly Depreciation Calculations
// ============================================================================

func GetMonthlyDepreciationCalculations(period string) ([]dto.MonthlyDepreciationResponse, error) {
	var calculations []models.MonthlyDepreciationCalculation

	// FIX: query pakai period VARCHAR(7) bukan month+year terpisah
	if err := config.DB.
		Preload("Asset").
		Where("period = ?", period).
		Order("created_at DESC").
		Find(&calculations).Error; err != nil {
		return nil, err
	}

	return mapMonthlyDepreciationsToResponse(calculations), nil
}

// CanRunDepreciation — sama dengan pemeriksaan RequirePermission pada
// POST /depreciation/calculate, dipakai frontend untuk menampilkan tombol.
func CanRunDepreciation(userID string) bool {
	return userHasMenuPermission(userRoleIDs(userID), "/depreciation/calculate", []string{"run_depreciation"})
}

var depreciableAssetStatuses = []string{
	"ACTIVE",
	models.AssetStatusAvailable,
	models.AssetStatusInMutation,
	models.AssetStatusInDisposal,
	models.AssetStatusInHandover,
}

// CalculateMonthlyDepreciation — mengembalikan jumlah aset yang dihitung.
func CalculateMonthlyDepreciation(userID string, req dto.CalculateDepreciationRequest) (int, error) {
	processed := 0
	// FIX: cek pakai period string bukan month+year
	var existing models.MonthlyDepreciationCalculation
	if err := config.DB.
		Where("period = ? AND is_locked = ?", req.Period, true).
		First(&existing).Error; err == nil {
		return 0, errors.New("depreciation already calculated and locked for this period")
	}

	// Parse period untuk dapat calculation_date (hari pertama bulan tersebut)
	calculationDate, err := time.Parse("2006-01", req.Period)
	if err != nil {
		return 0, errors.New("invalid period format, use YYYY-MM")
	}
	if req.Period > time.Now().Format("2006-01") {
		return 0, errors.New("cannot calculate depreciation for a future period")
	}

	// Aset yang masih dimiliki perusahaan tetap disusutkan, termasuk yang sedang
	// dalam proses mutasi / disposal / serah terima. Dulu hanya AVAILABLE
	// (konstantanya tertulis dua kali), sehingga aset yang sedang diproses
	// berhenti disusutkan. DISPOSED dan PENDING_RECEIPT (belum diterima) tidak.
	var assets []models.Asset
	if err := config.DB.
		Where("asset_status IN ?", depreciableAssetStatuses).
		Find(&assets).Error; err != nil {
		return 0, err
	}

	// Setting DEFAULT (kalau ada) — cadangan untuk aset tanpa setting ASSET
	// maupun CATEGORY. Dibaca sekali, bukan per aset.
	var defaultSetting *models.DepreciationSetting
	var fallback models.DepreciationSetting
	if err := config.DB.Where("setting_type = ? AND is_active = ?", models.SettingTypeDefault, true).
		Order("id DESC").First(&fallback).Error; err == nil {
		defaultSetting = &fallback
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, asset := range assets {
		// Get depreciation setting - coba asset-specific dulu
		var setting models.DepreciationSetting

		// FIX: query pakai reference_id bukan asset_id/category_id
		err := tx.Where("setting_type = ? AND reference_id = ? AND is_active = ?",
			models.SettingTypeAsset, asset.ID, true).
			First(&setting).Error

		if err != nil {
			// Fallback ke category setting
			if asset.CategoryID != nil {
				err = tx.Where("setting_type = ? AND reference_id = ? AND is_active = ?",
					models.SettingTypeCategory, *asset.CategoryID, true).
					First(&setting).Error
			}
		}

		if err != nil && defaultSetting != nil {
			// Fallback terakhir: setting DEFAULT
			setting = *defaultSetting
			err = nil
		}

		if err != nil {
			// Skip asset yang tidak punya depreciation setting
			continue
		}

		var calculated int64
		tx.Model(&models.MonthlyDepreciationCalculation{}).
			Where("asset_id = ? AND period = ?", asset.ID, req.Period).
			Count(&calculated)
		if calculated > 0 {
			continue
		}

		// Get current active asset value
		var currentValue models.AssetValue
		if err := tx.Where("asset_id = ? AND is_active = ?", asset.ID, true).
			First(&currentValue).Error; err != nil {
			continue
		}

		// Nilai aktif yang sudah berada di/sesudah period ini (mis. aset baru
		// dibuat bulan ini) tidak disusutkan mundur.
		// Dibandingkan per bulan (YYYY-MM): DB dibaca dengan loc=Local sedangkan
		// calculationDate hasil time.Parse berzona UTC.
		if currentValue.EffectiveDate.Format("2006-01") >= req.Period {
			continue
		}

		// FIX: hapus residualValue dari kalkulasi, pakai DepreciationRate langsung
		depreciationAmount := calculateDepreciation(
			currentValue.BookValue,
			setting.CalculationMethod,
			setting.UsefulLifeMonths,
			setting.DepreciationRate,
		)

		// Pastikan book value tidak negatif
		newBookValue := currentValue.BookValue - depreciationAmount
		if newBookValue < 0 {
			newBookValue = 0
			depreciationAmount = currentValue.BookValue
		}

		beginningAccumDepr := currentValue.AccumulatedDepreciation
		endingAccumDepr := beginningAccumDepr + depreciationAmount

		// FIX: Create calculation record dengan field baru
		calculationMethod := setting.CalculationMethod
		calculation := models.MonthlyDepreciationCalculation{
			AssetID:                          asset.ID,
			Period:                           req.Period,      // FIX: pakai Period
			CalculationDate:                  calculationDate, // FIX: tambah
			BeginningBookValue:               currentValue.BookValue,
			DepreciationAmount:               depreciationAmount,
			BeginningAccumulatedDepreciation: beginningAccumDepr, // FIX: tambah
			EndingAccumulatedDepreciation:    endingAccumDepr,    // FIX: tambah
			EndingBookValue:                  newBookValue,
			CalculationMethod:                &calculationMethod, // FIX: *string
			DepreciationSettingID:            &setting.ID,        // FIX: tambah
			IsLocked:                         false,
		}

		// Aset yang sudah dihitung untuk period ini dilewati. Dulu kalkulasinya
		// di-update lalu asset_values baru tetap dibuat dari nilai yang SUDAH
		// disusutkan — menjalankan dua kali berarti penyusutan dobel.
		if err := tx.Create(&calculation).Error; err != nil {
			tx.Rollback()
			return 0, err
		}

		// Update asset value lama jadi tidak active
		if err := tx.Model(&currentValue).Update("is_active", false).Error; err != nil {
			tx.Rollback()
			return 0, err
		}

		// Buat asset value baru
		newAssetValue := models.AssetValue{
			AssetID:                 asset.ID,
			EffectiveDate:           calculationDate,
			BookValue:               newBookValue,
			AcquisitionValue:        currentValue.AcquisitionValue,
			AccumulatedDepreciation: endingAccumDepr,
			Condition:               currentValue.Condition,
			PhysicalStatus:          currentValue.PhysicalStatus,
			AssetStatus:             currentValue.AssetStatus,
			IsActive:                true,
		}

		if err := tx.Create(&newAssetValue).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
		processed++
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}
	return processed, nil
}

func LockMonthlyDepreciation(period string) error {
	result := config.DB.Model(&models.MonthlyDepreciationCalculation{}).
		Where("period = ? AND is_locked = ?", period, false).
		Update("is_locked", true)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no unlocked calculations found for period %s", period)
	}
	return nil
}

// FIX: hapus residualValue & usefulLifeMonths dari parameter, pakai rate langsung
func calculateDepreciation(bookValue float64, method string, usefulLifeMonths int, rate *float64) float64 {
	switch method {
	case models.CalculationMethodStraightLine:
		if usefulLifeMonths <= 0 {
			return 0
		}
		return bookValue / float64(usefulLifeMonths)
	case models.CalculationMethodDecliningBalance:
		if rate == nil || *rate <= 0 {
			return 0
		}
		return bookValue * (*rate / 100)
	default:
		return 0
	}
}
