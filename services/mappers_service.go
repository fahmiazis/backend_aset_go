package services

import (
	"backend-go/dto"
	"backend-go/models"
)

// ============================================================================
// TRANSACTION MAPPERS
// FIX: hapus strconv, ID sudah uint. Hapus Creator/Approver relation (tidak ada di model).
// ============================================================================

func mapTransactionHeaderToResponse(tx models.Transaction) dto.TransactionHeaderResponse {
	return dto.TransactionHeaderResponse{
		ID:                tx.ID,
		TransactionNumber: tx.TransactionNumber,
		TransactionType:   tx.TransactionType,
		TransactionDate:   tx.TransactionDate,
		Status:            tx.Status,
		Notes:             tx.Notes,
		CreatedBy:         tx.CreatedBy,
		ApprovedBy:        tx.ApprovedBy,
		ApprovedAt:        tx.ApprovedAt,
		CreatedAt:         tx.CreatedAt,
		UpdatedAt:         tx.UpdatedAt,
	}
}

func mapTransactionHeadersToResponse(transactions []models.Transaction) []dto.TransactionHeaderResponse {
	response := make([]dto.TransactionHeaderResponse, len(transactions))
	for i, tx := range transactions {
		response[i] = mapTransactionHeaderToResponse(tx)
	}
	return response
}

// ============================================================================
// PROCUREMENT MAPPERS
// FIX: hapus strconv, CategoryID sudah *uint di DTO
// ============================================================================

func mapProcurementItemToResponse(item models.TransactionProcurement) dto.ProcurementItemResponse {
	response := dto.ProcurementItemResponse{
		ID:                item.ID,
		TransactionID:     item.TransactionID,
		TransactionNumber: item.TransactionNumber,
		ItemName:          item.ItemName,
		CategoryID:        item.CategoryID, // FIX: sudah *uint, langsung assign
		Quantity:          item.Quantity,
		UnitPrice:         item.UnitPrice,
		TotalPrice:        item.TotalPrice,
		BranchCode:        item.BranchCode,
		Notes:             item.Notes,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}

	if item.Category != nil {
		response.CategoryName = &item.Category.CategoryName
	}

	if len(item.TransactionProcurementDetails) > 0 {
		details := make([]dto.ProcurementDetailResponse, len(item.TransactionProcurementDetails))
		for i, d := range item.TransactionProcurementDetails {
			details[i] = dto.ProcurementDetailResponse{
				ID:                       d.ID,
				TransactionProcurementID: d.TransactionProcurementID,
				BranchCode:               d.BranchCode,
				Quantity:                 d.Quantity,
				RequesterName:            d.RequesterName,
				Notes:                    d.Notes,
				CreatedAt:                d.CreatedAt,
				UpdatedAt:                d.UpdatedAt,
			}
		}
		response.Details = details
	}

	return response
}

func mapProcurementItemsToResponse(items []models.TransactionProcurement) []dto.ProcurementItemResponse {
	response := make([]dto.ProcurementItemResponse, len(items))
	for i, item := range items {
		response[i] = mapProcurementItemToResponse(item)
	}
	return response
}

// ============================================================================
// MUTATION MAPPERS
// FIX: hapus strconv
// ============================================================================

func mapMutationItemToResponse(item models.TransactionMutation) dto.MutationItemResponse {
	response := dto.MutationItemResponse{
		ID:                item.ID,
		TransactionID:     item.TransactionID,
		TransactionNumber: item.TransactionNumber,
		AssetID:           item.AssetID,
		AssetNumber:       item.AssetNumber,
		FromBranchCode:    item.FromBranchCode,
		ToBranchCode:      item.ToBranchCode,
		FromLocation:      item.FromLocation,
		ToLocation:        item.ToLocation,
		Notes:             item.Notes,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}

	if item.Asset != nil {
		response.AssetName = &item.Asset.AssetName
	}

	return response
}

func mapMutationItemsToResponse(items []models.TransactionMutation) []dto.MutationItemResponse {
	response := make([]dto.MutationItemResponse, len(items))
	for i, item := range items {
		response[i] = mapMutationItemToResponse(item)
	}
	return response
}

// ============================================================================
// DISPOSAL MAPPERS
// FIX: hapus strconv
// ============================================================================

