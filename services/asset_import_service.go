package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ============================================================================
// Upload asset — POST /assets/import?mode=new|update
//   new    : asset_number dibentuk otomatis (GenerateAssetNumber, sama dengan
//            procurement), status AVAILABLE, nilai awal di asset_values
//   update : dicari lewat asset_number; kolom kosong = tidak diubah. Nilai aset
//            tidak bisa diubah di sini (lewat penyusutan)
//
// Setiap aset wajib punya category_code yang terdaftar & aktif di Asset
// Category, dan branch_code yang terdaftar & aktif. Non-admin hanya boleh
// cabang miliknya (AssetBranchScope), sama dengan GET /assets.
// ============================================================================

const (
	AssetHistoryImport       = "IMPORT"
	AssetHistoryImportUpdate = "IMPORT_UPDATE"
)

// CanImportAssets — hak akses import_asset di menu permission "Asset Upload"
func CanImportAssets(userID string) bool {
	return userHasMenuPermission(userRoleIDs(userID), "/assets/import", []string{"import_asset"})
}

// kolom deskriptif yang sama di kedua mode
var assetImportDetailColumns = []importColumn{
	{Key: "description", Note: "Keterangan", Example: ""},
	{Key: "brand", Note: "Merek, maks. 100 karakter", Example: "HP"},
	{Key: "unit_of_measure", Note: "Satuan, maks. 50 karakter", Example: "UNIT"},
	{Key: "unit_quantity", Note: "Angka", Example: "1"},
	{Key: "location", Note: "Lokasi, maks. 255 karakter", Example: "Lantai 2"},
	{Key: "grouping", Note: "Maks. 100 karakter", Example: ""},
	{Key: "io_number", Note: "Nomor IO, maks. 100 karakter", Example: ""},
}

func assetImportColumns(mode string) []importColumn {
	var cols []importColumn
	if mode == dto.ImportModeUpdate {
		cols = []importColumn{
			{Key: "asset_number", Required: true, Note: "Nomor aset yang sudah terdaftar (kunci, tidak bisa diubah)", Example: "IT0905260025"},
			{Key: "asset_name", Note: "Nama aset baru", Example: "Laptop HP"},
			{Key: "category_code", Note: "Kode kategori yang terdaftar di Asset Category (lihat daftar di bawah)", Example: "IT"},
			{Key: "branch_code", Note: "Kode cabang terdaftar. Hanya untuk koreksi data — aset harus AVAILABLE dan tidak dipegang user; pemindahan biasa lewat Mutation", Example: "BC000005"},
		}
	} else {
		cols = []importColumn{
			{Key: "asset_name", Required: true, Note: "Nama aset, maks. 255 karakter", Example: "Laptop HP"},
			{Key: "category_code", Required: true, Note: "Kode kategori yang terdaftar & aktif di Asset Category (lihat daftar di bawah)", Example: "IT"},
			{Key: "branch_code", Required: true, Note: "Kode cabang terdaftar & aktif (lihat daftar di bawah)", Example: "BC000005"},
			{Key: "acquisition_value", Required: true, Note: "Nilai perolehan (angka, tanpa titik/koma juga boleh)", Example: "15000000"},
			{Key: "acquisition_date", Required: true, Note: "Tanggal perolehan YYYY-MM-DD atau DD/MM/YYYY", Example: "2026-05-01"},
			{Key: "accumulated_depreciation", Note: "Akumulasi penyusutan s/d tanggal perolehan di atas. Kosong = 0. Nilai buku = perolehan - akumulasi", Example: "0"},
		}
	}
	return append(cols, assetImportDetailColumns...)
}

