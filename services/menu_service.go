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

// ─── CRUD ────────────────────────────────────────────────────────────────────

func GetAllMenus() ([]dto.MenuResponse, error) {
	var menus []models.Menu

	// Ambil semua menu yang top level (parent_id IS NULL), order by order_index
	if err := config.DB.
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", "active").Order("order_index ASC")
		}).
		Where("parent_id IS NULL AND status = ?", "active").
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
	// Validate parent_id kalau ada
	if req.ParentID != nil && *req.ParentID != "" {
		var parent models.Menu
		if err := config.DB.First(&parent, "id = ?", *req.ParentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.New("parent menu not found")
			}
			return nil, err
		}
		// Cegah nesting lebih dari 1 level (parent tidak boleh punya parent)
		if parent.ParentID != nil {
			return nil, errors.New("maximum nesting level is 1 (menu > sub menu)")
		}
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	menu := models.Menu{
		ParentID:   req.ParentID,
		Name:       req.Name,
		Path:       req.Path,
		RoutePath:  req.RoutePath,
		IconName:   req.IconName,
		OrderIndex: req.OrderIndex,
		Status:     status,
	}

	if err := config.DB.Create(&menu).Error; err != nil {
		return nil, err
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

	// Validate parent_id kalau diubah
	if req.ParentID != nil && *req.ParentID != "" {
		var parent models.Menu
		if err := config.DB.First(&parent, "id = ?", *req.ParentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.New("parent menu not found")
			}
			return nil, err
		}
		if parent.ParentID != nil {
			return nil, errors.New("maximum nesting level is 1 (menu > sub menu)")
		}
	}

	updates := make(map[string]interface{})

	if req.ParentID != nil {
		updates["parent_id"] = req.ParentID
	}
	if req.Name != "" {
		updates["name"] = req.Name
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

// AssignMenusToRole assign menu list + permissions ke sebuah role (replace semua)
func AssignMenusToRole(roleID string, req dto.AssignMenusRequest) error {
	// Validate role exists
	var role models.Role
	if err := config.DB.First(&role, "id = ?", roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("role not found")
		}
		return err
	}

	// Collect menu IDs dari request
	menuIDs := make([]string, len(req.Menus))
	for i, item := range req.Menus {
		menuIDs[i] = item.MenuID
	}

	// Validate semua menu exists
	var menus []models.Menu
	if err := config.DB.Where("id IN ?", menuIDs).Find(&menus).Error; err != nil {
		return err
	}
	if len(menus) != len(menuIDs) {
		return errors.New("one or more menus not found")
	}

	// Create new assignments dengan permissions
	for _, item := range req.Menus {

		var roleMenu models.RoleMenu

		err := config.DB.
			Where("role_id = ? AND menu_id = ?", roleID, item.MenuID).
			First(&roleMenu).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			roleMenu = models.RoleMenu{
				RoleID:      roleID,
				MenuID:      item.MenuID,
				Permissions: item.Permissions,
			}

			if err := config.DB.Create(&roleMenu).Error; err != nil {
				return err
			}

		} else if err == nil {
			permJSON, _ := json.Marshal(item.Permissions)

			updateFields := map[string]any{
				"permissions": permJSON,
			}

			if err := config.DB.
				Model(&roleMenu).
				Updates(updateFields).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
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

	// Ambil menus
	var menus []models.Menu
	if err := config.DB.
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("id IN ? AND status = ?", menuIDs, "active").Order("order_index ASC")
		}).
		Where("id IN ? AND parent_id IS NULL AND status = ?", menuIDs, "active").
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
	fmt.Println("=== GetSidebarMenus CALLED ===")
	fmt.Printf("Received User ID: '%s'\n", userID)
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

	// 4. Ambil menus + children
	// PERBAIKAN: Children juga harus dicek apakah ada di menuIDs
	var menus []models.Menu
	if err := config.DB.
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			// Filter children yang ada di menuIDs dan active
			return db.Where("id IN ? AND status = ?", menuIDs, "active").Order("order_index ASC")
		}).
		Where("id IN ? AND parent_id IS NULL AND status = ?", menuIDs, "active").
		Order("order_index ASC").
		Find(&menus).Error; err != nil {
		return nil, err
	}

	fmt.Printf("User ID: %s\n", userID)
	fmt.Printf("Role IDs: %v\n", roleIDs)
	fmt.Printf("Menu IDs found: %v\n", menuIDs)
	fmt.Printf("Total menus retrieved: %d\n", len(menus))

	return mapMenusToResponseWithPermissions(menus, mergedPermissions), nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// mapMenuToResponse — untuk CRUD menus (admin view, tanpa permissions)
func mapMenuToResponse(menu models.Menu) dto.MenuResponse {
	response := dto.MenuResponse{
		ID:         menu.ID,
		ParentID:   menu.ParentID,
		Name:       menu.Name,
		Path:       menu.Path,
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
		Path:        menu.Path,
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
