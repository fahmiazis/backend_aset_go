package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateProcurement(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.CreateProcurementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	procurement, err := services.CreateProcurement(userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Procurement created successfully", procurement)
}

func GetProcurementByNumber(c *gin.Context) {
	transactionNumber := c.Param("number")

	procurement, err := services.GetProcurementByTransactionNumber(transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Procurement retrieved successfully", procurement)
}

func GetAllProcurements(c *gin.Context) {
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

	procurements, total, err := services.GetAllProcurements(filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"data":  procurements,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	}

	utils.SuccessResponse(c, http.StatusOK, "Procurements retrieved successfully", response)
}

func UpdateProcurement(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Param("number")

	var req dto.CreateProcurementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	procurement, err := services.UpdateProcurement(transactionNumber, userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Procurement updated successfully", procurement)
}

func DeleteProcurement(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Param("number")

	if err := services.DeleteProcurement(transactionNumber, userID); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Procurement deleted successfully", nil)
}