// BuildAssetImportTemplate — GET /assets/import/template?mode=
func BuildAssetImportTemplate(viewer AssetViewer, mode string) (*excelize.File, error) {
	if err := validateImportMode(mode); err != nil {
		return nil, err
	}

	var categories []models.AssetCategory
	config.DB.Where("deleted_at IS NULL AND is_active = ?", true).Order("category_code ASC").Find(&categories)
	catRows := make([][]string, len(categories))
	for i, c := range categories {
		catRows[i] = []string{c.CategoryCode, c.CategoryName}
	}

	branchQuery := config.DB.Where("status = ?", "active").Order("branch_code ASC")
	if codes, all := AssetBranchScope(viewer); !all {
		branchQuery = branchQuery.Where("branch_code IN ?", append(codes, ""))
	}
	var branches []models.Branch
	branchQuery.Find(&branches)
	branchRows := make([][]string, len(branches))
	for i, b := range branches {
		branchRows[i] = []string{b.BranchCode, b.BranchName, b.BranchType}
	}

	notes := []string{
		"Isi data di sheet Data mulai baris 2. Jangan ubah nama kolom di baris 1; kolom merah wajib diisi.",
		"Satu file diproses utuh: kalau ada satu baris yang salah, tidak ada yang disimpan.",
		"category_code harus sudah terdaftar di menu Asset Category, dan branch_code harus cabang terdaftar yang boleh Anda akses.",
	}
	title := "Upload Asset Baru"
	if mode == dto.ImportModeUpdate {
		title = "Mass Update Asset"
		notes = append(notes, "Kolom yang dikosongkan tidak diubah. Nilai aset tidak bisa diubah lewat upload.")
	} else {
		notes = append(notes, "asset_number tidak perlu diisi — dibentuk otomatis dari kode kategori, sama dengan aset hasil procurement. Status aset baru: AVAILABLE.")
	}

	return buildImportTemplate(title, assetImportColumns(mode), notes, []importRefList{
		{Title: "Kategori aset", Headers: []string{"category_code", "category_name"}, Rows: catRows},
		{Title: "Cabang", Headers: []string{"branch_code", "branch_name", "branch_type"}, Rows: branchRows},
	})
}

type assetImportItem struct {
	line        int
	asset       *models.Asset // update: aset yang ada
	name        string
	category    *models.AssetCategory
	branchCode  string
	acqValue    float64
	accumulated float64
	acqDate     time.Time
	details     map[string]string // kolom deskriptif yang diisi
	quantity    *float64
}

