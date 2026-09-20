package dto

// CreateMenuRequest represents the request body for creating a menu
type CreateMenuRequest struct {
	ParentID   *string `json:"parent_id"` // NULL kalau top level
	Name       string  `json:"name" binding:"required,min=2,max=100"`
	MenuType   string  `json:"menu_type" binding:"omitempty,oneof=page group permission"`
	Path       *string `json:"path"`       // frontend route path
	RoutePath  *string `json:"route_path"` // backend API route path
	IconName   *string `json:"icon_name"`  // nama icon lucide-react
	OrderIndex int     `json:"order_index"`
	Status     string  `json:"status" binding:"omitempty,oneof=active inactive"`
}

// UpdateMenuRequest represents the request body for updating a menu
type UpdateMenuRequest struct {
	ParentID   *string `json:"parent_id"`
	Name       string  `json:"name" binding:"omitempty,min=2,max=100"`
	MenuType   string  `json:"menu_type" binding:"omitempty,oneof=page group permission"`
	Path       *string `json:"path"`
	RoutePath  *string `json:"route_path"`
	IconName   *string `json:"icon_name"`
	OrderIndex *int    `json:"order_index"` // pointer biar bisa update ke 0
	Status     string  `json:"status" binding:"omitempty,oneof=active inactive"`
}

// MenuResponse represents a single menu item in response
type MenuResponse struct {
	ID          string         `json:"id"`
	ParentID    *string        `json:"parent_id"`
	Name        string         `json:"name"`
	MenuType    string         `json:"menu_type"`
	Path        *string        `json:"path"`
	RoutePath   *string        `json:"route_path"`
	IconName    *string        `json:"icon_name"`
	OrderIndex  int            `json:"order_index"`
	Status      string         `json:"status"`
	Permissions []string       `json:"permissions,omitempty"` // dari role_menus, hanya ada di sidebar & role menus response
	Children    []MenuResponse `json:"children,omitempty"`
}

// RoleMenuItem represents one menu + permissions saat assign ke role.
// Permission divalidasi terhadap models.PermissionCatalog di service layer,
// bukan lewat tag `oneof` — supaya daftar permission cukup dikelola di satu tempat.
type RoleMenuItem struct {
	MenuID      string   `json:"menu_id" binding:"required"`
	Permissions []string `json:"permissions" binding:"required"`
}

// AssignMenusRequest represents the request to assign menus to a role.
// Menus boleh kosong — artinya semua akses role tersebut dicabut.
type AssignMenusRequest struct {
	Menus []RoleMenuItem `json:"menus"`
}

// PermissionResponse — satu hak akses
type PermissionResponse struct {
	ID          string `json:"id"`
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Module      string `json:"module"`
}

// PermissionGroupResponse — hak akses dikelompokkan per module
type PermissionGroupResponse struct {
	Key         string               `json:"key"`
	Label       string               `json:"label"`
	Permissions []PermissionResponse `json:"permissions"`
}

// MenuPermissionOptions — permission yang relevan untuk satu menu,
// diambil dari tabel menu_permissions.
type MenuPermissionOptions struct {
	MenuID    string   `json:"menu_id"`
	RoutePath *string  `json:"route_path"`
	Values    []string `json:"values"`
}

// PermissionCatalogResponse — seluruh permission + pemetaannya ke menu.
// `groups` dipakai untuk menampilkan label/deskripsi, `menus` menentukan
// permission mana yang ditawarkan pada tiap menu.
type PermissionCatalogResponse struct {
	Groups []PermissionGroupResponse `json:"groups"`
	Menus  []MenuPermissionOptions   `json:"menus"`
}

// AddPermissionRequest — tambah permission baru (mode development).
// Kalau MenuID diisi, permission langsung dipetakan ke menu tersebut.
type AddPermissionRequest struct {
	Value       string  `json:"value" binding:"required"`
	Label       string  `json:"label"`
	Description string  `json:"description"`
	Module      string  `json:"module"`
	MenuID      *string `json:"menu_id"`
}

// MapPermissionsToMenuRequest — set permission apa saja yang relevan untuk sebuah menu
type MapPermissionsToMenuRequest struct {
	Permissions []string `json:"permissions"`
}

// ReorderMenuItem — satu baris dalam request reorder
type ReorderMenuItem struct {
	ID         string  `json:"id" binding:"required"`
	ParentID   *string `json:"parent_id"`
	OrderIndex int     `json:"order_index"`
}

// ReorderMenusRequest — ubah urutan (dan parent) banyak menu sekaligus
type ReorderMenusRequest struct {
	Menus []ReorderMenuItem `json:"menus" binding:"required,min=1,dive"`
}

// RoleMenuResponse represents a role with its menus (for admin view)
type RoleMenuResponse struct {
	RoleID   string         `json:"role_id"`
	RoleName string         `json:"role_name"`
	Menus    []MenuResponse `json:"menus"`
}
