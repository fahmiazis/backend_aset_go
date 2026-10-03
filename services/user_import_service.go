package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"fmt"
	"io"
	"net/mail"
	"strings"

	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ============================================================================
// Upload user — POST /users/import?mode=new|update
//   new    : user + satu role + homebase aktif (opsional)
//   update : dicari lewat username; kolom kosong = tidak diubah
// ============================================================================

func userImportColumns(mode string) []importColumn {
	if mode == dto.ImportModeUpdate {
		return []importColumn{
			{Key: "username", Required: true, Note: "Username yang sudah terdaftar (kunci, tidak bisa diubah)", Example: "budi"},
			{Key: "fullname", Note: "3-100 karakter", Example: "Budi Santoso"},
			{Key: "email", Note: "Email baru, harus unik", Example: "budi@contoh.com"},
			{Key: "password", Note: "Isi hanya kalau ingin mengganti password (min. 6 karakter)", Example: ""},
			{Key: "role", Note: "Nama role (lihat daftar di bawah). Satu user satu role", Example: "pic asset"},
			{Key: "homebase_branch_code", Note: "Kode cabang homebase aktif yang baru. Homebase lama dinonaktifkan", Example: "BC000005"},
			{Key: "nik", Note: "Harus unik", Example: "3201xxxxxxxx"},
			{Key: "mpn_number", Note: "", Example: ""},
			{Key: "status", Note: "active / inactive", Example: "active"},
		}
	}
	return []importColumn{
		{Key: "username", Required: true, Note: "3-50 karakter, unik", Example: "budi"},
		{Key: "fullname", Required: true, Note: "3-100 karakter", Example: "Budi Santoso"},
		{Key: "email", Required: true, Note: "Harus unik", Example: "budi@contoh.com"},
		{Key: "password", Required: true, Note: "Minimal 6 karakter", Example: "rahasia123"},
		{Key: "role", Required: true, Note: "Nama role (lihat daftar di bawah). Satu user satu role", Example: "pic asset"},
		{Key: "homebase_branch_code", Note: "Kode cabang homebase (lihat daftar di bawah). Boleh kosong", Example: "BC000005"},
		{Key: "nik", Note: "Harus unik. Boleh kosong", Example: "3201xxxxxxxx"},
		{Key: "mpn_number", Note: "Boleh kosong", Example: ""},
		{Key: "status", Note: "active / inactive. Kosong = active", Example: "active"},
	}
}

// BuildUserImportTemplate — GET /users/import/template?mode=
func BuildUserImportTemplate(mode string) (*excelize.File, error) {
	if err := validateImportMode(mode); err != nil {
		return nil, err
	}

	var roles []models.Role
	config.DB.Order("name ASC").Find(&roles)
	roleRows := make([][]string, len(roles))
	for i, r := range roles {
		roleRows[i] = []string{r.Name, r.Description}
	}

	var branches []models.Branch
	config.DB.Where("status = ?", "active").Order("branch_code ASC").Find(&branches)
	branchRows := make([][]string, len(branches))
	for i, b := range branches {
		branchRows[i] = []string{b.BranchCode, b.BranchName, b.BranchType}
	}

	notes := []string{
		"Isi data di sheet Data mulai baris 2. Jangan ubah nama kolom di baris 1; kolom merah wajib diisi.",
		"Satu file diproses utuh: kalau ada satu baris yang salah, tidak ada yang disimpan.",
	}
	title := "Upload User Baru"
	if mode == dto.ImportModeUpdate {
		title = "Mass Update User"
		notes = append(notes, "Kolom yang dikosongkan tidak diubah.")
	}

	return buildImportTemplate(title, userImportColumns(mode), notes, []importRefList{
		{Title: "Role", Headers: []string{"role", "keterangan"}, Rows: roleRows},
		{Title: "Cabang aktif", Headers: []string{"branch_code", "branch_name", "branch_type"}, Rows: branchRows},
	})
}

type userImportItem struct {
	line     int
	user     *models.User // update: user yang ada
	username string
	fullname string
	email    string
	password string
	roleID   string
	branchID string
	nik      string
	mpn      string
	status   string
}

