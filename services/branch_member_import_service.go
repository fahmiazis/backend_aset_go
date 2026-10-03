package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ============================================================================
// Upload branch mode "member" — set homebase & akses cabang user lewat Excel.
//   type = homebase   → jadi homebase AKTIF user (homebase lama dinonaktifkan,
//                       sama dengan panel Anggota Homebase)
//   type = assignment → akses cabang tambahan; kalau user sudah punya baris di
//                       cabang itu dibiarkan (sama dengan panel Akses Cabang)
// Hanya menambah, tidak mencabut akses yang tidak ada di file.
// Satu user hanya boleh punya satu baris homebase dalam satu file.
// ============================================================================

// hasil per baris (dto.ImportRowResult.Result) — diterjemahkan frontend
const (
	memberResultSetHomebase = "set_homebase"
	memberResultAddAccess   = "add_access"
	memberResultUnchanged   = "unchanged"
)

var branchMemberTypes = []string{branchTypeHomebase, branchTypeAssignment}

func branchMemberImportColumns() []importColumn {
	return []importColumn{
		{Key: "username", Required: true, Note: "Username yang sudah terdaftar", Example: "budi"},
		{Key: "branch_code", Required: true, Note: "Kode cabang terdaftar & aktif (lihat daftar di bawah)", Example: "BC000005"},
		{Key: "type", Required: true, Note: "homebase = jadi homebase aktif (homebase lama dinonaktifkan); assignment = akses cabang tambahan. Satu user hanya boleh satu baris homebase", Example: "homebase"},
	}
}

func buildBranchMemberTemplate() (*excelize.File, error) {
	var branches []models.Branch
	config.DB.Where("status = ?", "active").Order("branch_code ASC").Find(&branches)
	branchRows := make([][]string, len(branches))
	for i, b := range branches {
		branchRows[i] = []string{b.BranchCode, b.BranchName, b.BranchType}
	}

	return buildImportTemplate("Upload Homebase & Akses Cabang", branchMemberImportColumns(), []string{
		"Isi data di sheet Data mulai baris 2. Jangan ubah nama kolom di baris 1; kolom merah wajib diisi.",
		"Satu file diproses utuh: kalau ada satu baris yang salah, tidak ada yang disimpan.",
		"Satu baris = satu user di satu cabang. Satu user boleh banyak baris assignment, tapi hanya SATU baris homebase.",
		"Upload hanya menambah — akses yang tidak tercantum di file tidak dicabut.",
	}, []importRefList{
		{Title: "Tipe", Headers: []string{"type", "keterangan"}, Rows: [][]string{
			{branchTypeHomebase, "Homebase aktif (menentukan kode cabang pada nomor transaksi)"},
			{branchTypeAssignment, "Akses cabang tambahan"},
		}},
		{Title: "Cabang aktif", Headers: []string{"branch_code", "branch_name", "branch_type"}, Rows: branchRows},
	})
}

type branchMemberItem struct {
	line     int
	userID   string
	branchID string
	mtype    string
}

