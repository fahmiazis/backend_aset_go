package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// seedBasicMenuPermissions memberi sebuah menu daftar hak akses dasar
// (module "basic": read/write/delete) di tabel menu_permissions.
//
// Tanpa ini, menu yang baru dibuat tidak punya satu pun pilihan permission di
// matriks hak akses role — artinya menu tersebut TIDAK BISA di-assign sama
// sekali. Seed pada migrasi hanya menjangkau menu yang sudah ada saat itu.
func seedBasicMenuPermissions(tx *gorm.DB, menuID string) error {
	var basics []models.Permission
	if err := tx.Where("module = ?", models.PermissionModuleBasic).
		Find(&basics).Error; err != nil {
		return err
	}

	for _, p := range basics {
		var existing models.MenuPermission
		err := tx.Where("menu_id = ? AND permission_id = ?", menuID, p.ID).
			First(&existing).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&models.MenuPermission{
				MenuID:       menuID,
				PermissionID: p.ID,
			}).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	return nil
}

// expandWithAncestors melengkapi sekumpulan menu ID dengan seluruh menu
// induknya sampai level atas.
//
// Dipakai tampilan admin: response menu selalu berbentuk pohon yang berakar di
// menu level atas, jadi sebuah sub menu hanya ikut terbawa kalau induknya juga
// ada di daftar. Tanpa ini, sub menu yang di-assign sementara grup induknya
// tidak, akan hilang dari response — dan karena penyimpanan hak akses bersifat
// replace-all, hak akses itu ikut tercabut saat disimpan ulang.
func expandWithAncestors(menuIDs []string) ([]string, error) {
	seen := make(map[string]bool, len(menuIDs))
	result := make([]string, 0, len(menuIDs))
	for _, id := range menuIDs {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}

	// Kedalaman maksimal 3 tingkat; batas iterasi menjaga dari data siklik.
	frontier := result
	for depth := 0; depth < 5 && len(frontier) > 0; depth++ {
		var parentIDs []string
		if err := config.DB.Model(&models.Menu{}).
			Distinct().
			Where("id IN ? AND parent_id IS NOT NULL", frontier).
			Pluck("parent_id", &parentIDs).Error; err != nil {
			return nil, err
		}

		next := make([]string, 0, len(parentIDs))
		for _, pid := range parentIDs {
			if pid == "" || seen[pid] {
				continue
			}
			seen[pid] = true
			result = append(result, pid)
			next = append(next, pid)
		}
		frontier = next
	}

	return result, nil
}

// ─── Aturan nesting ──────────────────────────────────────────────────────────
//
// Kedalaman SIDEBAR dibatasi 2 tingkat: grup > menu.
// Menu bertipe permission tidak pernah dirender di sidebar, jadi ia tidak
// dihitung sebagai kedalaman dan boleh menempel di mana pun.

// visibleChildCount menghitung sub menu yang benar-benar tampil di sidebar
// (semua tipe kecuali permission).
func visibleChildCount(menuID string) (int64, error) {
	var count int64
	err := config.DB.Model(&models.Menu{}).
		Where("parent_id = ? AND menu_type <> ? AND deleted_at IS NULL",
			menuID, models.MenuTypePermission).
		Count(&count).Error
	return count, err
}

// validateNesting memeriksa apakah menu `menuID` (bertipe `menuType`) boleh
// dijadikan anak dari `parentID`. menuID boleh kosong saat pembuatan menu baru.
func validateNesting(menuID, menuType, parentID string) error {
	if parentID == "" {
		return nil
	}
	if menuID != "" && menuID == parentID {
		return errors.New("menu cannot be its own parent")
	}

	var parent models.Menu
	if err := config.DB.First(&parent, "id = ?", parentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("parent menu not found")
		}
		return err
	}

	// Menu permission tidak tampil di sidebar — boleh menempel pada menu apa pun,
	// termasuk yang sudah berada di dalam grup.
	if menuType == models.MenuTypePermission {
		return nil
	}

	// Selain itu, induknya wajib berada di level atas.
	if parent.ParentID != nil && *parent.ParentID != "" {
		return errors.New("maximum nesting level is 1 (menu > sub menu)")
	}

	// Dan menu ini tidak boleh punya sub menu yang tampil di sidebar.
	if menuID != "" {
		count, err := visibleChildCount(menuID)
		if err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf(
				"menu masih punya %d sub menu yang tampil di sidebar, pindahkan dulu sebelum menjadikannya sub menu", count)
		}
	}

	return nil
}

