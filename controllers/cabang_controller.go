package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAllCabangs - GET /cabangs
func GetAllCabangs(c *gin.Context) {
	cabangs, err := services.GetAllCabangs()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Cabangs retrieved successfully", cabangs)
}

// GetCabangByID - GET /cabangs/:id
func GetCabangByID(c *gin.Context) {
	id := c.Param("id")

	cabang, err := services.GetCabangByID(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Cabang retrieved successfully", cabang)
}

// CreateCabang - POST /cabangs
func CreateCabang(c *gin.Context) {
	var req dto.CreateCabangRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	cabang, err := services.CreateCabang(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Cabang created successfully", cabang)
}

// UpdateCabang - PUT /cabangs/:id
func UpdateCabang(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateCabangRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	cabang, err := services.UpdateCabang(id, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Cabang updated successfully", cabang)
}

// DeleteCabang - DELETE /cabangs/:id
func DeleteCabang(c *gin.Context) {
	id := c.Param("id")

	if err := services.DeleteCabang(id); err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Cabang deleted successfully", nil)
}

// AssignCabangsToUser - POST /users/:id/cabangs
func AssignCabangsToUser(c *gin.Context) {
	userID := c.Param("id")

	var req dto.AssignCabangRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.AssignCabangsToUser(userID, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Cabangs assigned successfully", nil)
}

// GetUserCabangs - GET /users/:id/cabangs
func GetUserCabangs(c *gin.Context) {
	userID := c.Param("id")

	cabangs, err := services.GetUserCabangs(userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "User cabangs retrieved successfully", cabangs)
}

// GetCabangUsers - GET /cabangs/:id/users
func GetCabangUsers(c *gin.Context) {
	cabangID := c.Param("id")

	users, err := services.GetCabangUsers(cabangID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Cabang users retrieved successfully", users)
}

// RemoveCabangFromUser - DELETE /users/:id/cabangs/:cabang_id
func RemoveCabangFromUser(c *gin.Context) {
	userID := c.Param("id")
	cabangID := c.Param("cabang_id")

	if err := services.RemoveCabangFromUser(userID, cabangID); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Cabang removed from user successfully", nil)
}
