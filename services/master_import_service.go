package services

import (
	"backend-go/dto"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ============================================================================
// Bagian bersama upload master data (branch_import_service.go,
// user_import_service.go, asset_import_service.go).
//
// Format file: sheet "Data" (atau sheet pertama), baris 1 = header kolom,
// data mulai baris 2. Nama header dicocokkan tanpa peduli huruf besar/kecil,
// spasi, dan tanda "*" penanda wajib. Kolom yang tidak dikenal diabaikan.
// ============================================================================

const (
	importDataSheet  = "Data"
	importGuideSheet = "Petunjuk"
	importMaxRows    = 2000
)

// ErrImportFile — file tidak bisa dibaca / formatnya salah (400, bukan error baris)
var ErrImportFile = errors.New("invalid import file")

type importColumn struct {
	Key      string
	Required bool
	Note     string
	Example  string
}

type importRow struct {
	Line   int
	Values map[string]string
}

// importRefList — daftar referensi di sheet Petunjuk (mis. kode cabang yang valid)
type importRefList struct {
	Title   string
	Headers []string
	Rows    [][]string
}

func importFileError(format string, args ...interface{}) error {
	return fmt.Errorf("%w: %s", ErrImportFile, fmt.Sprintf(format, args...))
}

func normalizeImportHeader(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.TrimRight(h, "* ")
	h = strings.Join(strings.Fields(h), "_")
	return h
}

func importColumnKeys(columns []importColumn) []string {
	keys := make([]string, len(columns))
	for i, col := range columns {
		keys[i] = col.Key
	}
	return keys
}

// readImportRows membaca sheet data. Baris yang seluruh kolomnya kosong dilewati.
func readImportRows(r io.Reader, columns []importColumn) ([]importRow, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, importFileError("file must be an .xlsx workbook")
	}
	defer f.Close()

	sheet := importDataSheet
	if idx, _ := f.GetSheetIndex(sheet); idx < 0 {
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, importFileError("workbook has no sheet")
		}
		sheet = sheets[0]
	}

	// RawCellValue: angka tidak ikut format tampilan (mis. "1,500,000"),
	// tanggal berformat date terbaca sebagai serial Excel.
	raw, err := f.GetRows(sheet, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, importFileError("failed to read sheet %s", sheet)
	}
	if len(raw) == 0 {
		return nil, importFileError("sheet %s is empty", sheet)
	}

	known := make(map[string]bool, len(columns))
	for _, col := range columns {
		known[col.Key] = true
	}

	colIndex := make(map[string]int)
	for i, h := range raw[0] {
		key := normalizeImportHeader(h)
		if known[key] {
			if _, dup := colIndex[key]; dup {
				return nil, importFileError("column %s appears more than once", key)
			}
			colIndex[key] = i
		}
	}

	var missing []string
	for _, col := range columns {
		if _, ok := colIndex[col.Key]; !ok && col.Required {
			missing = append(missing, col.Key)
		}
	}
	if len(missing) > 0 {
		return nil, importFileError("missing column(s): %s — download the template for this mode", strings.Join(missing, ", "))
	}

	var rows []importRow
	for i := 1; i < len(raw); i++ {
		values := make(map[string]string, len(columns))
		empty := true
		for _, col := range columns {
			idx, ok := colIndex[col.Key]
			v := ""
			if ok && idx < len(raw[i]) {
				v = strings.TrimSpace(raw[i][idx])
			}
			if v != "" {
				empty = false
			}
			values[col.Key] = v
		}
		if empty {
			continue
		}
		rows = append(rows, importRow{Line: i + 1, Values: values})
	}

	if len(rows) == 0 {
		return nil, importFileError("no data rows found in sheet %s", sheet)
	}
	if len(rows) > importMaxRows {
		return nil, importFileError("maximum %d rows per upload, file has %d", importMaxRows, len(rows))
	}
	return rows, nil
}