// ─── CRUD ────────────────────────────────────────────────────────────────────

// GetAllMenus — daftar menu untuk halaman admin.
// Sengaja TIDAK memfilter status: menu inactive harus tetap terlihat supaya
// bisa diaktifkan kembali dan ikut diatur urutannya. Filter "active" hanya
// berlaku untuk sidebar (GetSidebarMenus) dan hak akses role (GetRoleMenus).
func GetAllMenus() ([]dto.MenuResponse, error) {
	var menus []models.Menu

	// Preload bertingkat: menu bertipe permission boleh berada di kedalaman 3
	// (grup > menu > permission). Tanpa ini menu tsb hilang dari daftar admin
	// dan dari matriks hak akses, sehingga permission-nya bisa tercabut tanpa sengaja.
	if err := config.DB.
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Preload("Children.Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Where("parent_id IS NULL").
		Order("order_index ASC").
		Find(&menus).Error; err != nil {
		return nil, err
	}

	return mapMenusToResponse(menus), nil
}

func GetMenuByID(id string) (*dto.MenuResponse, error) {
	var menu models.Menu

	if err := config.DB.
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Preload("Children.Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		First(&menu, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("menu not found")
		}
		return nil, err
	}

	response := mapMenuToResponse(menu)
	return &response, nil
}

func CreateMenu(req dto.CreateMenuRequest) (*dto.MenuResponse, error) {
	status := req.Status
	if status == "" {
		status = "active"
	}

	menuType := req.MenuType
	if menuType == "" {
		// Tanpa path berarti tidak bisa dibuka sebagai halaman — anggap grup.
		if req.Path == nil || *req.Path == "" {
			menuType = models.MenuTypeGroup
		} else {
			menuType = models.MenuTypePage
		}
	}

	parentID := ""
	if req.ParentID != nil {
		parentID = *req.ParentID
	}
	if err := validateNesting("", menuType, parentID); err != nil {
		return nil, err
	}

	menu := models.Menu{
		ParentID:   req.ParentID,
		Name:       req.Name,
		MenuType:   menuType,
		Path:       req.Path,
		RoutePath:  req.RoutePath,
		IconName:   req.IconName,
		OrderIndex: req.OrderIndex,
		Status:     status,
	}

	if err := config.DB.Create(&menu).Error; err != nil {
		return nil, err
	}

	// Supaya menu ini langsung bisa diberi hak akses lewat halaman Role.
	// Kegagalan di sini tidak membatalkan pembuatan menu — hak aksesnya masih
	// bisa ditambahkan manual lewat PUT /menus/:id/permissions.
	if err := seedBasicMenuPermissions(config.DB, menu.ID); err != nil {
		fmt.Printf("warn: gagal seed hak akses dasar untuk menu %s: %v\n", menu.ID, err)
	}

	response := mapMenuToResponse(menu)
	return &response, nil
}

func UpdateMenu(id string, req dto.UpdateMenuRequest) (*dto.MenuResponse, error) {
	var menu models.Menu
	if err := config.DB.First(&menu, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("menu not found")
		}
		return nil, err
	}

	// Validate parent_id kalau diubah.
	// Tipe yang dipakai untuk validasi adalah tipe BARU kalau ikut diubah.
	if req.ParentID != nil && *req.ParentID != "" {
		effectiveType := menu.MenuType
		if req.MenuType != "" {
			effectiveType = req.MenuType
		}
		if err := validateNesting(menu.ID, effectiveType, *req.ParentID); err != nil {
			return nil, err
		}
	}

	updates := make(map[string]interface{})

	if req.ParentID != nil {
		if *req.ParentID == "" {
			updates["parent_id"] = nil
		} else {
			updates["parent_id"] = *req.ParentID
		}
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.MenuType != "" {
		updates["menu_type"] = req.MenuType
	}
	if req.Path != nil {
		updates["path"] = req.Path
	}
	if req.RoutePath != nil {
		updates["route_path"] = req.RoutePath
	}
	if req.IconName != nil {
		updates["icon_name"] = req.IconName
	}
	if req.OrderIndex != nil {
		updates["order_index"] = *req.OrderIndex
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	if err := config.DB.Model(&menu).Updates(updates).Error; err != nil {
		return nil, err
	}

	return GetMenuByID(id)
}

func DeleteMenu(id string) error {
	var menu models.Menu
	if err := config.DB.First(&menu, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("menu not found")
		}
		return err
	}

	return config.DB.Delete(&menu).Error
}

