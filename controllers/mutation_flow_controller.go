package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ============================================================
// DRAFT MANAGEMENT
// ============================================================

func CreateMutationDraft(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.CreateMutationDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.CreateMutationDraft(userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Mutation draft created successfully", result)
}

func GetMutationDetail(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	result, err := services.GetMutationDetail(transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Mutation detail retrieved successfully", result)
}

func AddAssetToMutation(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.AddMutationAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.AddAssetToMutation(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset added to mutation successfully", result)
}

func RemoveAssetFromMutation(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.RemoveMutationAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.RemoveAssetFromMutation(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset removed from mutation successfully", result)
}

// ============================================================
// FLOW ACTIONS
// ============================================================

func SubmitMutation(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.SubmitMutationRequest
	_ = c.ShouldBindJSON(&req)

	result, err := services.SubmitMutation(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Mutation submitted successfully", result)
}

func InitiateMutationApproval(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.InitiateApprovalRequest
	_ = c.ShouldBindJSON(&req)

	if err := services.InitiateMutationApproval(userID, transactionNumber, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval initiated successfully", nil)
}

func GetMutationApprovalStatus(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	result, err := services.GetTransactionApprovalStatus(transactionNumber, "mutation")
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval status retrieved successfully", result)
}

func ExecuteMutation(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.ExecuteMutationRequest
	_ = c.ShouldBindJSON(&req)

	result, err := services.ExecuteMutation(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Mutation executed successfully", result)
}

func RejectMutation(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.RejectMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.RejectMutation(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Mutation rejected", result)
}

// ============================================================
// ATTACHMENT PER ASSET
// ============================================================

func UploadMutationAttachment(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	mutationAssetIDStr := c.PostForm("transaction_mutation_asset_id")
	mutationAssetID, err := strconv.ParseUint(mutationAssetIDStr, 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid transaction_mutation_asset_id")
		return
	}

	configIDStr := c.PostForm("attachment_config_id")
	configID, err := strconv.ParseUint(configIDStr, 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid attachment_config_id")
		return
	}

	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	result, err := services.UploadMutationAttachment(
		userID, transactionNumber, uint(mutationAssetID),
		uint(configID), file, fileHeader,
	)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Attachment uploaded successfully", result)
}

func ReviewMutationAttachment(c *gin.Context) {
	reviewerID := c.GetString("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req dto.ReviewMutationAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.ReviewMutationAttachment(reviewerID, uint(id), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Attachment reviewed successfully", result)
}

func GetMutationAttachmentStatus(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	detail, err := services.GetMutationDetail(transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	result, err := services.GetMutationAttachmentStatus(transactionNumber, detail.Transaction.ID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Attachment status retrieved successfully", result)
}