// buildImportTemplate — sheet Data (header saja, semua kolom berformat teks
// supaya kode seperti 0012 tidak berubah jadi angka) + sheet Petunjuk.
func buildImportTemplate(title string, columns []importColumn, notes []string, refs []importRefList) (*excelize.File, error) {
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", importDataSheet); err != nil {
		return nil, err
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"4F46E5"}},
	})
	requiredStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"DC2626"}},
	})
	textStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 49}) // "@" = teks
	boldStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	titleStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})

	for i, col := range columns {
		name, _ := excelize.ColumnNumberToName(i + 1)
		header := col.Key
		style := headerStyle
		if col.Required {
			header += "*"
			style = requiredStyle
		}
		f.SetCellValue(importDataSheet, name+"1", header)
		f.SetCellStyle(importDataSheet, name+"1", name+"1", style)
		f.SetColWidth(importDataSheet, name, name, 22)
		f.SetCellStyle(importDataSheet, name+"2", fmt.Sprintf("%s%d", name, importMaxRows+1), textStyle)
	}
	f.SetPanes(importDataSheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	// ---------- Petunjuk ----------
	if _, err := f.NewSheet(importGuideSheet); err != nil {
		return nil, err
	}
	g := importGuideSheet
	f.SetCellValue(g, "A1", title)
	f.SetCellStyle(g, "A1", "A1", titleStyle)
	f.SetColWidth(g, "A", "A", 28)
	f.SetColWidth(g, "B", "B", 12)
	f.SetColWidth(g, "C", "C", 70)
	f.SetColWidth(g, "D", "D", 24)

	row := 3
	for _, note := range notes {
		f.SetCellValue(g, fmt.Sprintf("A%d", row), "• "+note)
		row++
	}

	row++
	for i, h := range []string{"Kolom", "Wajib", "Keterangan", "Contoh"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		f.SetCellValue(g, cell, h)
		f.SetCellStyle(g, cell, cell, headerStyle)
	}
	row++
	for _, col := range columns {
		required := ""
		if col.Required {
			required = "Ya"
		}
		f.SetCellValue(g, fmt.Sprintf("A%d", row), col.Key)
		f.SetCellValue(g, fmt.Sprintf("B%d", row), required)
		f.SetCellValue(g, fmt.Sprintf("C%d", row), col.Note)
		f.SetCellValue(g, fmt.Sprintf("D%d", row), col.Example)
		row++
	}

	for _, ref := range refs {
		row += 2
		f.SetCellValue(g, fmt.Sprintf("A%d", row), ref.Title)
		f.SetCellStyle(g, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
		row++
		for i, h := range ref.Headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellValue(g, cell, h)
			f.SetCellStyle(g, cell, cell, headerStyle)
		}
		row++
		for _, r := range ref.Rows {
			for i, v := range r {
				cell, _ := excelize.CoordinatesToCellName(i+1, row)
				f.SetCellValue(g, cell, v)
			}
			row++
		}
	}

	f.SetActiveSheet(0)
	return f, nil
}

// fillImportRows mengisi sheet Data mulai baris 2 — dipakai unduhan data
// (format sama dengan template mass update, bisa langsung diunggah ulang).
func fillImportRows(f *excelize.File, rows [][]string) {
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			f.SetCellStr(importDataSheet, cell, v)
		}
	}
}

// ---------------------------------------------------------------------------
// Kesalahan baris
// ---------------------------------------------------------------------------

func importErr(code, field, message string, params ...string) dto.ImportRowError {
	e := dto.ImportRowError{Code: code, Field: field, Message: message}
	if len(params) > 0 {
		e.Params = make(map[string]string, len(params)/2)
		for i := 0; i+1 < len(params); i += 2 {
			e.Params[params[i]] = params[i+1]
		}
	}
	return e
}

func errRequired(field string) dto.ImportRowError {
	return importErr("required", field, field+" is required")
}

func errDuplicateInFile(field, value string, line int) dto.ImportRowError {
	return importErr("duplicate_in_file", field,
		fmt.Sprintf("%s %q is duplicated in row %d", field, value, line),
		"value", value, "row", strconv.Itoa(line))
}

func errAlreadyExists(field, value string) dto.ImportRowError {
	return importErr("already_exists", field, fmt.Sprintf("%s %q is already registered", field, value), "value", value)
}

func errNotFound(field, value string) dto.ImportRowError {
	return importErr("not_found", field, fmt.Sprintf("%s %q is not registered", field, value), "value", value)
}

func errInactive(field, value string) dto.ImportRowError {
	return importErr("inactive", field, fmt.Sprintf("%s %q is inactive", field, value), "value", value)
}

func errInvalidOption(field, value string, options []string) dto.ImportRowError {
	return importErr("invalid_option", field,
		fmt.Sprintf("%s %q must be one of: %s", field, value, strings.Join(options, ", ")),
		"value", value, "options", strings.Join(options, ", "))
}

func errLength(field string, min, max int) dto.ImportRowError {
	return importErr("length", field, fmt.Sprintf("%s must be %d-%d characters", field, min, max),
		"min", strconv.Itoa(min), "max", strconv.Itoa(max))
}