// ImportUsers — dryRun=true hanya validasi.
func ImportUsers(r io.Reader, mode string, dryRun bool) (*dto.ImportResult, error) {
	if err := validateImportMode(mode); err != nil {
		return nil, err
	}
	columns := userImportColumns(mode)
	rows, err := readImportRows(r, columns)
	if err != nil {
		return nil, err
	}

	var roles []models.Role
	if err := config.DB.Find(&roles).Error; err != nil {
		return nil, err
	}
	roleByName := make(map[string]string, len(roles))
	for _, role := range roles {
		roleByName[strings.ToLower(role.Name)] = role.ID
	}

	var branches []models.Branch
	if err := config.DB.Find(&branches).Error; err != nil {
		return nil, err
	}
	branchByCode := make(map[string]models.Branch, len(branches))
	for _, b := range branches {
		branchByCode[strings.ToUpper(b.BranchCode)] = b
	}

	// User yang sudah ada, dicari sekaligus. Unscoped karena index unik
	// username/email/nik ikut mencakup user yang sudah di-soft-delete.
	var usernames, emails, niks []string
	for _, row := range rows {
		usernames = append(usernames, row.Values["username"])
		if row.Values["email"] != "" {
			emails = append(emails, row.Values["email"])
		}
		if row.Values["nik"] != "" {
			niks = append(niks, row.Values["nik"])
		}
	}
	var existing []models.User
	q := config.DB.Unscoped().Where("username IN ?", usernames)
	if len(emails) > 0 {
		q = q.Or("email IN ?", emails)
	}
	if len(niks) > 0 {
		q = q.Or("nik IN ?", niks)
	}
	if err := q.Find(&existing).Error; err != nil {
		return nil, err
	}
	byUsername := map[string]*models.User{}
	emailOwner := map[string]string{} // email → user id
	nikOwner := map[string]string{}
	for i := range existing {
		u := &existing[i]
		byUsername[strings.ToLower(u.Username)] = u
		emailOwner[strings.ToLower(u.Email)] = u.ID
		if u.NIK != nil && *u.NIK != "" {
			nikOwner[strings.ToLower(*u.NIK)] = u.ID
		}
	}

	results := newImportRowResults(rows, "password")
	items := make([]userImportItem, len(rows))
	seenUsername, seenEmail, seenNIK := map[string]int{}, map[string]int{}, map[string]int{}

	for i, row := range rows {
		v := row.Values
		res := &results[i]
		item := userImportItem{line: row.Line, username: v["username"]}
		ownID := ""

		// ---------- username ----------
		switch {
		case item.username == "":
			res.Errors = append(res.Errors, errRequired("username"))
		case len(item.username) < 3 || len(item.username) > 50:
			res.Errors = append(res.Errors, errLength("username", 3, 50))
		default:
			if first := trackDuplicate(seenUsername, item.username, row.Line); first > 0 {
				res.Errors = append(res.Errors, errDuplicateInFile("username", item.username, first))
				break
			}
			u, found := byUsername[strings.ToLower(item.username)]
			if mode == dto.ImportModeNew {
				if found {
					res.Errors = append(res.Errors, errAlreadyExists("username", item.username))
				}
			} else if !found || u.DeletedAt.Valid {
				res.Errors = append(res.Errors, errNotFound("username", item.username))
			} else {
				item.user = u
				ownID = u.ID
				res.Result = u.Username
			}
		}

		if mode == dto.ImportModeUpdate {
			changed := false
			for _, col := range columns[1:] {
				if v[col.Key] != "" {
					changed = true
				}
			}
			if !changed {
				res.Errors = append(res.Errors, importErr("no_changes", "", "nothing to update in this row"))
			}
		} else {
			res.Result = item.username
		}

		// ---------- fullname ----------
		item.fullname = v["fullname"]
		if item.fullname == "" {
			if mode == dto.ImportModeNew {
				res.Errors = append(res.Errors, errRequired("fullname"))
			}
		} else if len(item.fullname) < 3 || len(item.fullname) > 100 {
			res.Errors = append(res.Errors, errLength("fullname", 3, 100))
		}

		// ---------- email ----------
		item.email = v["email"]
		if item.email == "" {
			if mode == dto.ImportModeNew {
				res.Errors = append(res.Errors, errRequired("email"))
			}
		} else if addr, err := mail.ParseAddress(item.email); err != nil || addr.Address != item.email {
			res.Errors = append(res.Errors, importErr("invalid_email", "email", fmt.Sprintf("email %q is not valid", item.email), "value", item.email))
		} else if first := trackDuplicate(seenEmail, item.email, row.Line); first > 0 {
			res.Errors = append(res.Errors, errDuplicateInFile("email", item.email, first))
		} else if owner, ok := emailOwner[strings.ToLower(item.email)]; ok && owner != ownID {
			res.Errors = append(res.Errors, errAlreadyExists("email", item.email))
		}

		// ---------- password ----------
		item.password = v["password"]
		if item.password == "" {
			if mode == dto.ImportModeNew {
				res.Errors = append(res.Errors, errRequired("password"))
			}
		} else if len(item.password) < 6 {
			res.Errors = append(res.Errors, importErr("min_length", "password", "password must be at least 6 characters", "min", "6"))
		}

		// ---------- role ----------
		if role := v["role"]; role == "" {
			if mode == dto.ImportModeNew {
				res.Errors = append(res.Errors, errRequired("role"))
			}
		} else if id, ok := roleByName[strings.ToLower(role)]; !ok {
			res.Errors = append(res.Errors, errNotFound("role", role))
		} else {
			item.roleID = id
		}

		// ---------- homebase ----------
		if code := strings.ToUpper(v["homebase_branch_code"]); code != "" {
			if b, ok := branchByCode[code]; !ok {
				res.Errors = append(res.Errors, errNotFound("homebase_branch_code", code))
			} else if b.Status != "active" {
				res.Errors = append(res.Errors, errInactive("homebase_branch_code", code))
			} else {
				item.branchID = b.ID
			}
		}

		// ---------- nik / mpn ----------
		item.nik = v["nik"]
		if item.nik != "" {
			if len(item.nik) > 20 {
				res.Errors = append(res.Errors, errLength("nik", 1, 20))
			} else if first := trackDuplicate(seenNIK, item.nik, row.Line); first > 0 {
				res.Errors = append(res.Errors, errDuplicateInFile("nik", item.nik, first))
			} else if owner, ok := nikOwner[strings.ToLower(item.nik)]; ok && owner != ownID {
				res.Errors = append(res.Errors, errAlreadyExists("nik", item.nik))
			}
		}
		item.mpn = v["mpn_number"]
		if len(item.mpn) > 50 {
			res.Errors = append(res.Errors, errLength("mpn_number", 1, 50))
		}

		// ---------- status ----------
		if v["status"] != "" {
			status, e := parseImportStatus("status", v["status"])
			if e != nil {
				res.Errors = append(res.Errors, *e)
			}
			item.status = status
		} else if mode == dto.ImportModeNew {
			item.status = "active"
		}

		items[i] = item
	}

	result := finishImportResult("user", mode, dryRun, columns, results)
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
		if err := applyUserImportItem(tx, mode, item); err != nil {
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

func applyUserImportItem(tx *gorm.DB, mode string, item userImportItem) error {
	var hashed string
	if item.password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(item.password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password")
		}
		hashed = string(h)
	}

	userID := ""
	if mode == dto.ImportModeNew {
		user := models.User{
			Username:  item.username,
			Fullname:  item.fullname,
			Email:     item.email,
			Password:  hashed,
			NIK:       ptrOrNil(item.nik),
			MPNNumber: ptrOrNil(item.mpn),
			Status:    item.status,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		userID = user.ID
	} else {
		userID = item.user.ID
		updates := map[string]interface{}{}
		if item.fullname != "" {
			updates["fullname"] = item.fullname
		}
		if item.email != "" {
			updates["email"] = item.email
		}
		if hashed != "" {
			updates["password"] = hashed
		}
		if item.nik != "" {
			updates["nik"] = item.nik
		}
		if item.mpn != "" {
			updates["mpn_number"] = item.mpn
		}
		if item.status != "" {
			updates["status"] = item.status
		}
		if len(updates) > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
				return err
			}
		}
	}

	// satu user = satu role (lihat AssignRoles)
	if item.roleID != "" {
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.UserRole{UserID: userID, RoleID: item.roleID}).Error; err != nil {
			return err
		}
	}

	if item.branchID != "" {
		if err := setActiveHomebaseTx(tx, userID, item.branchID); err != nil {
			return err
		}
	}
	return nil
}

