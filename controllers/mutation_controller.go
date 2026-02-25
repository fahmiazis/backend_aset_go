package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateMutation(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.CreateMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	mutation, err := services.CreateMutation(userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Mutation created successfully", mutation)
}

func GetMutationByNumber(c *gin.Context) {
	transactionNumber := c.Param("number")

	mutation, err := services.GetMutationByTransactionNumber(transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Mutation retrieved successfully", mutation)
}

func GetAllMutations(c *gin.Context) {
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

	mutations, total, err := services.GetAllMutations(filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"data":  mutations,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	}

	utils.SuccessResponse(c, http.StatusOK, "Mutations retrieved successfully", response)
}

func UpdateMutation(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Param("number")

	var req dto.CreateMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	mutation, err := services.UpdateMutation(transactionNumber, userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Mutation updated successfully", mutation)
}

func DeleteMutation(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Param("number")

	if err := services.DeleteMutation(transactionNumber, userID); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Mutation deleted successfully", nil)
}
