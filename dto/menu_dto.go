package dto

// CreateMenuRequest represents the request body for creating a menu
type CreateMenuRequest struct {
	ParentID   *string `json:"parent_id"` // NULL kalau top level
	Name       string  `json:"name" binding:"required,min=2,max=100"`
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
	Path        *string        `json:"path"`
	IconName    *string        `json:"icon_name"`
	OrderIndex  int            `json:"order_index"`
	Status      string         `json:"status"`
	Permissions []string       `json:"permissions,omitempty"` // dari role_menus, hanya ada di sidebar & role menus response
	Children    []MenuResponse `json:"children,omitempty"`
}

// RoleMenuItem represents one menu + permissions saat assign ke role
type RoleMenuItem struct {
	MenuID      string   `json:"menu_id" binding:"required"`
	Permissions []string `json:"permissions" binding:"required,min=1,dive,oneof=read write delete"`
}

// AssignMenusRequest represents the request to assign menus to a role
type AssignMenusRequest struct {
	Menus []RoleMenuItem `json:"menus" binding:"required,min=1"`
}

// RoleMenuResponse represents a role with its menus (for admin view)
type RoleMenuResponse struct {
	RoleID   string         `json:"role_id"`
	RoleName string         `json:"role_name"`
	Menus    []MenuResponse `json:"menus"`
}
