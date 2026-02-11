package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// HOMEBASE MANAGEMENT
// ============================================================================

// GetUserHomebases - GET /user/homebases
// Get all homebase branches for current user
func GetUserHomebases(c *gin.Context) {
	userID := c.GetString("user_id")

	homebases, err := services.GetUserHomebaseBranches(userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Map to response
	response := make([]dto.HomebaseBranchResponse, len(homebases))
	for i, hb := range homebases {
		response[i] = dto.HomebaseBranchResponse{
			ID:         hb.ID,
			UserID:     hb.UserID,
			BranchID:   hb.BranchID,
			BranchType: hb.BranchType,
			IsActive:   hb.IsActive,
			CreatedAt:  hb.CreatedAt,
		}

		if hb.Branch != nil {
			response[i].Branch = &dto.HomebaseBranchDetail{
				ID:         hb.Branch.ID,
				BranchCode: hb.Branch.BranchCode,
				BranchName: hb.Branch.BranchName,
				BranchType: hb.Branch.BranchType,
				Status:     hb.Branch.Status,
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "Homebases retrieved successfully", response)
}

// SetActiveHomebase - POST /user/homebase/set-active
// Set which homebase is active for the user (for login selection)
func SetActiveHomebase(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.SetActiveHomebaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.SetActiveHomebase(userID, req.BranchID); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Active homebase set successfully", nil)
}

// ============================================================================
// TRANSACTION NUMBER MANAGEMENT
// ============================================================================

// GenerateTransactionNumber - POST /transaction-number/generate
// Generate new transaction number for current user's active homebase
func GenerateTransactionNumber(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.GenerateTransactionNumberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	transactionNumber, err := services.GenerateTransactionNumber(userID, req.TransactionType)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get branch info for response
	homebases, _ := services.GetUserHomebaseBranches(userID)
	var branchCode, branchName string
	for _, hb := range homebases {
		if hb.IsActive && hb.Branch != nil {
			branchCode = hb.Branch.BranchCode
			branchName = hb.Branch.BranchName
			break
		}
	}

	response := dto.GenerateTransactionNumberResponse{
		TransactionNumber: transactionNumber,
		BranchCode:        branchCode,
		BranchName:        branchName,
		TransactionType:   req.TransactionType,
		Status:            "delayed",
	}

	utils.SuccessResponse(c, http.StatusCreated, "Transaction number generated successfully", response)
}

// MarkTransactionUsed - POST /transaction-number/mark-used
// Mark transaction as used (when form is submitted)
func MarkTransactionUsed(c *gin.Context) {
	var req struct {
		TransactionNumber string `json:"transaction_number" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.MarkTransactionAsUsed(req.TransactionNumber); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Transaction marked as used", nil)
}

// MarkTransactionExpired - POST /transaction-number/mark-expired
// Mark transaction as expired (when cancelled/replaced)
func MarkTransactionExpired(c *gin.Context) {
	var req struct {
		TransactionNumber string `json:"transaction_number" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.MarkTransactionAsExpired(req.TransactionNumber); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Transaction marked as expired", nil)
}

// GetTransactionStatus - GET /transaction-number/status/:number
// Get status of a transaction number
func GetTransactionStatus(c *gin.Context) {
	transactionNumber := c.Param("number")

	status, err := services.GetTransactionStatus(transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	response := map[string]string{
		"transaction_number": transactionNumber,
		"status":             status,
	}

	utils.SuccessResponse(c, http.StatusOK, "Transaction status retrieved successfully", response)
}
