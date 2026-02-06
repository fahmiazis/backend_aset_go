package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAllRoles - GET /roles
func GetAllRoles(c *gin.Context) {
	roles, err := services.GetAllRoles()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Roles retrieved successfully", roles)
}

// GetRoleByID - GET /roles/:id
func GetRoleByID(c *gin.Context) {
	id := c.Param("id")

	role, err := services.GetRoleByID(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Role retrieved successfully", role)
}

// CreateRole - POST /roles
func CreateRole(c *gin.Context) {
	var req dto.CreateRoleRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	role, err := services.CreateRole(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Role created successfully", role)
}

// UpdateRole - PUT /roles/:id
func UpdateRole(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	role, err := services.UpdateRole(id, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Role updated successfully", role)
}

// DeleteRole - DELETE /roles/:id
func DeleteRole(c *gin.Context) {
	id := c.Param("id")

	if err := services.DeleteRole(id); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Role deleted successfully", nil)
}