// ─── Role-Menu Assignment ────────────────────────────────────────────────────

// AssignMenusToRole mengganti SELURUH assignment menu+permission milik sebuah role.
// Menu yang tidak ada di request akan dicabut aksesnya — ini disengaja supaya UI
// matriks permission bisa mencabut akses hanya dengan menghilangkan centang.
func AssignMenusToRole(roleID string, req dto.AssignMenusRequest) error {
	// Validate role exists
	var role models.Role
	if err := config.DB.First(&role, "id = ?", roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("role not found")
		}
		return err
	}

	// Validate permissions terhadap katalog + tolak menu duplikat
	seenMenu := make(map[string]bool)
	menuIDs := make([]string, 0, len(req.Menus))
	for _, item := range req.Menus {
		if seenMenu[item.MenuID] {
			return fmt.Errorf("duplicate menu_id in request: %s", item.MenuID)
		}
		seenMenu[item.MenuID] = true

		if len(item.Permissions) == 0 {
			return fmt.Errorf("menu %s has no permissions", item.MenuID)
		}
		for _, p := range item.Permissions {
			// Sengaja tidak dibatasi daftar putih: selama development,
			// permission baru boleh ditambah langsung. Yang ditolak hanya
			// bentuk yang pasti gagal dicocokkan middleware.
			if !models.IsValidPermissionFormat(p) {
				return fmt.Errorf("permission tidak valid: %q", p)
			}
		}
		menuIDs = append(menuIDs, item.MenuID)
	}

	// Validate semua menu exists
	if len(menuIDs) > 0 {
		var count int64
		if err := config.DB.Model(&models.Menu{}).
			Where("id IN ?", menuIDs).
			Count(&count).Error; err != nil {
			return err
		}
		if int(count) != len(menuIDs) {
			return errors.New("one or more menus not found")
		}
	}

	// Replace dalam satu transaksi
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Cabut menu yang tidak lagi ada di request
	revoke := tx.Where("role_id = ?", roleID)
	if len(menuIDs) > 0 {
		revoke = revoke.Where("menu_id NOT IN ?", menuIDs)
	}
	if err := revoke.Delete(&models.RoleMenu{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Upsert sisanya
	for _, item := range req.Menus {
		var roleMenu models.RoleMenu
		err := tx.Where("role_id = ? AND menu_id = ?", roleID, item.MenuID).
			First(&roleMenu).Error

		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			roleMenu = models.RoleMenu{
				RoleID:      roleID,
				MenuID:      item.MenuID,
				Permissions: item.Permissions,
			}
			if err := tx.Create(&roleMenu).Error; err != nil {
				tx.Rollback()
				return err
			}
		case err == nil:
			permJSON, _ := json.Marshal(item.Permissions)
			if err := tx.Model(&roleMenu).
				Update("permissions", permJSON).Error; err != nil {
				tx.Rollback()
				return err
			}
		default:
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// ReorderMenus mengubah order_index (dan opsional parent_id) banyak menu sekaligus
// dalam satu transaksi, supaya urutan tidak pernah tersimpan setengah jalan.
func ReorderMenus(req dto.ReorderMenusRequest) error {
	ids := make([]string, 0, len(req.Menus))
	seen := make(map[string]bool)
	for _, item := range req.Menus {
		if seen[item.ID] {
			return fmt.Errorf("duplicate menu id in request: %s", item.ID)
		}
		seen[item.ID] = true
		ids = append(ids, item.ID)
	}

	// Pastikan semua menu ada
	var menus []models.Menu
	if err := config.DB.Where("id IN ?", ids).Find(&menus).Error; err != nil {
		return err
	}
	if len(menus) != len(ids) {
		return errors.New("one or more menus not found")
	}

	byID := make(map[string]models.Menu, len(menus))
	for _, m := range menus {
		byID[m.ID] = m
	}

	// Validasi parent memakai aturan nesting yang sama dengan Create/Update:
	// menu bertipe permission boleh menempel di mana pun, selain itu induknya
	// wajib top level dan menu tsb tidak boleh punya sub menu yang tampil.
	//
	// Perlu memperhitungkan perpindahan lain dalam request yang sama: sebuah
	// menu bisa jadi baru saja dipindah ke level atas pada request ini.
	newParentInRequest := make(map[string]string, len(req.Menus))
	for _, item := range req.Menus {
		if item.ParentID == nil {
			newParentInRequest[item.ID] = ""
		} else {
			newParentInRequest[item.ID] = *item.ParentID
		}
	}

	for _, item := range req.Menus {
		if item.ParentID == nil || *item.ParentID == "" {
			continue
		}

		menu, ok := byID[item.ID]
		if !ok {
			return fmt.Errorf("menu not found: %s", item.ID)
		}

		if err := validateNesting(item.ID, menu.MenuType, *item.ParentID); err != nil {
			return err
		}

		// Induknya ikut dipindah ke dalam grup lain pada request yang sama?
		// Menu permission dikecualikan karena tidak menambah kedalaman sidebar.
		if menu.MenuType != models.MenuTypePermission {
			if parentNewParent, moved := newParentInRequest[*item.ParentID]; moved && parentNewParent != "" {
				return errors.New("maximum nesting level is 1 (menu > sub menu)")
			}
		}
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, item := range req.Menus {
		updates := map[string]interface{}{
			"order_index": item.OrderIndex,
		}
		if item.ParentID == nil || *item.ParentID == "" {
			updates["parent_id"] = nil
		} else {
			updates["parent_id"] = *item.ParentID
		}

		if err := tx.Model(&models.Menu{}).
			Where("id = ?", item.ID).
			Updates(updates).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// GetRoleMenus ambil semua menu yang di-assign ke sebuah role, lengkap dengan permissions
func GetRoleMenus(roleID string) ([]dto.MenuResponse, error) {
	// Ambil role_menus untuk build permission map
	var roleMenus []models.RoleMenu
	if err := config.DB.Where("role_id = ?", roleID).Find(&roleMenus).Error; err != nil {
		return nil, err
	}

	// Build map: menu_id → permissions
	permissionMap := make(map[string][]string)
	menuIDs := make([]string, len(roleMenus))
	for i, rm := range roleMenus {
		permissionMap[rm.MenuID] = rm.Permissions
		menuIDs[i] = rm.MenuID
	}

	if len(menuIDs) == 0 {
		return []dto.MenuResponse{}, nil
	}

	// Lengkapi dengan menu induk supaya pohonnya utuh. Grup biasanya TIDAK
	// di-assign (tidak punya route_path), jadi tanpa langkah ini sub menu yang
	// sudah di-assign tidak akan ikut terbawa ke response.
	treeIDs, err := expandWithAncestors(menuIDs)
	if err != nil {
		return nil, err
	}

	// Ambil menus.
	// Sengaja TIDAK memfilter status maupun menu_type: ini tampilan admin untuk
	// matriks hak akses. Kalau difilter, permission pada menu inactive atau menu
	// bertipe permission tidak akan terbaca UI dan bisa tercabut saat disimpan.
	//
	// Menu induk yang ikut terbawa tapi tidak di-assign tidak punya entri di
	// permissionMap, jadi di UI tampil tanpa centang — persis seperti seharusnya.
	var menus []models.Menu
	if err := config.DB.
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("id IN ?", treeIDs).Order("order_index ASC")
		}).
		Preload("Children.Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("id IN ?", treeIDs).Order("order_index ASC")
		}).
		Where("id IN ? AND parent_id IS NULL", treeIDs).
		Order("order_index ASC").
		Find(&menus).Error; err != nil {
		return nil, err
	}

	return mapMenusToResponseWithPermissions(menus, permissionMap), nil
}

// ─── Sidebar (per user) ──────────────────────────────────────────────────────

// GetSidebarMenus ambil menu untuk sidebar berdasarkan roles user.
// Kalau user punya multiple roles dan role-role itu punya permissions berbeda di menu yang sama,
// permissions-nya di-merge (union). Misal: role A punya ["read"], role B punya ["read", "write"]
// → hasilnya ["read", "write"].
// GetSidebarMenus ambil menu untuk sidebar berdasarkan roles user.
func GetSidebarMenus(userID string) ([]dto.MenuResponse, error) {
	// 1. Ambil semua role IDs dari user
	var userRoles []models.UserRole
	if err := config.DB.Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, err
	}

	roleIDs := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roleIDs[i] = ur.RoleID
	}

	if len(roleIDs) == 0 {
		return []dto.MenuResponse{}, nil
	}

	// 2. Ambil semua role_menus dari role-role tersebut
	var roleMenus []models.RoleMenu
	if err := config.DB.Where("role_id IN ?", roleIDs).Find(&roleMenus).Error; err != nil {
		return nil, err
	}

	if len(roleMenus) == 0 {
		return []dto.MenuResponse{}, nil
	}

	// 3. Build permission map dengan merge dari semua roles
	permissionMap := make(map[string]map[string]bool)
	allMenuIDs := make(map[string]bool) // Pakai map untuk avoid duplicate

	for _, rm := range roleMenus {
		allMenuIDs[rm.MenuID] = true // Kumpulkan semua menu IDs (parent + children)

		if _, exists := permissionMap[rm.MenuID]; !exists {
			permissionMap[rm.MenuID] = make(map[string]bool)
		}
		for _, p := range rm.Permissions {
			permissionMap[rm.MenuID][p] = true
		}
	}

	// Convert map ke slice
	menuIDs := make([]string, 0, len(allMenuIDs))
	for id := range allMenuIDs {
		menuIDs = append(menuIDs, id)
	}

	// Convert set → slice untuk permissions
	mergedPermissions := make(map[string][]string)
	for menuID, permSet := range permissionMap {
		perms := make([]string, 0, len(permSet))
		for p := range permSet {
			perms = append(perms, p)
		}
		mergedPermissions[menuID] = perms
	}

	// 4. Sertakan menu induk (grup) dari setiap sub menu yang boleh diakses.
	//
	// Menu grup biasanya tidak diberi hak akses sendiri karena tidak punya
	// route_path — ia hanya wadah. Tanpa langkah ini, query di bawah (yang
	// mensyaratkan parent_id IS NULL AND id IN menuIDs) tidak akan menemukan
	// grupnya, sehingga sub menu yang sudah di-assign tidak pernah muncul.
	var parentIDs []string
	if err := config.DB.Model(&models.Menu{}).
		Distinct().
		Where("id IN ? AND parent_id IS NOT NULL AND menu_type <> ?",
			menuIDs, models.MenuTypePermission).
		Pluck("parent_id", &parentIDs).Error; err != nil {
		return nil, err
	}
	for _, pid := range parentIDs {
		if pid != "" && !allMenuIDs[pid] {
			allMenuIDs[pid] = true
			menuIDs = append(menuIDs, pid)
		}
	}

	// 5. Ambil menus + children
	// Children juga dicek apakah ada di menuIDs
	// Menu bertipe permission tidak pernah tampil di sidebar — ia hanya wadah
	// pemetaan route_path ke permission untuk middleware.
	var menus []models.Menu
	if err := config.DB.
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("id IN ? AND status = ? AND menu_type <> ?",
				menuIDs, "active", models.MenuTypePermission).
				Order("order_index ASC")
		}).
		Where("id IN ? AND parent_id IS NULL AND status = ? AND menu_type <> ?",
			menuIDs, "active", models.MenuTypePermission).
		Order("order_index ASC").
		Find(&menus).Error; err != nil {
		return nil, err
	}

	// Buang menu yang tidak bisa diklik DAN tidak punya sub menu terlihat.
	// Terjadi misalnya pada grup yang seluruh isinya bertipe permission: grupnya
	// lolos filter di atas, tapi anaknya tersaring habis sehingga yang tersisa
	// hanya label mati di sidebar.
	visible := make([]models.Menu, 0, len(menus))
	for _, m := range menus {
		hasPath := m.Path != nil && *m.Path != ""
		if !hasPath && len(m.Children) == 0 {
			continue
		}
		visible = append(visible, m)
	}

	// Diagnostik: kalau ada menu yang tidak muncul di sidebar, alasannya
	// tercetak di sini. Grup paling sering hilang karena sub menunya belum
	// di-assign ke role, sehingga grup itu kosong dan ikut dibuang.
	fmt.Printf("[sidebar] user=%s roles=%d assigned_menus=%d parents_added=%d\n",
		userID, len(roleIDs), len(allMenuIDs)-len(parentIDs), len(parentIDs))
	for _, m := range menus {
		hasPath := m.Path != nil && *m.Path != ""
		status := "TAMPIL"
		if !hasPath && len(m.Children) == 0 {
			status = "DIBUANG (tanpa path & tanpa sub menu yang di-assign)"
		}
		fmt.Printf("[sidebar]   %-24s type=%-10s children=%d  %s\n",
			m.Name, m.MenuType, len(m.Children), status)
	}
	if len(visible) == 0 {
		fmt.Printf("[sidebar]   (kosong — cek apakah role sudah punya menu di halaman Role)\n")
	}

	return mapMenusToResponseWithPermissions(visible, mergedPermissions), nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// mapMenuToResponse — untuk CRUD menus (admin view, tanpa permissions)
