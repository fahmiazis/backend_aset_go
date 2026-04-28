package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateDisposal(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.CreateDisposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	disposal, err := services.CreateDisposal(userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Disposal created successfully", disposal)
}

func GetDisposalByNumber(c *gin.Context) {
	transactionNumber := c.Param("number")

	disposal, err := services.GetDisposalByTransactionNumber(transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Disposal retrieved successfully", disposal)
}

// func GetAllDisposals(c *gin.Context) {
// 	var filter dto.TransactionListFilter

// 	if err := c.ShouldBindQuery(&filter); err != nil {
// 		utils.ValidationErrorResponse(c, err)
// 		return
// 	}

// 	if filter.Page == 0 {
// 		filter.Page = 1
// 	}
// 	if filter.Limit == 0 {
// 		filter.Limit = 10
// 	}

// 	disposals, total, err := services.GetAllDisposals(filter)
// 	if err != nil {
// 		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
// 		return
// 	}

// 	response := map[string]interface{}{
// 		"data":  disposals,
// 		"total": total,
// 		"page":  filter.Page,
// 		"limit": filter.Limit,
// 	}

// 	utils.SuccessResponse(c, http.StatusOK, "Disposals retrieved successfully", response)
// }

func UpdateDisposal(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Param("number")

	var req dto.CreateDisposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	disposal, err := services.UpdateDisposal(transactionNumber, userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Disposal updated successfully", disposal)
}

func DeleteDisposal(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Param("number")

	if err := services.DeleteDisposal(transactionNumber, userID); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Disposal deleted successfully", nil)
}
