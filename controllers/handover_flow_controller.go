package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ============================================================
// SERAH TERIMA ASET — /transactions/handover
// ============================================================

func requireTransactionNumber(c *gin.Context) (string, bool) {
	number := c.Query("transaction_number")
	if number == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return "", false
	}
	return number, true
}

func CreateHandoverDraft(c *gin.Context) {
	var req dto.CreateHandoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}
	result, err := services.CreateHandoverDraft(c.GetString("user_id"), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "Handover draft created successfully", result)
}

func GetAllHandovers(c *gin.Context) {
	var filter dto.HandoverListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}
	filter.ViewerUserID = c.GetString("user_id")

	items, total, err := services.GetAllHandovers(filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	page, limit := filter.Page, filter.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	utils.SuccessResponse(c, http.StatusOK, "Handovers retrieved successfully", gin.H{
		"data": items, "total": total, "page": page, "limit": limit,
	})
}

func GetHandoverDetail(c *gin.Context) {
	number, ok := requireTransactionNumber(c)
	if !ok {
		return
	}
	result, err := services.GetHandoverDetail(number, c.GetString("user_id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Handover detail retrieved successfully", result)
}

// GET /transactions/handover/eligible-assets?handover_type=&search=&exclude_transaction_number=
func GetHandoverEligibleAssets(c *gin.Context) {
	result, err := services.GetHandoverEligibleAssets(c.GetString("user_id"),
		c.Query("handover_type"), c.Query("search"), c.Query("exclude_transaction_number"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Eligible assets retrieved successfully", result)
}

func GetHandoverRecipients(c *gin.Context) {
	result, err := services.GetHandoverRecipients(c.GetString("user_id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Recipients retrieved successfully", result)
}

func UpdateHandoverDraft(c *gin.Context) {
	number, ok := requireTransactionNumber(c)
	if !ok {
		return
	}
	var req dto.UpdateHandoverDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}
	result, err := services.UpdateHandoverDraft(c.GetString("user_id"), number, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Handover draft updated successfully", result)
}

func bindOptionalNotes(c *gin.Context) dto.HandoverActionRequest {
	var req dto.HandoverActionRequest
	_ = c.ShouldBindJSON(&req) // body opsional
	return req
}

func SubmitHandover(c *gin.Context) {
	number, ok := requireTransactionNumber(c)
	if !ok {
		return
	}
	result, err := services.SubmitHandover(c.GetString("user_id"), number, bindOptionalNotes(c))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Handover submitted successfully", result)
}

func GetHandoverApprovalStatus(c *gin.Context) {
	number, ok := requireTransactionNumber(c)
	if !ok {
		return
	}
	result, err := services.GetTransactionApprovalStatus(number, services.TxHandover)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Approval status retrieved successfully", result)
}

func ConfirmHandoverReceiving(c *gin.Context) {
	number, ok := requireTransactionNumber(c)
	if !ok {
		return
	}
	result, err := services.ConfirmHandoverReceiving(c.GetString("user_id"), number, bindOptionalNotes(c))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Handover received successfully", result)
}

func RejectHandoverReceiving(c *gin.Context) {
	number, ok := requireTransactionNumber(c)
	if !ok {
		return
	}
	var req dto.RejectHandoverReceivingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}
	result, err := services.RejectHandoverReceiving(c.GetString("user_id"), number, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Handover rejected", result)
}

func ReturnHandoverForRevision(c *gin.Context) {
	number, ok := requireTransactionNumber(c)
	if !ok {
		return
	}
	var req dto.ReturnForRevisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}
	result, err := services.ReturnHandoverForRevision(c.GetString("user_id"), number, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Handover returned for revision", result)
}

func CancelHandover(c *gin.Context) {
	number, ok := requireTransactionNumber(c)
	if !ok {
		return
	}
	var req dto.CancelTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}
	result, err := services.CancelHandover(c.GetString("user_id"), number, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Handover cancelled", result)
}
