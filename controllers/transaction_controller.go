package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetTransactionByNumber - GET /transactions/:number
func GetTransactionByNumber(c *gin.Context) {
	transactionNumber := c.Query("number")

	transaction, err := services.GetTransactionByNumber(transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Transaction retrieved successfully", transaction)
}

// GetAllTransactions - GET /transactions
func GetAllTransactions(c *gin.Context) {
	var filter dto.TransactionListFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	// Set defaults
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 10
	}

	transactions, total, err := services.GetAllTransactions(filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"data":  transactions,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	}

	utils.SuccessResponse(c, http.StatusOK, "Transactions retrieved successfully", response)
}

// GetMyTransactions - GET /transactions/my
func GetMyTransactions(c *gin.Context) {
	userID := c.GetString("user_id")

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

	transactions, total, err := services.GetTransactionsByUser(userID, filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"data":  transactions,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	}

	utils.SuccessResponse(c, http.StatusOK, "My transactions retrieved successfully", response)
}
