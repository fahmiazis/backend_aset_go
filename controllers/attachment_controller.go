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
// ATTACHMENT CONFIG
// ============================================================

func GetAttachmentConfigs(c *gin.Context) {
	transactionType := c.Query("transaction_type")
	stage := c.Query("stage")
	branchCode := c.Query("branch_code")

	configs, err := services.GetAttachmentConfigs(transactionType, stage, branchCode)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Attachment configs retrieved successfully", configs)
}

func GetAttachmentConfigByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}

	cfg, err := services.GetAttachmentConfigByID(uint(id))
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Attachment config retrieved successfully", cfg)
}

func CreateAttachmentConfig(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.CreateAttachmentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	cfg, err := services.CreateAttachmentConfig(userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Attachment config created successfully", cfg)
}

func UpdateAttachmentConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req dto.UpdateAttachmentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	cfg, err := services.UpdateAttachmentConfig(uint(id), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Attachment config updated successfully", cfg)
}

func DeleteAttachmentConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := services.DeleteAttachmentConfig(uint(id)); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Attachment config deleted successfully", nil)
}

// ============================================================
// TRANSACTION ATTACHMENTS
// ============================================================

func UploadAttachment(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	transactionType := c.Query("transaction_type")
	stage := c.Query("stage")

	if transactionNumber == "" || transactionType == "" || stage == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number, transaction_type, and stage are required")
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

	result, err := services.UploadAttachment(
		userID, transactionNumber, transactionType, stage,
		uint(configID), file, fileHeader,
	)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Attachment uploaded successfully", result)
}

func ReviewAttachment(c *gin.Context) {
	reviewerID := c.GetString("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req dto.ReviewAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.ReviewAttachment(reviewerID, uint(id), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Attachment reviewed successfully", result)
}

func GetTransactionAttachments(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	transactionType := c.Query("transaction_type")
	stage := c.Query("stage") // optional

	if transactionNumber == "" || transactionType == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number and transaction_type are required")
		return
	}

	attachments, err := services.GetTransactionAttachments(transactionNumber, transactionType, stage)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Attachments retrieved successfully", attachments)
}

func GetAttachmentStatusSummary(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	transactionType := c.Query("transaction_type")
	stage := c.Query("stage")
	branchCode := c.Query("branch_code")

	if transactionNumber == "" || transactionType == "" || stage == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number, transaction_type, and stage are required")
		return
	}

	summary, err := services.GetAttachmentStatusSummary(transactionNumber, transactionType, stage, branchCode)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Attachment status retrieved successfully", summary)
}
