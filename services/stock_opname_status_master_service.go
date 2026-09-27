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
// ditambah lewat data, bukan lewat kode. Status & kondisi cuma CREATE + LIST
// + SOFT DELETE (tidak ada UPDATE) — lihat komentar di
// dto/stock_opname_status_master_dto.go. Relasi status fisik -> kondisi yang
// boleh dipilih (stock_opname_physical_condition_rules) boleh diubah.
// ============================================================

var stockOpnameCodePattern = regexp.MustCompile(`^[A-Z0-9_]+$`)

func normalizeStockOpnameStatusCode(code string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	if !stockOpnameCodePattern.MatchString(normalized) {
		return "", errors.New("code hanya boleh huruf kapital, angka, dan underscore (contoh: RUSAK_PARAH)")
	}
	return normalized, nil
}

// ---------------- RELASI STATUS FISIK -> KONDISI ----------------

type stockOpnameConditionRuleRow struct {
	PhysicalStatusID   uint
	PhysicalStatusCode string
	ConditionID        uint
	ConditionCode      string
	ConditionLabel     string
}

// loadStockOpnameConditionRuleRows: semua relasi yang status fisik &
// kondisinya masih aktif (belum di-soft-delete), urut id kondisi.
func loadStockOpnameConditionRuleRows() ([]stockOpnameConditionRuleRow, error) {
	var rows []stockOpnameConditionRuleRow
	if err := config.DB.Raw(`
		SELECT p.id AS physical_status_id, p.code AS physical_status_code,
		       c.id AS condition_id, c.code AS condition_code, c.label AS condition_label
		FROM stock_opname_physical_condition_rules r
		JOIN stock_opname_physical_status_masters p ON p.id = r.physical_status_id AND p.deleted_at IS NULL
		JOIN stock_opname_condition_masters c ON c.id = r.condition_id AND c.deleted_at IS NULL
		ORDER BY c.id ASC
	`).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load physical status/condition rules: %w", err)
	}
	return rows, nil
}

// loadStockOpnameConditionRules: kode status fisik -> set kode kondisi yang
// boleh dipilih. Dipakai validateFindingPhysicalConditionPair.
func loadStockOpnameConditionRules() (map[string]map[string]bool, error) {
	rows, err := loadStockOpnameConditionRuleRows()
	if err != nil {
		return nil, err
	}
	rules := make(map[string]map[string]bool)
	for _, r := range rows {
		if rules[r.PhysicalStatusCode] == nil {
			rules[r.PhysicalStatusCode] = make(map[string]bool)
		}
		rules[r.PhysicalStatusCode][r.ConditionCode] = true
	}
	return rules, nil
}

// resolveConditionIDs: dedupe + pastikan semua id kondisi ada & aktif.
func resolveConditionIDs(tx *gorm.DB, ids []uint) ([]uint, error) {
	unique := dedupeUintIDs(ids)
	if len(unique) == 0 {
		return nil, errors.New("pilih minimal 1 kondisi")
	}
	var count int64
	if err := tx.Model(&models.StockOpnameConditionMaster{}).Where("id IN ?", unique).Count(&count).Error; err != nil {
		return nil, err
	}
	if int(count) != len(unique) {
		return nil, errors.New("ada kondisi yang tidak ditemukan atau sudah dihapus")
	}
	return unique, nil
}

func resolvePhysicalStatusIDs(tx *gorm.DB, ids []uint) ([]uint, error) {
	unique := dedupeUintIDs(ids)
	if len(unique) == 0 {
		return unique, nil
	}
	var count int64
	if err := tx.Model(&models.StockOpnamePhysicalStatusMaster{}).Where("id IN ?", unique).Count(&count).Error; err != nil {
		return nil, err
	}
	if int(count) != len(unique) {
		return nil, errors.New("ada status fisik yang tidak ditemukan atau sudah dihapus")
	}
	return unique, nil
}

func dedupeUintIDs(ids []uint) []uint {
	seen := make(map[uint]bool, len(ids))
	unique := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id != 0 && !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	return unique
}

