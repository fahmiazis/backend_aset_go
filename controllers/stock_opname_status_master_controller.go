package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ---------------- PHYSICAL STATUS ----------------

func GetAllStockOpnamePhysicalStatusMasters(c *gin.Context) {
	rows, err := services.GetAllStockOpnamePhysicalStatusMasters()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Stock opname physical status masters retrieved successfully", rows)
}

func CreateStockOpnamePhysicalStatusMaster(c *gin.Context) {
	var req dto.CreateStockOpnamePhysicalStatusMasterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	row, err := services.CreateStockOpnamePhysicalStatusMaster(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "Stock opname physical status master created successfully", row)
}

func DeleteStockOpnamePhysicalStatusMaster(c *gin.Context) {
	id := c.Param("id")
	if err := services.DeleteStockOpnamePhysicalStatusMaster(id); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Stock opname physical status master deleted successfully", nil)
}

// ---------------- CONDITION ----------------

func GetAllStockOpnameConditionMasters(c *gin.Context) {
	rows, err := services.GetAllStockOpnameConditionMasters()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Stock opname condition masters retrieved successfully", rows)
}

func CreateStockOpnameConditionMaster(c *gin.Context) {
	var req dto.CreateStockOpnameConditionMasterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	row, err := services.CreateStockOpnameConditionMaster(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "Stock opname condition master created successfully", row)
}

func DeleteStockOpnameConditionMaster(c *gin.Context) {
	id := c.Param("id")
	if err := services.DeleteStockOpnameConditionMaster(id); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Stock opname condition master deleted successfully", nil)
}
