package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAllBranchs - GET /branchs
func GetAllBranchs(c *gin.Context) {
	branchs, err := services.GetAllBranchs()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Branchs retrieved successfully", branchs)
}

// GetBranchByID - GET /branchs/:id
func GetBranchByID(c *gin.Context) {
	id := c.Param("id")

	branch, err := services.GetBranchByID(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Branch retrieved successfully", branch)
}

// CreateBranch - POST /branchs
func CreateBranch(c *gin.Context) {
	var req dto.CreateBranchRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	branch, err := services.CreateBranch(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Branch created successfully", branch)
}

// UpdateBranch - PUT /branchs/:id
func UpdateBranch(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	branch, err := services.UpdateBranch(id, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Branch updated successfully", branch)
}

// DeleteBranch - DELETE /branchs/:id
func DeleteBranch(c *gin.Context) {
	id := c.Param("id")

	if err := services.DeleteBranch(id); err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Branch deleted successfully", nil)
}

// AssignBranchsToUser - POST /users/:id/branchs
func AssignBranchsToUser(c *gin.Context) {
	userID := c.Param("id")

	var req dto.AssignBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.AssignBranchsToUser(userID, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Branchs assigned successfully", nil)
}

// GetUserBranchs - GET /users/:id/branchs
func GetUserBranchs(c *gin.Context) {
	userID := c.Param("id")

	branchs, err := services.GetUserBranchs(userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "User branchs retrieved successfully", branchs)
}

// GetBranchUsers - GET /branchs/:id/users
func GetBranchUsers(c *gin.Context) {
	branchID := c.Param("id")

	users, err := services.GetBranchUsers(branchID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Branch users retrieved successfully", users)
}

// RemoveBranchFromUser - DELETE /users/:id/branchs/:branch_id
func RemoveBranchFromUser(c *gin.Context) {
	userID := c.Param("id")
	branchID := c.Param("branch_id")

	if err := services.RemoveBranchFromUser(userID, branchID); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Branch removed from user successfully", nil)
}
