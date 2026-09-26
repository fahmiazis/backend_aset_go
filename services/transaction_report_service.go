package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ============================================================
// REPORT PROCUREMENT / MUTATION / DISPOSAL
//
// Dibatasi per cabang user, sama dengan AssetBranchScope: non-admin hanya
// melihat transaksi cabang yang dia punya di user_branchs (homebase +
// assignment/temporary), admin melihat semua.
//
// "Cabang transaksi" = homebase pembuat saat nomor transaksi dibuat. transactions
// tidak punya kolom cabang, tapi kodenya tertanam di segmen kedua nomor
// (0001/BC000005/BANDUNG BARAT/IX/2026-DPSL) — lihat GenerateTransactionNumber.
// Sengaja tidak memakai homebase pembuat saat ini (GetCreatorBranchCode):
// kalau pembuatnya pindah homebase, transaksinya tidak boleh ikut pindah
// cabang di report.
// ============================================================

var ErrReportBranchForbidden = errors.New("you do not have access to this branch")

const reportBranchExpr = "SUBSTRING_INDEX(SUBSTRING_INDEX(transactions.transaction_number, '/', 2), '/', -1)"

func transactionBranchCode(number string) string {
	parts := strings.Split(number, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

type reportScope struct {
	codes []string // cabang yang boleh dilihat; nil + all = semua
	all   bool
}

// resolveReportScope — gabungan hak cabang user dengan filter branch_code.
func resolveReportScope(viewer AssetViewer, branchCode string) (reportScope, error) {
	codes, all := AssetBranchScope(viewer)
	if branchCode == "" || branchCode == "ALL" {
		return reportScope{codes: codes, all: all}, nil
	}
	if !all && !containsString(codes, branchCode) {
		return reportScope{}, ErrReportBranchForbidden
	}
	return reportScope{codes: []string{branchCode}}, nil
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// baseReportQuery — transaksi satu jenis dalam rentang tanggal + pencarian.
// Filter cabang dan stage dipasang pemanggil karena aturannya beda per jenis.
func baseReportQuery(txType string, filter dto.TransactionReportFilter) *gorm.DB {
	query := config.DB.Model(&models.Transaction{}).
		Where("transactions.transaction_type = ?", txType)
	if filter.StartDate != "" {
		query = query.Where("transactions.transaction_date >= ?", filter.StartDate)
	}
	if filter.EndDate != "" {
		query = query.Where("transactions.transaction_date <= ?", filter.EndDate)
	}
	return query
}

// applyOriginBranchScope — cabang pengaju saja (procurement, disposal).
func applyOriginBranchScope(query *gorm.DB, scope reportScope) *gorm.DB {
	if scope.all {
		return query
	}
	if len(scope.codes) == 0 {
		return query.Where("1 = 0")
	}
	return query.Where(reportBranchExpr+" IN ?", scope.codes)
}

// fetchReportTransactions — menjalankan query, menghitung per stage sebelum
// filter stage (untuk tab), lalu menyaring stage di memori.
func fetchReportTransactions(query *gorm.DB, stage string) ([]models.Transaction, []dto.ReportStageCount, error) {
	var transactions []models.Transaction
	if err := query.
		Order("transactions.transaction_date DESC, transactions.id DESC").
		Find(&transactions).Error; err != nil {
		return nil, nil, err
	}

	counts := map[string]int{}
	order := []string{}
	filtered := make([]models.Transaction, 0, len(transactions))
	for _, t := range transactions {
		if _, ok := counts[t.CurrentStage]; !ok {
			order = append(order, t.CurrentStage)
		}
		counts[t.CurrentStage]++
		if stage == "" || t.CurrentStage == stage {
			filtered = append(filtered, t)
		}
	}
	sort.Strings(order)
	byStage := make([]dto.ReportStageCount, 0, len(order))
	for _, s := range order {
		byStage = append(byStage, dto.ReportStageCount{Stage: s, Count: counts[s]})
	}
	return filtered, byStage, nil
}

func reportBranchNames() map[string]string {
	var branches []models.Branch
	config.DB.Select("branch_code", "branch_name").Find(&branches)
	names := make(map[string]string, len(branches))
	for _, b := range branches {
		names[b.BranchCode] = b.BranchName
	}
	return names
}

func reportCategoryNames() map[uint]string {
	var categories []models.AssetCategory
	config.DB.Select("id", "category_name").Find(&categories)
	names := make(map[uint]string, len(categories))
	for _, c := range categories {
		names[c.ID] = c.CategoryName
	}
	return names
}

func reportAssets(ids []uint) map[uint]models.Asset {
	result := map[uint]models.Asset{}
	if len(ids) == 0 {
		return result
	}
	var assets []models.Asset
	config.DB.Select("id", "asset_number", "asset_name", "category_id").Where("id IN ?", ids).Find(&assets)
	for _, a := range assets {
		result[a.ID] = a
	}
	return result
}

func transactionIDs(transactions []models.Transaction) []uint {
	ids := make([]uint, len(transactions))
	for i, t := range transactions {
		ids[i] = t.ID
	}
	return ids
}

func reportTransactionInfo(t models.Transaction, branchNames map[string]string, creators map[string]string) dto.ReportTransactionInfo {
	code := transactionBranchCode(t.TransactionNumber)
	return dto.ReportTransactionInfo{
		TransactionNumber: t.TransactionNumber,
		TransactionDate:   t.TransactionDate.Format("2006-01-02"),
		BranchCode:        code,
		BranchName:        branchNames[code],
		CurrentStage:      t.CurrentStage,
		Status:            t.Status,
		Notes:             t.Notes,
		CreatedByName:     creators[t.CreatedBy],
	}
}

func reportCreators(transactions []models.Transaction) map[string]string {
	ids := make([]string, len(transactions))
	for i, t := range transactions {
		ids[i] = t.CreatedBy
	}
	return resolveUserFullnames(ids)
}

// GetReportBranchOptions — isi dropdown cabang report.
func GetReportBranchOptions(viewer AssetViewer) ([]dto.BranchOption, bool, error) {
	_, all := AssetBranchScope(viewer)
	branches, err := GetViewableAssetBranches(viewer)
	return branches, all, err
}

func searchLike(search string) string {
	return "%" + strings.TrimSpace(search) + "%"
}

// ============================================================
// EXPORT EXCEL
// Sheet "Summary" (filter + total + jumlah per stage) dan "Detail" (baris
// yang sama persis dengan tabel di layar).
// ============================================================

type reportColumn struct {
	header string
	width  float64
	money  bool
}

type reportExport struct {
	title   string
	columns []reportColumn
	rows    [][]interface{}
	totals  [][2]interface{} // label, nilai — ditulis di sheet Summary
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func floatOrNil(f *float64) interface{} {
	if f == nil {
		return nil
	}
	return *f
}

func reportFilterLines(filter dto.TransactionReportFilter, branchNames map[string]string) [][2]interface{} {
	period := "Semua tanggal"
	if filter.StartDate != "" || filter.EndDate != "" {
		period = fmt.Sprintf("%s s/d %s", orDash(filter.StartDate), orDash(filter.EndDate))
	}
	branch := "Semua cabang yang dapat diakses"
	if filter.BranchCode != "" && filter.BranchCode != "ALL" {
		branch = filter.BranchCode
		if name := branchNames[filter.BranchCode]; name != "" {
			branch += " - " + name
		}
	}
	stage := "Semua"
	if filter.Stage != "" {
		stage = filter.Stage
	}
	lines := [][2]interface{}{
		{"Periode", period},
		{"Cabang", branch},
		{"Stage", stage},
	}
	if strings.TrimSpace(filter.Search) != "" {
		lines = append(lines, [2]interface{}{"Pencarian", filter.Search})
	}
	return append(lines, [2]interface{}{"Dibuat", time.Now().Format("2006-01-02 15:04")})
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func buildReportWorkbook(exp reportExport, filter dto.TransactionReportFilter, summary dto.TransactionReportSummary) (*excelize.File, error) {
	f := excelize.NewFile()
	branchNames := reportBranchNames()

	titleStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	boldStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	moneyStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 3}) // #,##0
	moneyBoldStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 3, Font: &excelize.Font{Bold: true}})
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"4F46E5"}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "D1D5DB", Style: 1}, {Type: "right", Color: "D1D5DB", Style: 1},
			{Type: "top", Color: "D1D5DB", Style: 1}, {Type: "bottom", Color: "D1D5DB", Style: 1},
		},
	})

	// ---------- Summary ----------
	summarySheet := "Summary"
	f.SetSheetName("Sheet1", summarySheet)
	f.SetCellValue(summarySheet, "A1", exp.title)
	f.SetCellStyle(summarySheet, "A1", "A1", titleStyle)

	row := 3
	for _, line := range reportFilterLines(filter, branchNames) {
		f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), line[0])
		f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), line[1])
		f.SetCellStyle(summarySheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
		row++
	}

	row++
	for _, total := range exp.totals {
		f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), total[0])
		f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), total[1])
		f.SetCellStyle(summarySheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
		if _, isFloat := total[1].(float64); isFloat {
			f.SetCellStyle(summarySheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), moneyBoldStyle)
		}
		row++
	}

	row++
	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), "Stage")
	f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), "Jumlah Transaksi")
	f.SetCellStyle(summarySheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), headerStyle)
	row++
	for _, s := range summary.ByStage {
		f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), s.Stage)
		f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), s.Count)
		row++
	}
	f.SetColWidth(summarySheet, "A", "A", 28)
	f.SetColWidth(summarySheet, "B", "B", 45)

	// ---------- Detail ----------
	detailSheet := "Detail"
	if _, err := f.NewSheet(detailSheet); err != nil {
		return nil, err
	}
	for i, col := range exp.columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(detailSheet, cell, col.header)
		f.SetCellStyle(detailSheet, cell, cell, headerStyle)
		name, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(detailSheet, name, name, col.width)
	}
	for r, values := range exp.rows {
		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, r+2)
			f.SetCellValue(detailSheet, cell, v)
			if exp.columns[i].money {
				f.SetCellStyle(detailSheet, cell, cell, moneyStyle)
			}
		}
	}
	lastCol, _ := excelize.ColumnNumberToName(len(exp.columns))
	lastRow := len(exp.rows) + 1
	if len(exp.rows) > 0 {
		f.AutoFilter(detailSheet, fmt.Sprintf("A1:%s%d", lastCol, lastRow), nil)
	}
	f.SetPanes(detailSheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	f.SetActiveSheet(0)
	return f, nil
}

func reportFilename(prefix string, filter dto.TransactionReportFilter) string {
	name := prefix
	if filter.StartDate != "" {
		name += "_" + strings.ReplaceAll(filter.StartDate, "-", "")
	}
	if filter.EndDate != "" {
		name += "_" + strings.ReplaceAll(filter.EndDate, "-", "")
	}
	if filter.BranchCode != "" && filter.BranchCode != "ALL" {
		name += "_" + filter.BranchCode
	}
	return name + ".xlsx"
}

var reportInfoColumns = []reportColumn{
	{"No. Transaksi", 42, false},
	{"Tanggal", 12, false},
	{"Kode Cabang", 12, false},
	{"Nama Cabang", 22, false},
	{"Stage", 20, false},
	{"Status", 12, false},
	{"Dibuat Oleh", 22, false},
}

func reportInfoValues(info dto.ReportTransactionInfo) []interface{} {
	return []interface{}{
		info.TransactionNumber, info.TransactionDate, info.BranchCode, info.BranchName,
		info.CurrentStage, info.Status, info.CreatedByName,
	}
}

func branchLabelWithName(code, name string) string {
	if name == "" {
		return code
	}
	return code + " - " + name
}