// ImportAssets — dryRun=true hanya validasi.
func ImportAssets(viewer AssetViewer, r io.Reader, mode string, dryRun bool) (*dto.ImportResult, error) {
	if err := validateImportMode(mode); err != nil {
		return nil, err
	}
	columns := assetImportColumns(mode)
	rows, err := readImportRows(r, columns)
	if err != nil {
		return nil, err
	}

	var categories []models.AssetCategory
	if err := config.DB.Where("deleted_at IS NULL").Find(&categories).Error; err != nil {
		return nil, err
	}
	catByCode := make(map[string]*models.AssetCategory, len(categories))
	for i := range categories {
		catByCode[strings.ToUpper(categories[i].CategoryCode)] = &categories[i]
	}

	var branches []models.Branch
	if err := config.DB.Find(&branches).Error; err != nil {
		return nil, err
	}
	branchByCode := make(map[string]models.Branch, len(branches))
	for _, b := range branches {
		branchByCode[strings.ToUpper(b.BranchCode)] = b
	}

	scopeCodes, scopeAll := AssetBranchScope(viewer)
	inScope := make(map[string]bool, len(scopeCodes))
	for _, c := range scopeCodes {
		inScope[strings.ToUpper(c)] = true
	}

	assetByNumber := map[string]*models.Asset{}
	if mode == dto.ImportModeUpdate {
		var numbers []string
		for _, row := range rows {
			numbers = append(numbers, row.Values["asset_number"])
		}
		var assets []models.Asset
		if err := config.DB.Where("asset_number IN ? AND deleted_at IS NULL", numbers).Find(&assets).Error; err != nil {
			return nil, err
		}
		for i := range assets {
			assetByNumber[strings.ToUpper(assets[i].AssetNumber)] = &assets[i]
		}
	}

	results := newImportRowResults(rows)
	items := make([]assetImportItem, len(rows))
	seenNumber := map[string]int{}

	for i, row := range rows {
		v := row.Values
		res := &results[i]
		item := assetImportItem{line: row.Line, details: map[string]string{}}

		// ---------- asset_number (update) ----------
		if mode == dto.ImportModeUpdate {
			number := strings.ToUpper(v["asset_number"])
			switch {
			case number == "":
				res.Errors = append(res.Errors, errRequired("asset_number"))
			default:
				if first := trackDuplicate(seenNumber, number, row.Line); first > 0 {
					res.Errors = append(res.Errors, errDuplicateInFile("asset_number", number, first))
					break
				}
				a, ok := assetByNumber[number]
				current := ""
				if ok && a.BranchCode != nil {
					current = strings.ToUpper(*a.BranchCode)
				}
				// aset cabang lain diperlakukan "tidak terdaftar", sama dengan detail aset (404)
				if !ok || (!scopeAll && !inScope[current]) {
					res.Errors = append(res.Errors, errNotFound("asset_number", number))
				} else if a.AssetStatus == models.AssetStatusDisposed {
					res.Errors = append(res.Errors, importErr("asset_disposed", "asset_number",
						fmt.Sprintf("asset %s is already disposed", a.AssetNumber), "value", a.AssetNumber))
				} else {
					item.asset = a
					res.Result = a.AssetNumber
				}
			}
			changed := false
			for _, col := range columns[1:] {
				if v[col.Key] != "" {
					changed = true
				}
			}
			if !changed {
				res.Errors = append(res.Errors, importErr("no_changes", "", "nothing to update in this row"))
			}
		}

		// ---------- asset_name ----------
		item.name = v["asset_name"]
		if item.name == "" {
			if mode == dto.ImportModeNew {
				res.Errors = append(res.Errors, errRequired("asset_name"))
			}
		} else if len(item.name) > 255 {
			res.Errors = append(res.Errors, errLength("asset_name", 1, 255))
		}

		// ---------- category_code — wajib terdaftar di Asset Category ----------
		if code := strings.ToUpper(v["category_code"]); code == "" {
			if mode == dto.ImportModeNew {
				res.Errors = append(res.Errors, errRequired("category_code"))
			}
		} else if cat, ok := catByCode[code]; !ok {
			res.Errors = append(res.Errors, errNotFound("category_code", code))
		} else if !cat.IsActive {
			res.Errors = append(res.Errors, errInactive("category_code", code))
		} else {
			item.category = cat
		}

		// ---------- branch_code — wajib terdaftar & dalam akses user ----------
		if code := strings.ToUpper(v["branch_code"]); code == "" {
			if mode == dto.ImportModeNew {
				res.Errors = append(res.Errors, errRequired("branch_code"))
			}
		} else if b, ok := branchByCode[code]; !ok {
			res.Errors = append(res.Errors, errNotFound("branch_code", code))
		} else if b.Status != "active" {
			res.Errors = append(res.Errors, errInactive("branch_code", code))
		} else if !scopeAll && !inScope[code] {
			res.Errors = append(res.Errors, importErr("out_of_scope", "branch_code",
				fmt.Sprintf("you have no access to branch %s", code), "value", code))
		} else {
			item.branchCode = b.BranchCode
			// pindah cabang di luar Mutation hanya untuk aset yang sedang tidak
			// dalam transaksi dan tidak dipegang user
			if a := item.asset; a != nil && (a.BranchCode == nil || !strings.EqualFold(*a.BranchCode, code)) {
				if a.AssetStatus != models.AssetStatusAvailable && a.AssetStatus != "ACTIVE" {
					res.Errors = append(res.Errors, importErr("asset_locked", "branch_code",
						fmt.Sprintf("branch of asset %s cannot be changed while its status is %s", a.AssetNumber, a.AssetStatus),
						"value", a.AssetNumber, "status", a.AssetStatus))
				} else if a.AssignedUserID != nil && *a.AssignedUserID != "" {
					res.Errors = append(res.Errors, importErr("asset_held", "branch_code",
						fmt.Sprintf("asset %s is held by a user — return it first", a.AssetNumber), "value", a.AssetNumber))
				}
			}
		}

		// ---------- nilai (new) ----------
		if mode == dto.ImportModeNew {
			if s := v["acquisition_value"]; s == "" {
				res.Errors = append(res.Errors, errRequired("acquisition_value"))
			} else if n, ok := parseImportNumber(s); !ok {
				res.Errors = append(res.Errors, importErr("invalid_number", "acquisition_value", fmt.Sprintf("acquisition_value %q is not a number", s), "value", s))
			} else if n < 0 {
				res.Errors = append(res.Errors, importErr("negative_number", "acquisition_value", "acquisition_value cannot be negative", "value", s))
			} else {
				item.acqValue = n
			}

			if s := v["accumulated_depreciation"]; s != "" {
				if n, ok := parseImportNumber(s); !ok {
					res.Errors = append(res.Errors, importErr("invalid_number", "accumulated_depreciation", fmt.Sprintf("accumulated_depreciation %q is not a number", s), "value", s))
				} else if n < 0 {
					res.Errors = append(res.Errors, importErr("negative_number", "accumulated_depreciation", "accumulated_depreciation cannot be negative", "value", s))
				} else if n > item.acqValue {
					res.Errors = append(res.Errors, importErr("accumulated_exceeds", "accumulated_depreciation", "accumulated_depreciation cannot exceed acquisition_value"))
				} else {
					item.accumulated = n
				}
			}

			if s := v["acquisition_date"]; s == "" {
				res.Errors = append(res.Errors, errRequired("acquisition_date"))
			} else if d, ok := parseImportDate(s); !ok {
				res.Errors = append(res.Errors, importErr("invalid_date", "acquisition_date", fmt.Sprintf("acquisition_date %q is not a valid date", s), "value", s))
			} else if d.After(time.Now()) {
				res.Errors = append(res.Errors, importErr("future_date", "acquisition_date", "acquisition_date cannot be in the future", "value", s))
			} else {
				item.acqDate = d
				res.Values["acquisition_date"] = d.Format("2006-01-02")
			}
		}

		// ---------- kolom deskriptif ----------
		limits := map[string]int{"brand": 100, "unit_of_measure": 50, "location": 255, "grouping": 100, "io_number": 100}
		for _, col := range assetImportDetailColumns {
			val := v[col.Key]
			if val == "" {
				continue
			}
			if col.Key == "unit_quantity" {
				n, ok := parseImportNumber(val)
				if !ok {
					res.Errors = append(res.Errors, importErr("invalid_number", "unit_quantity", fmt.Sprintf("unit_quantity %q is not a number", val), "value", val))
				} else if n < 0 {
					res.Errors = append(res.Errors, importErr("negative_number", "unit_quantity", "unit_quantity cannot be negative", "value", val))
				} else {
					item.quantity = &n
				}
				continue
			}
			if max, ok := limits[col.Key]; ok && len(val) > max {
				res.Errors = append(res.Errors, errLength(col.Key, 1, max))
				continue
			}
			item.details[col.Key] = val
		}

		items[i] = item
	}

	result := finishImportResult("asset", mode, dryRun, columns, results)
	if dryRun || result.ErrorRows > 0 {
		return result, nil
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for i, item := range items {
		var number string
		var err error
		if mode == dto.ImportModeNew {
			number, err = createImportedAsset(tx, viewer.UserID, item)
		} else {
			number, err = updateImportedAsset(tx, viewer.UserID, item)
		}
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("row %d: %w", item.line, err)
		}
		results[i].Result = number
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	result.Imported = len(items)
	return result, nil
}

func createImportedAsset(tx *gorm.DB, userID string, item assetImportItem) (string, error) {
	number, err := GenerateAssetNumber(tx, item.category.CategoryCode)
	if err != nil {
		return "", fmt.Errorf("failed to generate asset number: %w", err)
	}

	branchCode := item.branchCode
	categoryID := item.category.ID
	asset := models.Asset{
		AssetNumber:   number,
		AssetName:     item.name,
		CategoryID:    &categoryID,
		BranchCode:    &branchCode,
		AssetStatus:   models.AssetStatusAvailable,
		Description:   ptrOrNil(item.details["description"]),
		Brand:         ptrOrNil(item.details["brand"]),
		UnitOfMeasure: ptrOrNil(item.details["unit_of_measure"]),
		UnitQuantity:  item.quantity,
		Location:      ptrOrNil(item.details["location"]),
		Grouping:      ptrOrNil(item.details["grouping"]),
		IONumber:      ptrOrNil(item.details["io_number"]),
	}
	if err := tx.Create(&asset).Error; err != nil {
		return "", err
	}

	status := models.AssetStatusAvailable
	if err := tx.Create(&models.AssetValue{
		AssetID:                 asset.ID,
		EffectiveDate:           item.acqDate,
		AcquisitionValue:        item.acqValue,
		AccumulatedDepreciation: item.accumulated,
		BookValue:               item.acqValue - item.accumulated,
		AssetStatus:             &status,
		IsActive:                true,
	}).Error; err != nil {
		return "", err
	}

	after, _ := json.Marshal(map[string]interface{}{
		"asset_number":             number,
		"asset_name":               item.name,
		"category_code":            item.category.CategoryCode,
		"branch_code":              branchCode,
		"acquisition_value":        item.acqValue,
		"accumulated_depreciation": item.accumulated,
		"acquisition_date":         item.acqDate.Format("2006-01-02"),
	})
	afterStr := string(after)
	now := time.Now()
	if err := CreateAssetHistory(tx, models.AssetHistory{
		AssetID:         asset.ID,
		TransactionType: AssetHistoryImport,
		TransactionDate: &now,
		AfterData:       &afterStr,
		ChangedBy:       &userID,
	}); err != nil {
		return "", err
	}
	return number, nil
}

func updateImportedAsset(tx *gorm.DB, userID string, item assetImportItem) (string, error) {
	a := item.asset
	updates := map[string]interface{}{}
	before := map[string]interface{}{}
	after := map[string]interface{}{}

	set := func(column string, old interface{}, value interface{}) {
		updates[column] = value
		before[column] = old
		after[column] = value
	}
	str := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}

	if item.name != "" {
		set("asset_name", a.AssetName, item.name)
	}
	if item.category != nil {
		set("category_id", a.CategoryID, item.category.ID)
	}
	if item.branchCode != "" {
		set("branch_code", str(a.BranchCode), item.branchCode)
	}
	if item.quantity != nil {
		set("unit_quantity", a.UnitQuantity, *item.quantity)
	}
	current := map[string]string{
		"description": str(a.Description), "brand": str(a.Brand), "unit_of_measure": str(a.UnitOfMeasure),
		"location": str(a.Location), "grouping": str(a.Grouping), "io_number": str(a.IONumber),
	}
	for column, value := range item.details {
		set(column, current[column], value)
	}

	if len(updates) == 0 {
		return a.AssetNumber, nil
	}
	if err := tx.Model(&models.Asset{}).Where("id = ?", a.ID).Updates(updates).Error; err != nil {
		return "", err
	}

	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	beforeStr, afterStr := string(beforeJSON), string(afterJSON)
	now := time.Now()
	if err := CreateAssetHistory(tx, models.AssetHistory{
		AssetID:         a.ID,
		TransactionType: AssetHistoryImportUpdate,
		TransactionDate: &now,
		BeforeData:      &beforeStr,
		AfterData:       &afterStr,
		ChangedBy:       &userID,
	}); err != nil {
		return "", err
	}
	return a.AssetNumber, nil
}