func importBranchMembers(r io.Reader, dryRun bool) (*dto.ImportResult, error) {
	mode := dto.ImportModeMember
	columns := branchMemberImportColumns()
	rows, err := readImportRows(r, columns)
	if err != nil {
		return nil, err
	}

	var usernames []string
	for _, row := range rows {
		usernames = append(usernames, row.Values["username"])
	}
	var users []models.User
	if err := config.DB.Where("username IN ?", usernames).Find(&users).Error; err != nil {
		return nil, err
	}
	userByName := make(map[string]models.User, len(users))
	userIDs := make([]string, 0, len(users))
	for _, u := range users {
		userByName[strings.ToLower(u.Username)] = u
		userIDs = append(userIDs, u.ID)
	}

	var branches []models.Branch
	if err := config.DB.Find(&branches).Error; err != nil {
		return nil, err
	}
	branchByCode := make(map[string]models.Branch, len(branches))
	for _, b := range branches {
		branchByCode[strings.ToUpper(b.BranchCode)] = b
	}

	// baris user_branchs yang sudah ada, untuk menentukan hasil "unchanged"
	existing := map[string]models.UserBranch{} // user_id|branch_id
	if len(userIDs) > 0 {
		var ubs []models.UserBranch
		if err := config.DB.Where("user_id IN ?", userIDs).Find(&ubs).Error; err != nil {
			return nil, err
		}
		for _, ub := range ubs {
			existing[ub.UserID+"|"+ub.BranchID] = ub
		}
	}

	results := newImportRowResults(rows)
	items := make([]branchMemberItem, len(rows))
	seenPair := map[string]int{}
	homebaseLines := map[string][]int{} // username → index baris bertipe homebase

	for i, row := range rows {
		v := row.Values
		res := &results[i]
		item := branchMemberItem{line: row.Line}

		username := v["username"]
		if username == "" {
			res.Errors = append(res.Errors, errRequired("username"))
		} else if u, ok := userByName[strings.ToLower(username)]; !ok {
			res.Errors = append(res.Errors, errNotFound("username", username))
		} else {
			item.userID = u.ID
		}

		code := strings.ToUpper(v["branch_code"])
		if code == "" {
			res.Errors = append(res.Errors, errRequired("branch_code"))
		} else if b, ok := branchByCode[code]; !ok {
			res.Errors = append(res.Errors, errNotFound("branch_code", code))
		} else if b.Status != "active" {
			res.Errors = append(res.Errors, errInactive("branch_code", code))
		} else {
			item.branchID = b.ID
		}

		mtype := strings.ToLower(v["type"])
		switch {
		case mtype == "":
			res.Errors = append(res.Errors, errRequired("type"))
		case mtype != branchTypeHomebase && mtype != branchTypeAssignment:
			res.Errors = append(res.Errors, errInvalidOption("type", v["type"], branchMemberTypes))
		default:
			item.mtype = mtype
			if mtype == branchTypeHomebase && username != "" {
				key := strings.ToLower(username)
				homebaseLines[key] = append(homebaseLines[key], i)
			}
		}

		if username != "" && code != "" {
			if first := trackDuplicate(seenPair, username+"|"+code, row.Line); first > 0 {
				res.Errors = append(res.Errors, errDuplicateInFile("branch_code", username+" / "+code, first))
			}
		}

		if item.userID != "" && item.branchID != "" && item.mtype != "" {
			ub, has := existing[item.userID+"|"+item.branchID]
			switch {
			case item.mtype == branchTypeHomebase && has && ub.BranchType == branchTypeHomebase && ub.IsActive:
				res.Result = memberResultUnchanged
			case item.mtype == branchTypeHomebase:
				res.Result = memberResultSetHomebase
			case has:
				// sudah punya baris di cabang ini (termasuk homebase) — dibiarkan
				res.Result = memberResultUnchanged
			default:
				res.Result = memberResultAddAccess
			}
		}

		items[i] = item
	}

	// satu user hanya boleh satu homebase per file — semua barisnya ditandai
	for _, idxs := range homebaseLines {
		if len(idxs) < 2 {
			continue
		}
		lines := make([]int, len(idxs))
		for j, idx := range idxs {
			lines[j] = rows[idx].Line
		}
		sort.Ints(lines)
		lineStrs := make([]string, len(lines))
		for j, l := range lines {
			lineStrs[j] = strconv.Itoa(l)
		}
		name := rows[idxs[0]].Values["username"]
		for _, idx := range idxs {
			results[idx].Errors = append(results[idx].Errors, importErr("multiple_homebase", "type",
				fmt.Sprintf("user %s has more than one homebase row (rows %s)", name, strings.Join(lineStrs, ", ")),
				"value", name, "rows", strings.Join(lineStrs, ", ")))
		}
	}

	result := finishImportResult("branch", mode, dryRun, columns, results)
	if dryRun || result.ErrorRows > 0 {
		return result, nil
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, item := range items {
		if err := applyBranchMemberItem(tx, item); err != nil {
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

func applyBranchMemberItem(tx *gorm.DB, item branchMemberItem) error {
	if item.mtype == branchTypeHomebase {
		return setActiveHomebaseTx(tx, item.userID, item.branchID)
	}

	// assignment — aturan AssignBranchUsers: baris yang sudah ada dibiarkan,
	// homebase tidak pernah diturunkan jadi assignment
	var existing models.UserBranch
	err := tx.Where("user_id = ? AND branch_id = ?", item.userID, item.branchID).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return tx.Create(&models.UserBranch{
			UserID:     item.userID,
			BranchID:   item.branchID,
			BranchType: branchTypeAssignment,
		}).Error
	case err == nil:
		return nil
	default:
		return err
	}
}
