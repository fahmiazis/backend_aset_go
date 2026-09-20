package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ============================================================
// PERMISSION SERVICE
//
// Sumber data: tabel `permissions` (master hak akses) dan
// `menu_permissions` (hak akses mana yang relevan untuk tiap menu).
// Keduanya dibuat lewat migrations/20260501000001 & ...002.
// ============================================================

func mapPermissionToResponse(p models.Permission) dto.PermissionResponse {
	description := ""
	if p.Description != nil {
		description = *p.Description
	}
	module := ""
	if p.Module != nil {
		module = *p.Module
	}

	return dto.PermissionResponse{
		ID:          p.ID,
		Value:       p.Value,
		Label:       p.Label,
		Description: description,
		Module:      module,
	}
}

// GetPermissionCatalog mengembalikan seluruh permission (dikelompokkan per module)
// beserta pemetaan permission → menu.
func GetPermissionCatalog() (*dto.PermissionCatalogResponse, error) {
	var permissions []models.Permission
	if err := config.DB.Order("module ASC, label ASC").Find(&permissions).Error; err != nil {
		return nil, err
	}

	// Kelompokkan per module
	byModule := make(map[string][]dto.PermissionResponse)
	for _, p := range permissions {
		module := ""
		if p.Module != nil {
			module = *p.Module
		}
		byModule[module] = append(byModule[module], mapPermissionToResponse(p))
	}

	groups := make([]dto.PermissionGroupResponse, 0, len(byModule))
	seen := make(map[string]bool)

	// Module yang dikenal dulu, sesuai urutan yang ditentukan
	for _, module := range models.PermissionModuleOrder {
		if items, ok := byModule[module]; ok {
			groups = append(groups, dto.PermissionGroupResponse{
				Key:         module,
				Label:       models.PermissionModuleLabels[module],
				Permissions: items,
			})
			seen[module] = true
		}
	}

	// Sisanya (module custom yang ditambah sendiri lewat DB)
	for module, items := range byModule {
		if seen[module] {
			continue
		}
		label := module
		if label == "" {
			label = "Lainnya"
		}
		groups = append(groups, dto.PermissionGroupResponse{
			Key:         module,
			Label:       label,
			Permissions: items,
		})
	}

	// Pemetaan permission per menu
	type menuPermissionRow struct {
		MenuID          string
		RoutePath       *string
		PermissionValue string
	}

	var rows []menuPermissionRow
	if err := config.DB.
		Table("menu_permissions AS mp").
		Select("mp.menu_id AS menu_id, m.route_path AS route_path, p.value AS permission_value").
		Joins("JOIN menus m ON m.id = mp.menu_id AND m.deleted_at IS NULL").
		Joins("JOIN permissions p ON p.id = mp.permission_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	order := make([]string, 0)
	grouped := make(map[string]*dto.MenuPermissionOptions)
	for _, row := range rows {
		entry, ok := grouped[row.MenuID]
		if !ok {
			entry = &dto.MenuPermissionOptions{
				MenuID:    row.MenuID,
				RoutePath: row.RoutePath,
				Values:    []string{},
			}
			grouped[row.MenuID] = entry
			order = append(order, row.MenuID)
		}
		entry.Values = append(entry.Values, row.PermissionValue)
	}

	menus := make([]dto.MenuPermissionOptions, 0, len(order))
	for _, menuID := range order {
		menus = append(menus, *grouped[menuID])
	}

	return &dto.PermissionCatalogResponse{Groups: groups, Menus: menus}, nil
}

// AddPermission menambah permission baru ke master (mode development).
// Kalau MenuID diisi, permission langsung dipetakan ke menu tersebut.
func AddPermission(req dto.AddPermissionRequest) (*dto.PermissionResponse, error) {
	value := strings.TrimSpace(req.Value)
	if !models.IsValidPermissionFormat(value) {
		return nil, errors.New("permission tidak valid: tidak boleh kosong, mengandung spasi, atau lebih dari 100 karakter")
	}

	var permission models.Permission
	err := config.DB.Where("value = ?", value).First(&permission).Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		label := strings.TrimSpace(req.Label)
		if label == "" {
			label = value
		}

		permission = models.Permission{Value: value, Label: label}
		if req.Description != "" {
			d := req.Description
			permission.Description = &d
		}
		if req.Module != "" {
			m := req.Module
			permission.Module = &m
		}

		if err := config.DB.Create(&permission).Error; err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	}

	// Petakan ke menu kalau diminta
	if req.MenuID != nil && *req.MenuID != "" {
		var menu models.Menu
		if err := config.DB.First(&menu, "id = ?", *req.MenuID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("menu not found")
			}
			return nil, err
		}

		var existing models.MenuPermission
		err := config.DB.
			Where("menu_id = ? AND permission_id = ?", menu.ID, permission.ID).
			First(&existing).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := config.DB.Create(&models.MenuPermission{
				MenuID:       menu.ID,
				PermissionID: permission.ID,
			}).Error; err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
	}

	response := mapPermissionToResponse(permission)
	return &response, nil
}

// SetMenuPermissions mengganti daftar permission yang relevan untuk sebuah menu.
// Permission yang belum ada di master akan dibuat otomatis (mode development),
// supaya bisa langsung dipakai tanpa migrasi baru.
func SetMenuPermissions(menuID string, values []string) error {
	var menu models.Menu
	if err := config.DB.First(&menu, "id = ?", menuID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("menu not found")
		}
		return err
	}

	cleaned := make([]string, 0, len(values))
	seen := make(map[string]bool)
	for _, v := range values {
		value := strings.TrimSpace(v)
		if value == "" || seen[value] {
			continue
		}
		if !models.IsValidPermissionFormat(value) {
			return fmt.Errorf("permission tidak valid: %q", v)
		}
		seen[value] = true
		cleaned = append(cleaned, value)
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	permissionIDs := make([]string, 0, len(cleaned))
	for _, value := range cleaned {
		var permission models.Permission
		err := tx.Where("value = ?", value).First(&permission).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			permission = models.Permission{Value: value, Label: value}
			if err := tx.Create(&permission).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else if err != nil {
			tx.Rollback()
			return err
		}

		permissionIDs = append(permissionIDs, permission.ID)
	}

	// Buang mapping yang tidak lagi dipilih
	remove := tx.Where("menu_id = ?", menuID)
	if len(permissionIDs) > 0 {
		remove = remove.Where("permission_id NOT IN ?", permissionIDs)
	}
	if err := remove.Delete(&models.MenuPermission{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Tambahkan yang belum ada
	for _, permissionID := range permissionIDs {
		var existing models.MenuPermission
		err := tx.Where("menu_id = ? AND permission_id = ?", menuID, permissionID).
			First(&existing).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&models.MenuPermission{
				MenuID:       menuID,
				PermissionID: permissionID,
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}