// ExportAssets — GET /assets/export: aset dalam format template mass update,
// dibatasi cabang user (AssetBranchScope) dan filter halaman Asset. Aset
// DISPOSED tidak ikut karena memang tidak bisa diubah lewat upload.
func ExportAssets(viewer AssetViewer, filter dto.AssetListFilter) (*excelize.File, error) {
	f, err := BuildAssetImportTemplate(viewer, dto.ImportModeUpdate)
	if err != nil {
		return nil, err
	}

	query := config.DB.Model(&models.Asset{}).
		Where("deleted_at IS NULL AND asset_status <> ?", models.AssetStatusDisposed)
	if codes, all := AssetBranchScope(viewer); !all {
		query = query.Where("branch_code IN ?", append(codes, ""))
	}
	if filter.BranchCode != nil && *filter.BranchCode != "" {
		query = query.Where("branch_code = ?", *filter.BranchCode)
	}
	if filter.CategoryID != nil {
		query = query.Where("category_id = ?", *filter.CategoryID)
	}
	if filter.AssetStatus != nil && *filter.AssetStatus != "" {
		query = query.Where("asset_status = ?", *filter.AssetStatus)
	}
	if filter.Search != nil && *filter.Search != "" {
		search := "%" + *filter.Search + "%"
		query = query.Where("(asset_number LIKE ? OR asset_name LIKE ?)", search, search)
	}

	var assets []models.Asset
	if err := query.Preload("Category").Order("asset_number ASC").Find(&assets).Error; err != nil {
		f.Close()
		return nil, err
	}

	str := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}

	// urutan = assetImportColumns(update)
	rows := make([][]string, len(assets))
	for i, a := range assets {
		category := ""
		if a.Category != nil {
			category = a.Category.CategoryCode
		}
		quantity := ""
		if a.UnitQuantity != nil {
			quantity = strconv.FormatFloat(*a.UnitQuantity, 'f', -1, 64)
		}
		rows[i] = []string{
			a.AssetNumber, a.AssetName, category, str(a.BranchCode),
			str(a.Description), str(a.Brand), str(a.UnitOfMeasure), quantity,
			str(a.Location), str(a.Grouping), str(a.IONumber),
		}
	}
	fillImportRows(f, rows)
	return f, nil
}
