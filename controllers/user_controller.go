package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAllUsers - GET /users
func GetAllUsers(c *gin.Context) {
	users, err := services.GetAllUsers()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Users retrieved successfully", users)
}

// GetUserByID - GET /users/:id
func GetUserByID(c *gin.Context) {
	id := c.Param("id")

	user, err := services.GetUserByID(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "User retrieved successfully", user)
}

// CreateUser - POST /users
func CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	// Convert CreateUserRequest to RegisterRequest
	registerReq := dto.RegisterRequest{
		Username:  req.Username,
		Fullname:  req.Fullname,
		Email:     req.Email,
		Password:  req.Password,
		NIK:       req.NIK,
		MPNNumber: req.MPNNumber,
	}

	user, err := services.Register(registerReq)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// If status is provided, update it
	if req.Status != "" {
		updateReq := dto.UpdateUserRequest{
			Status: req.Status,
		}
		services.UpdateUser(user.ID, updateReq)
	}

	if len(req.RoleIDs) > 0 {
		assignReq := dto.AssignRoleRequest{
			RoleIDs: req.RoleIDs,
		}
		services.AssignRoles(user.ID, assignReq)
	}

	utils.SuccessResponse(c, http.StatusCreated, "User created successfully", gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

// UpdateUser - PUT /users/:id
func UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	user, err := services.UpdateUser(id, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "User updated successfully", user)
}

// DeleteUser - DELETE /users/:id
func DeleteUser(c *gin.Context) {
	id := c.Param("id")

	if err := services.DeleteUser(id); err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "User deleted successfully", nil)
}

// AssignRoles - POST /users/:id/roles
func AssignRoles(c *gin.Context) {
	id := c.Param("id")

	var req dto.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.AssignRoles(id, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Roles assigned successfully", nil)
}
