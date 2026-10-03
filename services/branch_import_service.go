package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ============================================================================
// Upload branch — POST /branchs/import?mode=new|update|member
//   new    : branch_code dibentuk otomatis (BC000001, …) oleh models.Branch.BeforeCreate
//   update : baris dicari lewat branch_code; kolom kosong = tidak diubah
//   member : set homebase / akses cabang user (branch_member_import_service.go)
// ============================================================================

func branchImportColumns(mode string) []importColumn {
	if mode == dto.ImportModeUpdate {
		return []importColumn{
			{Key: "branch_code", Required: true, Note: "Kode cabang yang sudah terdaftar (kunci, tidak bisa diubah)", Example: "BC000004"},
			{Key: "branch_name", Note: "Nama baru. Kosongkan kalau tidak diubah", Example: "BANDUNG"},
			{Key: "branch_type", Note: "Tipe baru (lihat daftar tipe di bawah). Kosongkan kalau tidak diubah", Example: "HO"},
			{Key: "status", Note: "active / inactive. Kosongkan kalau tidak diubah", Example: "active"},
		}
	}
	return []importColumn{
		{Key: "branch_name", Required: true, Note: "Nama cabang", Example: "BANDUNG"},
		{Key: "branch_type", Required: true, Note: "Tipe cabang, mis. HO / SUB BRANCH (lihat daftar tipe di bawah)", Example: "SUB BRANCH"},
		{Key: "status", Note: "active / inactive. Kosong = active", Example: "active"},
	}
}

// BuildBranchImportTemplate — GET /branchs/import/template?mode=
func BuildBranchImportTemplate(mode string) (*excelize.File, error) {
	if mode == dto.ImportModeMember {
		return buildBranchMemberTemplate()
	}
	if err := validateImportMode(mode); err != nil {
		return nil, err
	}

	var branches []models.Branch
	config.DB.Order("branch_code ASC").Find(&branches)

	notes := []string{
		"Isi data di sheet Data mulai baris 2. Jangan ubah nama kolom di baris 1; kolom merah wajib diisi.",
		"Satu file diproses utuh: kalau ada satu baris yang salah, tidak ada yang disimpan.",
	}
	title := "Upload Branch Baru"
	refs := []importRefList{{Title: "Tipe cabang yang sudah dipakai", Headers: []string{"branch_type"}, Rows: branchTypeRows(branches)}}

	if mode == dto.ImportModeUpdate {
		title = "Mass Update Branch"
		notes = append(notes, "Kolom yang dikosongkan tidak diubah.")
		rows := make([][]string, len(branches))
		for i, b := range branches {
			rows[i] = []string{b.BranchCode, b.BranchName, b.BranchType, b.Status}
		}
		refs = append(refs, importRefList{Title: "Cabang terdaftar", Headers: []string{"branch_code", "branch_name", "branch_type", "status"}, Rows: rows})
	} else {
		notes = append(notes, "branch_code tidak perlu diisi — dibentuk otomatis berurutan saat disimpan.")
	}

	return buildImportTemplate(title, branchImportColumns(mode), notes, refs)
}

func branchTypeRows(branches []models.Branch) [][]string {
	seen := map[string]bool{}
	var types []string
	for _, b := range branches {
		if !seen[b.BranchType] {
			seen[b.BranchType] = true
			types = append(types, b.BranchType)
		}
	}
	sort.Strings(types)
	rows := make([][]string, len(types))
	for i, t := range types {
		rows[i] = []string{t}
	}
	return rows
}

// canonicalBranchType — "ho" ditulis jadi "HO" kalau tipe itu sudah dipakai,
// supaya tipe cabang tidak bercabang hanya karena beda huruf besar/kecil.
func canonicalBranchType(value string, branches []models.Branch) string {
	for _, b := range branches {
		if strings.EqualFold(b.BranchType, value) {
			return b.BranchType
		}
	}
	return value
}

type branchImportItem struct {
	line   int
	branch *models.Branch // update: baris yang ada
	name   string
	btype  string
	status string
}

