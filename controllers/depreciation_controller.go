package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Depreciation Settings
// ============================================================================

func GetAllDepreciationSettings(c *gin.Context) {
	settings, err := services.GetAllDepreciationSettings()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Depreciation settings retrieved successfully", settings)
}

func GetDepreciationSettingByID(c *gin.Context) {
	// FIX: parse id string -> uint sesuai signature service
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}

	setting, err := services.GetDepreciationSettingByID(uint(id))
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Depreciation setting retrieved successfully", setting)
}

func CreateDepreciationSetting(c *gin.Context) {
	var req dto.CreateDepreciationSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	setting, err := services.CreateDepreciationSetting(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Depreciation setting created successfully", setting)
}

func UpdateDepreciationSetting(c *gin.Context) {
	// FIX: parse id string -> uint sesuai signature service
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req dto.UpdateDepreciationSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	setting, err := services.UpdateDepreciationSetting(uint(id), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Depreciation setting updated successfully", setting)
}

func DeleteDepreciationSetting(c *gin.Context) {
	// FIX: parse id string -> uint sesuai signature service
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := services.DeleteDepreciationSetting(uint(id)); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Depreciation setting deleted successfully", nil)
}

// ============================================================================
// Monthly Depreciation
// ============================================================================

func GetMonthlyDepreciationCalculations(c *gin.Context) {
	// FIX: query param diganti dari month+year -> period "YYYY-MM"
	period := c.Query("period")
	if period == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "period is required (format: YYYY-MM)")
		return
	}

	calculations, err := services.GetMonthlyDepreciationCalculations(period)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Monthly depreciation calculations retrieved successfully", calculations)
}

func CalculateMonthlyDepreciation(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.CalculateDepreciationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.CalculateMonthlyDepreciation(userID, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Monthly depreciation calculated successfully", nil)
}

func LockMonthlyDepreciation(c *gin.Context) {
	period := c.Query("period")
	if period == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "period is required (format: YYYY-MM)")
		return
	}

	if err := services.LockMonthlyDepreciation(period); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Monthly depreciation locked successfully for period "+period, nil)
}
