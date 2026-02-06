package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAllMenus - GET /menus
func GetAllMenus(c *gin.Context) {
	menus, err := services.GetAllMenus()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Menus retrieved successfully", menus)
}

// GetMenuByID - GET /menus/:id
func GetMenuByID(c *gin.Context) {
	id := c.Param("id")

	menu, err := services.GetMenuByID(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Menu retrieved successfully", menu)
}

// CreateMenu - POST /menus
func CreateMenu(c *gin.Context) {
	var req dto.CreateMenuRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	menu, err := services.CreateMenu(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Menu created successfully", menu)
}

// UpdateMenu - PUT /menus/:id
func UpdateMenu(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	menu, err := services.UpdateMenu(id, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Menu updated successfully", menu)
}

// DeleteMenu - DELETE /menus/:id
func DeleteMenu(c *gin.Context) {
	id := c.Param("id")

	if err := services.DeleteMenu(id); err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Menu deleted successfully", nil)
}

// AssignMenusToRole - POST /roles/:id/menus
func AssignMenusToRole(c *gin.Context) {
	roleID := c.Param("id")

	var req dto.AssignMenusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.AssignMenusToRole(roleID, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Menus assigned to role successfully", nil)
}

// GetRoleMenus - GET /roles/:id/menus
func GetRoleMenus(c *gin.Context) {
	roleID := c.Param("id")

	menus, err := services.GetRoleMenus(roleID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Role menus retrieved successfully", menus)
}

// GetSidebarMenus - GET /menus/sidebar
// Ambil menu sidebar berdasarkan user yang sedang login
func GetSidebarMenus(c *gin.Context) {
	fmt.Println("=== GetSidebarMenus CONTROLLER CALLED ===")

	userID := c.GetString("user_id")
	fmt.Printf("User ID from context: '%s'\n", userID)

	if userID == "" {
		fmt.Println("ERROR: User ID is empty!")
		utils.ErrorResponse(c, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	menus, err := services.GetSidebarMenus(userID)
	if err != nil {
		fmt.Printf("ERROR from service: %v\n", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	fmt.Printf("Menus retrieved: %d items\n", len(menus))
	utils.SuccessResponse(c, http.StatusOK, "Sidebar menus retrieved successfully", menus)
}