// ImportBranches — dryRun=true hanya validasi.
func ImportBranches(r io.Reader, mode string, dryRun bool) (*dto.ImportResult, error) {
	if mode == dto.ImportModeMember {
		return importBranchMembers(r, dryRun)
	}
	if err := validateImportMode(mode); err != nil {
		return nil, err
	}
	columns := branchImportColumns(mode)
	rows, err := readImportRows(r, columns)
	if err != nil {
		return nil, err
	}

	var branches []models.Branch
	if err := config.DB.Find(&branches).Error; err != nil {
		return nil, err
	}
	byCode := make(map[string]*models.Branch, len(branches))
	for i := range branches {
		byCode[strings.ToUpper(branches[i].BranchCode)] = &branches[i]
	}

	// nama+tipe cabang aktif — aturan sama dengan CreateBranch
	activeKey := func(name, btype string) string {
		return strings.ToLower(name) + "\x00" + strings.ToLower(btype)
	}
	activeOwner := make(map[string]string) // key → branch_code pemilik
	for _, b := range branches {
		if b.Status == "active" {
			activeOwner[activeKey(b.BranchName, b.BranchType)] = b.BranchCode
		}
	}

	results := newImportRowResults(rows)
	items := make([]branchImportItem, len(rows))
	seenCode := map[string]int{}
	seenActive := map[string]int{}

	for i, row := range rows {
		v := row.Values
		res := &results[i]
		item := branchImportItem{line: row.Line}

		if mode == dto.ImportModeUpdate {
			code := strings.ToUpper(v["branch_code"])
			if code == "" {
				res.Errors = append(res.Errors, errRequired("branch_code"))
			} else if first := trackDuplicate(seenCode, code, row.Line); first > 0 {
				res.Errors = append(res.Errors, errDuplicateInFile("branch_code", code, first))
			} else if b, ok := byCode[code]; !ok {
				res.Errors = append(res.Errors, errNotFound("branch_code", code))
			} else {
				item.branch = b
				res.Result = b.BranchCode
			}
			if v["branch_name"] == "" && v["branch_type"] == "" && v["status"] == "" {
				res.Errors = append(res.Errors, importErr("no_changes", "", "nothing to update in this row"))
			}
		} else {
			if v["branch_name"] == "" {
				res.Errors = append(res.Errors, errRequired("branch_name"))
			}
			if v["branch_type"] == "" {
				res.Errors = append(res.Errors, errRequired("branch_type"))
			}
		}

		item.name = v["branch_name"]
		if len(item.name) > 255 {
			res.Errors = append(res.Errors, errLength("branch_name", 1, 255))
		}
		item.btype = canonicalBranchType(v["branch_type"], branches)
		if len(item.btype) > 50 {
			res.Errors = append(res.Errors, errLength("branch_type", 1, 50))
		}
		if v["status"] != "" {
			status, e := parseImportStatus("status", v["status"])
			if e != nil {
				res.Errors = append(res.Errors, *e)
			}
			item.status = status
		} else if mode == dto.ImportModeNew {
			item.status = "active"
		}

		// Hasil akhir nama+tipe+status, untuk cek bentrok dengan cabang aktif lain
		finalName, finalType, finalStatus := item.name, item.btype, item.status
		ownCode := ""
		if item.branch != nil {
			ownCode = item.branch.BranchCode
			if finalName == "" {
				finalName = item.branch.BranchName
			}
			if finalType == "" {
				finalType = item.branch.BranchType
			}
			if finalStatus == "" {
				finalStatus = item.branch.Status
			}
		}
		if finalName != "" && finalType != "" && finalStatus == "active" && (mode == dto.ImportModeNew || item.branch != nil) {
			key := activeKey(finalName, finalType)
			if owner, ok := activeOwner[key]; ok && owner != ownCode {
				res.Errors = append(res.Errors, importErr("branch_exists", "branch_name",
					fmt.Sprintf("active branch %s / %s already exists (%s)", finalName, finalType, owner),
					"value", finalName+" / "+finalType, "code", owner))
			} else if first := trackDuplicate(seenActive, key, row.Line); first > 0 {
				res.Errors = append(res.Errors, errDuplicateInFile("branch_name", finalName+" / "+finalType, first))
			}
		}

		items[i] = item
	}

	result := finishImportResult("branch", mode, dryRun, columns, results)

	if mode == dto.ImportModeNew && result.ErrorRows == 0 {
		// perkiraan kode untuk preview; kode sebenarnya dibentuk saat simpan
		var next int
		config.DB.Raw(`SELECT COALESCE(MAX(CAST(SUBSTRING(branch_code, 3) AS UNSIGNED)), 0) FROM branchs`).Scan(&next)
		for i := range results {
			next++
			results[i].Result = "BC" + leftPad(strconv.Itoa(next), 6)
		}
	}

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
		if mode == dto.ImportModeNew {
			branch := models.Branch{BranchName: item.name, BranchType: item.btype, Status: item.status}
			if err := tx.Create(&branch).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("row %d: %w", item.line, err)
			}
			results[i].Result = branch.BranchCode
			continue
		}

		updates := map[string]interface{}{}
		if item.name != "" {
			updates["branch_name"] = item.name
		}
		if item.btype != "" {
			updates["branch_type"] = item.btype
		}
		if item.status != "" {
			updates["status"] = item.status
		}
		if err := tx.Model(&models.Branch{}).Where("id = ?", item.branch.ID).Updates(updates).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("row %d: %w", item.line, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	result.Imported = len(items)
	return result, nil
}

func leftPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat("0", width-len(s)) + s
}

// ExportBranches — GET /branchs/export: seluruh cabang dalam format template
// mass update, jadi bisa diedit lalu diunggah ulang dengan mode update.
func ExportBranches() (*excelize.File, error) {
	f, err := BuildBranchImportTemplate(dto.ImportModeUpdate)
	if err != nil {
		return nil, err
	}
	var branches []models.Branch
	if err := config.DB.Order("branch_code ASC").Find(&branches).Error; err != nil {
		f.Close()
		return nil, err
	}
	rows := make([][]string, len(branches))
	for i, b := range branches {
		rows[i] = []string{b.BranchCode, b.BranchName, b.BranchType, b.Status}
	}
	fillImportRows(f, rows)
	return f, nil
}
