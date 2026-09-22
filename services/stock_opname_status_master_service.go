package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// ============================================================
// MASTER DATA STATUS FISIK & KONDISI STOCK OPNAME
//
// Menggantikan map hardcode (stockOpnamePhysicalStatusLabel/Value dkk di
// services/stock_opname_flow_service.go) supaya status/kondisi baru bisa
// ditambah lewat data, bukan lewat kode. Cuma CREATE + LIST + SOFT DELETE
// (tidak ada UPDATE) — lihat komentar di dto/stock_opname_status_master_dto.go.
// ============================================================

var stockOpnameCodePattern = regexp.MustCompile(`^[A-Z0-9_]+$`)

func normalizeStockOpnameStatusCode(code string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	if !stockOpnameCodePattern.MatchString(normalized) {
		return "", errors.New("code hanya boleh huruf kapital, angka, dan underscore (contoh: RUSAK_PARAH)")
	}
	return normalized, nil
}

// ---------------- PHYSICAL STATUS ----------------

func loadStockOpnamePhysicalStatusMap() (map[string]models.StockOpnamePhysicalStatusMaster, error) {
	var rows []models.StockOpnamePhysicalStatusMaster
	if err := config.DB.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load physical status masters: %w", err)
	}
	m := make(map[string]models.StockOpnamePhysicalStatusMaster, len(rows))
	for _, r := range rows {
		m[r.Code] = r
	}
	return m, nil
}

func mapPhysicalStatusMasterToResponse(m models.StockOpnamePhysicalStatusMaster) dto.StockOpnamePhysicalStatusMasterResponse {
	return dto.StockOpnamePhysicalStatusMasterResponse{
		ID:                             m.ID,
		Code:                           m.Code,
		Label:                          m.Label,
		RequiresBorrowDocument:         m.RequiresBorrowDocument,
		RequiresNotApplicableCondition: m.RequiresNotApplicableCondition,
		CountsAsMissing:                m.CountsAsMissing,
		IsSystem:                       m.IsSystem,
		CreatedAt:                      m.CreatedAt,
	}
}

func GetAllStockOpnamePhysicalStatusMasters() ([]dto.StockOpnamePhysicalStatusMasterResponse, error) {
	var rows []models.StockOpnamePhysicalStatusMaster
	if err := config.DB.Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]dto.StockOpnamePhysicalStatusMasterResponse, len(rows))
	for i, r := range rows {
		result[i] = mapPhysicalStatusMasterToResponse(r)
	}
	return result, nil
}

func CreateStockOpnamePhysicalStatusMaster(req dto.CreateStockOpnamePhysicalStatusMasterRequest) (*dto.StockOpnamePhysicalStatusMasterResponse, error) {
	code, err := normalizeStockOpnameStatusCode(req.Code)
	if err != nil {
		return nil, err
	}

	var existing models.StockOpnamePhysicalStatusMaster
	if err := config.DB.Where("code = ?", code).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("kode status fisik %q sudah dipakai", code)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	row := models.StockOpnamePhysicalStatusMaster{
		Code:                           code,
		Label:                          strings.TrimSpace(req.Label),
		RequiresBorrowDocument:         req.RequiresBorrowDocument,
		RequiresNotApplicableCondition: req.RequiresNotApplicableCondition,
		CountsAsMissing:                req.CountsAsMissing,
		IsSystem:                       false,
	}
	if err := config.DB.Create(&row).Error; err != nil {
		return nil, err
	}

	response := mapPhysicalStatusMasterToResponse(row)
	return &response, nil
}

func DeleteStockOpnamePhysicalStatusMaster(id string) error {
	var row models.StockOpnamePhysicalStatusMaster
	if err := config.DB.First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("status fisik tidak ditemukan")
		}
		return err
	}
	if row.IsSystem {
		return errors.New("status fisik bawaan sistem tidak bisa dihapus")
	}
	return config.DB.Delete(&row).Error
}

// ---------------- CONDITION ----------------

func loadStockOpnameConditionMap() (map[string]models.StockOpnameConditionMaster, error) {
	var rows []models.StockOpnameConditionMaster
	if err := config.DB.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load condition masters: %w", err)
	}
	m := make(map[string]models.StockOpnameConditionMaster, len(rows))
	for _, r := range rows {
		m[r.Code] = r
	}
	return m, nil
}

func mapConditionMasterToResponse(m models.StockOpnameConditionMaster) dto.StockOpnameConditionMasterResponse {
	return dto.StockOpnameConditionMasterResponse{
		ID:                   m.ID,
		Code:                 m.Code,
		Label:                m.Label,
		IsNotApplicableValue: m.IsNotApplicableValue,
		ReportBucket:         m.ReportBucket,
		IsSystem:             m.IsSystem,
		CreatedAt:            m.CreatedAt,
	}
}

func GetAllStockOpnameConditionMasters() ([]dto.StockOpnameConditionMasterResponse, error) {
	var rows []models.StockOpnameConditionMaster
	if err := config.DB.Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]dto.StockOpnameConditionMasterResponse, len(rows))
	for i, r := range rows {
		result[i] = mapConditionMasterToResponse(r)
	}
	return result, nil
}

func CreateStockOpnameConditionMaster(req dto.CreateStockOpnameConditionMasterRequest) (*dto.StockOpnameConditionMasterResponse, error) {
	code, err := normalizeStockOpnameStatusCode(req.Code)
	if err != nil {
		return nil, err
	}

	var existing models.StockOpnameConditionMaster
	if err := config.DB.Where("code = ?", code).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("kode kondisi %q sudah dipakai", code)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	row := models.StockOpnameConditionMaster{
		Code:                 code,
		Label:                strings.TrimSpace(req.Label),
		IsNotApplicableValue: req.IsNotApplicableValue,
		ReportBucket:         strings.ToUpper(strings.TrimSpace(req.ReportBucket)),
		IsSystem:             false,
	}
	if err := config.DB.Create(&row).Error; err != nil {
		return nil, err
	}

	response := mapConditionMasterToResponse(row)
	return &response, nil
}

func DeleteStockOpnameConditionMaster(id string) error {
	var row models.StockOpnameConditionMaster
	if err := config.DB.First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("kondisi tidak ditemukan")
		}
		return err
	}
	if row.IsSystem {
		return errors.New("kondisi bawaan sistem tidak bisa dihapus")
	}
	return config.DB.Delete(&row).Error
}
