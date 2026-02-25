package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateStockOpname(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.CreateStockOpnameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	stockOpname, err := services.CreateStockOpname(userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Stock opname created successfully", stockOpname)
}

func GetStockOpnameByNumber(c *gin.Context) {
	transactionNumber := c.Param("number")

	stockOpname, err := services.GetStockOpnameByTransactionNumber(transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opname retrieved successfully", stockOpname)
}

func GetAllStockOpnames(c *gin.Context) {
	var filter dto.TransactionListFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 10
	}

	stockOpnames, total, err := services.GetAllStockOpnames(filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"data":  stockOpnames,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opnames retrieved successfully", response)
}

func UpdateStockOpname(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Param("number")

	var req dto.CreateStockOpnameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	stockOpname, err := services.UpdateStockOpname(transactionNumber, userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opname updated successfully", stockOpname)
}

func DeleteStockOpname(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Param("number")

	if err := services.DeleteStockOpname(transactionNumber, userID); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opname deleted successfully", nil)
}