func mapDisposalItemToResponse(item models.TransactionDisposal) dto.DisposalItemResponse {
	response := dto.DisposalItemResponse{
		ID:                item.ID,
		TransactionID:     item.TransactionID,
		TransactionNumber: item.TransactionNumber,
		AssetID:           item.AssetID,
		AssetNumber:       item.AssetNumber,
		DisposalMethod:    item.DisposalMethod,
		DisposalValue:     item.DisposalValue,
		DisposalReason:    item.DisposalReason,
		Notes:             item.Notes,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}

	if item.Asset != nil {
		response.AssetName = &item.Asset.AssetName
	}

	return response
}

func mapDisposalItemsToResponse(items []models.TransactionDisposal) []dto.DisposalItemResponse {
	response := make([]dto.DisposalItemResponse, len(items))
	for i, item := range items {
		response[i] = mapDisposalItemToResponse(item)
	}
	return response
}

// ============================================================================
// STOCK OPNAME MAPPERS
// FIX: hapus strconv. AssetStatus di model adalah string biasa (bukan pointer),
//      tapi di DTO *string — wrap dengan & hanya jika tidak empty
// ============================================================================

func mapStockOpnameItemToResponse(item models.TransactionStockOpname) dto.StockOpnameItemResponse {
	response := dto.StockOpnameItemResponse{
		ID:                item.ID,
		TransactionID:     item.TransactionID,
		TransactionNumber: item.TransactionNumber,
		AssetID:           item.AssetID,
		AssetNumber:       item.AssetNumber,
		PhysicalStatus:    item.PhysicalStatus,
		Condition:         item.Condition,
		Notes:             item.Notes,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}

	// FIX: AssetStatus di model adalah string, di DTO adalah *string
	if item.AssetStatus != "" {
		response.AssetStatus = &item.AssetStatus
	}

	if item.Asset != nil {
		response.AssetName = &item.Asset.AssetName
	}

	return response
}

func mapStockOpnameItemsToResponse(items []models.TransactionStockOpname) []dto.StockOpnameItemResponse {
	response := make([]dto.StockOpnameItemResponse, len(items))
	for i, item := range items {
		response[i] = mapStockOpnameItemToResponse(item)
	}
	return response
}

// ============================================================================
// ASSET CATEGORY MAPPERS
// FIX: hapus strconv
// ============================================================================

func mapAssetCategoryToResponse(category models.AssetCategory) dto.AssetCategoryResponse {
	return dto.AssetCategoryResponse{
		ID:           category.ID,
		CategoryCode: category.CategoryCode,
		CategoryName: category.CategoryName,
		Description:  category.Description,
		IsActive:     category.IsActive,
		CreatedAt:    category.CreatedAt,
		UpdatedAt:    category.UpdatedAt,
	}
}

func mapAssetCategoriesToResponse(categories []models.AssetCategory) []dto.AssetCategoryResponse {
	response := make([]dto.AssetCategoryResponse, len(categories))
	for i, category := range categories {
		response[i] = mapAssetCategoryToResponse(category)
	}
	return response
}

// ============================================================================
// ASSET MASTER MAPPERS
// FIX: hapus strconv, CategoryID sudah *uint
// ============================================================================

func mapAssetToResponse(asset models.Asset) dto.AssetResponse {
	response := dto.AssetResponse{
		ID:            asset.ID,
		AssetNumber:   asset.AssetNumber,
		AssetName:     asset.AssetName,
		Description:   asset.Description,
		Brand:         asset.Brand,
		UnitOfMeasure: asset.UnitOfMeasure,
		UnitQuantity:  asset.UnitQuantity,
		Location:      asset.Location,
		Grouping:      asset.Grouping,
		CategoryID:    asset.CategoryID, // FIX: sudah *uint, langsung assign
		BranchCode:    asset.BranchCode,
		IONumber:      asset.IONumber,
		RecordType:    asset.RecordType,
		AssetStatus:   asset.AssetStatus,
		CreatedAt:     asset.CreatedAt,
		UpdatedAt:     asset.UpdatedAt,
	}

	if asset.Category != nil {
		response.CategoryName = &asset.Category.CategoryName
	}

	return response
}

func mapAssetValueToResponse(value models.AssetValue) dto.AssetValueResponse {
	return dto.AssetValueResponse{
		ID:                      value.ID,
		AssetID:                 value.AssetID,
		EffectiveDate:           value.EffectiveDate,
		BookValue:               value.BookValue,
		AcquisitionValue:        value.AcquisitionValue,
		AccumulatedDepreciation: value.AccumulatedDepreciation,
		Condition:               value.Condition,
		PhysicalStatus:          value.PhysicalStatus,
		AssetStatus:             value.AssetStatus,
		IsActive:                value.IsActive,
		CreatedAt:               value.CreatedAt,
		UpdatedAt:               value.UpdatedAt,
	}
}