func replacePhysicalStatusConditionRules(tx *gorm.DB, physicalStatusID uint, conditionIDs []uint) error {
	if err := tx.Where("physical_status_id = ?", physicalStatusID).
		Delete(&models.StockOpnamePhysicalConditionRule{}).Error; err != nil {
		return err
	}
	rules := make([]models.StockOpnamePhysicalConditionRule, len(conditionIDs))
	for i, id := range conditionIDs {
		rules[i] = models.StockOpnamePhysicalConditionRule{PhysicalStatusID: physicalStatusID, ConditionID: id}
	}
	return tx.Create(&rules).Error
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

func mapPhysicalStatusMasterToResponse(m models.StockOpnamePhysicalStatusMaster, allowed []dto.StockOpnameConditionRef) dto.StockOpnamePhysicalStatusMasterResponse {
	if allowed == nil {
		allowed = []dto.StockOpnameConditionRef{}
	}
	return dto.StockOpnamePhysicalStatusMasterResponse{
		ID:                     m.ID,
		Code:                   m.Code,
		Label:                  m.Label,
		RequiresBorrowDocument: m.RequiresBorrowDocument,
		CountsAsMissing:        m.CountsAsMissing,
		IsSystem:               m.IsSystem,
		CreatedAt:              m.CreatedAt,
		AllowedConditions:      allowed,
	}
}

func loadAllowedConditionsByPhysicalStatusID() (map[uint][]dto.StockOpnameConditionRef, error) {
	rows, err := loadStockOpnameConditionRuleRows()
	if err != nil {
		return nil, err
	}
	allowed := make(map[uint][]dto.StockOpnameConditionRef)
	for _, r := range rows {
		allowed[r.PhysicalStatusID] = append(allowed[r.PhysicalStatusID], dto.StockOpnameConditionRef{
			ID: r.ConditionID, Code: r.ConditionCode, Label: r.ConditionLabel,
		})
	}
	return allowed, nil
}

func getStockOpnamePhysicalStatusMasterResponse(id uint) (*dto.StockOpnamePhysicalStatusMasterResponse, error) {
	var row models.StockOpnamePhysicalStatusMaster
	if err := config.DB.First(&row, id).Error; err != nil {
		return nil, err
	}
	allowed, err := loadAllowedConditionsByPhysicalStatusID()
	if err != nil {
		return nil, err
	}
	response := mapPhysicalStatusMasterToResponse(row, allowed[row.ID])
	return &response, nil
}

func GetAllStockOpnamePhysicalStatusMasters() ([]dto.StockOpnamePhysicalStatusMasterResponse, error) {
	var rows []models.StockOpnamePhysicalStatusMaster
	if err := config.DB.Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	allowed, err := loadAllowedConditionsByPhysicalStatusID()
	if err != nil {
		return nil, err
	}
	result := make([]dto.StockOpnamePhysicalStatusMasterResponse, len(rows))
	for i, r := range rows {
		result[i] = mapPhysicalStatusMasterToResponse(r, allowed[r.ID])
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
		Code:                   code,
		Label:                  strings.TrimSpace(req.Label),
		RequiresBorrowDocument: req.RequiresBorrowDocument,
		CountsAsMissing:        req.CountsAsMissing,
		IsSystem:               false,
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		conditionIDs, err := resolveConditionIDs(tx, req.ConditionIDs)
		if err != nil {
			return err
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return replacePhysicalStatusConditionRules(tx, row.ID, conditionIDs)
	}); err != nil {
		return nil, err
	}

	return getStockOpnamePhysicalStatusMasterResponse(row.ID)
}

// UpdateStockOpnamePhysicalStatusConditions mengganti daftar kondisi yang
// boleh dipilih buat satu status fisik (termasuk status bawaan sistem).
// Temuan yang udah kesimpen gak diubah — aturan baru berlaku saat temuan
// disimpan/diupload berikutnya.
func UpdateStockOpnamePhysicalStatusConditions(id string, req dto.UpdateStockOpnamePhysicalStatusConditionsRequest) (*dto.StockOpnamePhysicalStatusMasterResponse, error) {
	var row models.StockOpnamePhysicalStatusMaster
	if err := config.DB.First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("status fisik tidak ditemukan")
		}
		return nil, err
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		conditionIDs, err := resolveConditionIDs(tx, req.ConditionIDs)
		if err != nil {
			return err
		}
		return replacePhysicalStatusConditionRules(tx, row.ID, conditionIDs)
	}); err != nil {
		return nil, err
	}

	return getStockOpnamePhysicalStatusMasterResponse(row.ID)
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
		ID:           m.ID,
		Code:         m.Code,
		Label:        m.Label,
		ReportBucket: m.ReportBucket,
		IsSystem:     m.IsSystem,
		CreatedAt:    m.CreatedAt,
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
		Code:         code,
		Label:        strings.TrimSpace(req.Label),
		ReportBucket: strings.ToUpper(strings.TrimSpace(req.ReportBucket)),
		IsSystem:     false,
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		physicalStatusIDs, err := resolvePhysicalStatusIDs(tx, req.PhysicalStatusIDs)
		if err != nil {
			return err
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if len(physicalStatusIDs) == 0 {
			return nil
		}
		rules := make([]models.StockOpnamePhysicalConditionRule, len(physicalStatusIDs))
		for i, id := range physicalStatusIDs {
			rules[i] = models.StockOpnamePhysicalConditionRule{PhysicalStatusID: id, ConditionID: row.ID}
		}
		return tx.Create(&rules).Error
	}); err != nil {
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

	// Jangan sampai ada status fisik yang kehilangan semua kondisinya —
	// temuan dengan status itu jadi gak bisa disimpan sama sekali.
	rules, err := loadStockOpnameConditionRuleRows()
	if err != nil {
		return err
	}
	conditionCount := make(map[string]int)
	usedBy := make(map[string]bool)
	for _, r := range rules {
		conditionCount[r.PhysicalStatusCode]++
		if r.ConditionID == row.ID {
			usedBy[r.PhysicalStatusCode] = true
		}
	}
	var orphaned []string
	for code := range usedBy {
		if conditionCount[code] == 1 {
			orphaned = append(orphaned, code)
		}
	}
	if len(orphaned) > 0 {
		return fmt.Errorf("kondisi ini satu-satunya kondisi untuk status fisik %s — atur kondisi lain untuk status itu dulu", strings.Join(orphaned, ", "))
	}

	return config.DB.Delete(&row).Error
}
