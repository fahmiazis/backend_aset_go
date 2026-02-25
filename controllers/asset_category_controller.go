package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetAllAssetCategories(c *gin.Context) {
	categories, err := services.GetAllAssetCategories()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset categories retrieved successfully", categories)
}

func GetAssetCategoryByID(c *gin.Context) {
	id := c.Param("id")

	category, err := services.GetAssetCategoryByID(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset category retrieved successfully", category)
}

func CreateAssetCategory(c *gin.Context) {
	var req dto.CreateAssetCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	category, err := services.CreateAssetCategory(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Asset category created successfully", category)
}

func UpdateAssetCategory(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateAssetCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	category, err := services.UpdateAssetCategory(id, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset category updated successfully", category)
}

func DeleteAssetCategory(c *gin.Context) {
	id := c.Param("id")

	if err := services.DeleteAssetCategory(id); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset category deleted successfully", nil)
}