func mapAssetValuesToResponse(values []models.AssetValue) []dto.AssetValueResponse {
	response := make([]dto.AssetValueResponse, len(values))
	for i, value := range values {
		response[i] = mapAssetValueToResponse(value)
	}
	return response
}

// ============================================================================
// ASSET HISTORY MAPPERS
// FIX: hapus strconv, TransactionID sudah *uint, hapus Notes & ChangedByUser
// ============================================================================

func mapAssetHistoryToResponse(history models.AssetHistory) dto.AssetHistoryResponse {
	response := dto.AssetHistoryResponse{
		ID:              history.ID,
		AssetID:         history.AssetID,
		TransactionType: history.TransactionType,
		TransactionID:   history.TransactionID, // FIX: sudah *uint
		DocumentNumber:  history.DocumentNumber,
		TransactionDate: history.TransactionDate,
		BeforeData:      history.BeforeData,
		AfterData:       history.AfterData,
		ChangedBy:       history.ChangedBy,
		CreatedAt:       history.CreatedAt,
	}

	if history.Asset != nil {
		response.AssetNumber = &history.Asset.AssetNumber
	}

	return response
}

func mapAssetHistoriesToResponse(histories []models.AssetHistory) []dto.AssetHistoryResponse {
	response := make([]dto.AssetHistoryResponse, len(histories))
	for i, history := range histories {
		response[i] = mapAssetHistoryToResponse(history)
	}
	return response
}

// ============================================================================
// DEPRECIATION MAPPERS
// FIX: hapus strconv, pakai ReferenceID/ReferenceValue, hapus CategoryID/AssetID/ResidualValue
//      MonthlyDepreciation: pakai Period, CalculationDate, field accumulation baru
// ============================================================================

func mapDepreciationSettingToResponse(setting models.DepreciationSetting) dto.DepreciationSettingResponse {
	return dto.DepreciationSettingResponse{
		ID:                 setting.ID,
		SettingType:        setting.SettingType,
		ReferenceID:        setting.ReferenceID,
		ReferenceValue:     setting.ReferenceValue,
		CalculationMethod:  setting.CalculationMethod,
		DepreciationPeriod: setting.DepreciationPeriod,
		UsefulLifeMonths:   setting.UsefulLifeMonths,
		DepreciationRate:   setting.DepreciationRate,
		StartDate:          setting.StartDate,
		EndDate:            setting.EndDate,
		IsActive:           setting.IsActive,
		CreatedAt:          setting.CreatedAt,
		UpdatedAt:          setting.UpdatedAt,
	}
}

func mapDepreciationSettingsToResponse(settings []models.DepreciationSetting) []dto.DepreciationSettingResponse {
	response := make([]dto.DepreciationSettingResponse, len(settings))
	for i, setting := range settings {
		response[i] = mapDepreciationSettingToResponse(setting)
	}
	return response
}

func mapMonthlyDepreciationToResponse(calc models.MonthlyDepreciationCalculation) dto.MonthlyDepreciationResponse {
	response := dto.MonthlyDepreciationResponse{
		ID:                               calc.ID,
		AssetID:                          calc.AssetID,
		Period:                           calc.Period,
		CalculationDate:                  calc.CalculationDate,
		BeginningBookValue:               calc.BeginningBookValue,
		DepreciationAmount:               calc.DepreciationAmount,
		BeginningAccumulatedDepreciation: calc.BeginningAccumulatedDepreciation,
		EndingAccumulatedDepreciation:    calc.EndingAccumulatedDepreciation,
		EndingBookValue:                  calc.EndingBookValue,
		CalculationMethod:                calc.CalculationMethod,
		DepreciationSettingID:            calc.DepreciationSettingID,
		IsLocked:                         calc.IsLocked,
		CreatedAt:                        calc.CreatedAt,
		UpdatedAt:                        calc.UpdatedAt,
	}

	if calc.Asset != nil {
		response.AssetNumber = calc.Asset.AssetNumber
		response.AssetName = calc.Asset.AssetName
	}

	return response
}

func mapMonthlyDepreciationsToResponse(calcs []models.MonthlyDepreciationCalculation) []dto.MonthlyDepreciationResponse {
	response := make([]dto.MonthlyDepreciationResponse, len(calcs))
	for i, calc := range calcs {
		response[i] = mapMonthlyDepreciationToResponse(calc)
	}
	return response
}