func mapMenuToResponse(menu models.Menu) dto.MenuResponse {
	response := dto.MenuResponse{
		ID:         menu.ID,
		ParentID:   menu.ParentID,
		Name:       menu.Name,
		MenuType:   menu.MenuType,
		Path:       menu.Path,
		RoutePath:  menu.RoutePath,
		IconName:   menu.IconName,
		OrderIndex: menu.OrderIndex,
		Status:     menu.Status,
	}

	if len(menu.Children) > 0 {
		response.Children = mapMenusToResponse(menu.Children)
	}

	return response
}

func mapMenusToResponse(menus []models.Menu) []dto.MenuResponse {
	response := make([]dto.MenuResponse, len(menus))
	for i, menu := range menus {
		response[i] = mapMenuToResponse(menu)
	}
	return response
}

// mapMenuToResponseWithPermissions — untuk sidebar & role menus (include permissions)
func mapMenuToResponseWithPermissions(menu models.Menu, permissionMap map[string][]string) dto.MenuResponse {
	response := dto.MenuResponse{
		ID:          menu.ID,
		ParentID:    menu.ParentID,
		Name:        menu.Name,
		MenuType:    menu.MenuType,
		Path:        menu.Path,
		RoutePath:   menu.RoutePath,
		IconName:    menu.IconName,
		OrderIndex:  menu.OrderIndex,
		Status:      menu.Status,
		Permissions: permissionMap[menu.ID],
	}

	if len(menu.Children) > 0 {
		response.Children = mapMenusToResponseWithPermissions2(menu.Children, permissionMap)
	}

	return response
}

func mapMenusToResponseWithPermissions(menus []models.Menu, permissionMap map[string][]string) []dto.MenuResponse {
	response := make([]dto.MenuResponse, len(menus))
	for i, menu := range menus {
		response[i] = mapMenuToResponseWithPermissions(menu, permissionMap)
	}
	return response
}

// helper untuk children (sama logicnya)
func mapMenusToResponseWithPermissions2(menus []models.Menu, permissionMap map[string][]string) []dto.MenuResponse {
	return mapMenusToResponseWithPermissions(menus, permissionMap)
}