// ExportUsers — GET /users/export: seluruh user dalam format template mass
// update. Kolom password sengaja kosong (tidak diubah saat diunggah ulang);
// homebase_branch_code = homebase yang sedang aktif.
func ExportUsers() (*excelize.File, error) {
	f, err := BuildUserImportTemplate(dto.ImportModeUpdate)
	if err != nil {
		return nil, err
	}

	var users []models.User
	if err := config.DB.Preload("UserRoles.Role").Order("username ASC").Find(&users).Error; err != nil {
		f.Close()
		return nil, err
	}

	type homebaseRow struct {
		UserID     string
		BranchCode string
	}
	var homebases []homebaseRow
	config.DB.Table("user_branchs ub").
		Select("ub.user_id, b.branch_code").
		Joins("JOIN branchs b ON b.id = ub.branch_id AND b.deleted_at IS NULL").
		Where("ub.branch_type = ? AND ub.is_active = ?", branchTypeHomebase, true).
		Scan(&homebases)
	homebaseOf := make(map[string]string, len(homebases))
	for _, h := range homebases {
		homebaseOf[h.UserID] = h.BranchCode
	}

	str := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}

	// urutan = userImportColumns(update)
	rows := make([][]string, len(users))
	for i, u := range users {
		role := ""
		if len(u.UserRoles) > 0 && u.UserRoles[0].Role != nil {
			role = u.UserRoles[0].Role.Name
		}
		rows[i] = []string{u.Username, u.Fullname, u.Email, "", role, homebaseOf[u.ID], str(u.NIK), str(u.MPNNumber), u.Status}
	}
	fillImportRows(f, rows)
	return f, nil
}