// ---------------------------------------------------------------------------
// Parser nilai
// ---------------------------------------------------------------------------

var (
	idThousands = regexp.MustCompile(`^\d{1,3}(\.\d{3})+(,\d+)?$`) // 1.500.000,50
	enThousands = regexp.MustCompile(`^\d{1,3}(,\d{3})+(\.\d+)?$`) // 1,500,000.50
)

// parseImportNumber menerima angka mentah Excel maupun teks berformat
// Indonesia (1.500.000) atau Inggris (1,500,000). Awalan "Rp" dibuang.
func parseImportNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "Rp"), "rp")
	s = strings.ReplaceAll(strings.TrimSpace(s), " ", "")
	if s == "" {
		return 0, false
	}
	switch {
	case idThousands.MatchString(s):
		s = strings.ReplaceAll(strings.ReplaceAll(s, ".", ""), ",", ".")
	case enThousands.MatchString(s):
		s = strings.ReplaceAll(s, ",", "")
	default:
		s = strings.ReplaceAll(s, ",", ".")
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

var importDateLayouts = []string{"2006-01-02", "02/01/2006", "2/1/2006", "02-01-2006", "2-1-2006", "2006/01/02"}

// parseImportDate menerima teks tanggal (YYYY-MM-DD / DD/MM/YYYY) atau serial
// tanggal Excel (sel berformat date).
func parseImportDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, layout := range importDateLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	if serial, err := strconv.ParseFloat(s, 64); err == nil && serial > 0 && serial < 2958466 {
		if t, err := excelize.ExcelDateToTime(serial, false); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), true
		}
	}
	return time.Time{}, false
}

func parseImportStatus(field, value string) (string, *dto.ImportRowError) {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "active" || v == "inactive" {
		return v, nil
	}
	e := errInvalidOption(field, value, []string{"active", "inactive"})
	return "", &e
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// finishImportResult mengisi hitungan ringkasan dari baris-baris hasil validasi.
func finishImportResult(entity, mode string, dryRun bool, columns []importColumn, rows []dto.ImportRowResult) *dto.ImportResult {
	res := &dto.ImportResult{
		Entity:    entity,
		Mode:      mode,
		DryRun:    dryRun,
		Columns:   importColumnKeys(columns),
		TotalRows: len(rows),
		Rows:      rows,
	}
	for i := range rows {
		if rows[i].Errors == nil {
			rows[i].Errors = []dto.ImportRowError{}
		}
		if len(rows[i].Errors) > 0 {
			res.ErrorRows++
		} else {
			res.ValidRows++
		}
	}
	return res
}

// newImportRowResults — kerangka hasil per baris, nilai disalin dari file.
// Kolom yang disebut di hidden tidak dikembalikan (mis. password).
func newImportRowResults(rows []importRow, hidden ...string) []dto.ImportRowResult {
	results := make([]dto.ImportRowResult, len(rows))
	for i, r := range rows {
		values := make(map[string]string, len(r.Values))
		for k, v := range r.Values {
			values[k] = v
		}
		for _, h := range hidden {
			if values[h] != "" {
				values[h] = "******"
			}
		}
		results[i] = dto.ImportRowResult{Row: r.Line, Values: values}
	}
	return results
}

func validateImportMode(mode string) error {
	if mode != dto.ImportModeNew && mode != dto.ImportModeUpdate {
		return importFileError("mode must be %q or %q", dto.ImportModeNew, dto.ImportModeUpdate)
	}
	return nil
}

// trackDuplicate mencatat nilai kunci per baris; mengembalikan baris pertama
// yang sudah memakai nilai itu (0 kalau belum ada). Pembanding case-insensitive.
func trackDuplicate(seen map[string]int, value string, line int) int {
	key := strings.ToLower(value)
	if first, ok := seen[key]; ok {
		return first
	}
	seen[key] = line
	return 0
}

// ImportSummaryMessage — pesan response controller upload
func ImportSummaryMessage(res *dto.ImportResult) string {
	if res.DryRun {
		return strconv.Itoa(res.ValidRows) + " valid, " + strconv.Itoa(res.ErrorRows) + " invalid"
	}
	if res.ErrorRows > 0 {
		return "nothing saved — " + strconv.Itoa(res.ErrorRows) + " row(s) invalid"
	}
	return strconv.Itoa(res.Imported) + " row(s) saved"
}
