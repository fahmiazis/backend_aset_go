package controllers

import (
	"backend-go/dto"
	"backend-go/models"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ============================================================
// DISPOSAL AGREEMENT
// ============================================================

// GetEligibleDisposalsForAgreement - GET /transactions/disposal-agreements/eligible
// Transaksi disposal yang sudah lolos approval request dan belum masuk
// agreement aktif.
func GetEligibleDisposalsForAgreement(c *gin.Context) {
	result, err := services.GetEligibleDisposalsForAgreement()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Eligible disposals retrieved successfully", result)
}

// CreateDisposalAgreement - POST /transactions/disposal-agreements
func CreateDisposalAgreement(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.CreateDisposalAgreementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.CreateDisposalAgreement(userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Disposal agreement created successfully", result)
}

// GetAllDisposalAgreements - GET /transactions/disposal-agreements
func GetAllDisposalAgreements(c *gin.Context) {
	var filter dto.DisposalAgreementListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, total, err := services.GetAllDisposalAgreements(filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Disposal agreements retrieved successfully", gin.H{
		"data":  result,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	})
}

// GetDisposalAgreementDetail - GET /transactions/disposal-agreements/detail?agreement_number=xxx
func GetDisposalAgreementDetail(c *gin.Context) {
	agreementNumber := c.Query("agreement_number")
	if agreementNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "agreement_number is required")
		return
	}

	result, err := services.GetDisposalAgreementDetail(agreementNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Disposal agreement detail retrieved successfully", result)
}

// GetDisposalAgreementApprovalStatus - GET /transactions/disposal-agreements/approval-status?agreement_number=xxx
func GetDisposalAgreementApprovalStatus(c *gin.Context) {
	agreementNumber := c.Query("agreement_number")
	if agreementNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "agreement_number is required")
		return
	}

	result, err := services.GetTransactionApprovalStatusByFlowCode(
		agreementNumber, services.TxDisposalAgreement, models.FlowDisposalApprovalAgreement)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval status retrieved successfully", result)
}
