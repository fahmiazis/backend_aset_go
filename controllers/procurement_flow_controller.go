package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ============================================================
// SUBMIT
// POST /api/transactions/procurement/submit?transaction_number=xxx
// ============================================================

func SubmitProcurement(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.SubmitProcurementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.SubmitProcurement(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Procurement submitted successfully", result)
}

// ============================================================
// VERIFIKASI ASET
// POST /api/transactions/procurement/verify?transaction_number=xxx
// ============================================================

func VerifyProcurement(c *gin.Context) {
	userID := c.GetString("user_id")
	branchCode := c.GetString("branch_code")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.VerifyProcurementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.VerifyProcurement(userID, branchCode, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Procurement verified successfully", result)
}

// ============================================================
// INITIATE APPROVAL
// POST /api/transactions/procurement/initiate-approval?transaction_number=xxx
// flow_id tidak perlu diisi — auto-lookup by PROCUREMENT_APPROVAL
// ============================================================

func InitiateProcurementApproval(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	// Body optional — hanya notes & metadata
	var req dto.InitiateApprovalRequest
	_ = c.ShouldBindJSON(&req)

	if err := services.InitiateProcurementApproval(userID, transactionNumber, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval initiated successfully", nil)
}

// ============================================================
// COMPLETE APPROVAL
// POST /api/transactions/procurement/complete-approval?transaction_number=xxx
// ============================================================

func CompleteProcurementApproval(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	result, err := services.CompleteProcurementApproval(userID, transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval completed, moved to budget process", result)
}

// ============================================================
// PROSES BUDGET
// POST /api/transactions/procurement/process-budget?transaction_number=xxx
// ============================================================

func ProcessProcurementBudget(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.ProcessBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.ProcessProcurementBudget(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Budget processed successfully", result)
}

// ============================================================
// EKSEKUSI ASET
// POST /api/transactions/procurement/execute?transaction_number=xxx
// ============================================================

func ExecuteProcurementAsset(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.ExecuteAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.ExecuteProcurementAsset(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Assets executed successfully", result)
}

// ============================================================
// GOOD RECEIPT
// POST /api/transactions/procurement/gr?transaction_number=xxx
// ============================================================

func CreateAssetGR(c *gin.Context) {
	userID := c.GetString("user_id")
	userBranchCode := c.GetString("branch_code")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.CreateGRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.CreateAssetGR(userID, userBranchCode, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Good receipt recorded successfully", result)
}

// ============================================================
// GET GR STATUS
// GET /api/transactions/procurement/gr?transaction_number=xxx
// ============================================================

func GetProcurementGRStatus(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	result, err := services.GetProcurementGRStatusDetail(transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "GR status retrieved successfully", result)
}

// ============================================================
// REJECT
// POST /api/transactions/procurement/reject?transaction_number=xxx
// ============================================================

func RejectProcurement(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.RejectProcurementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.RejectProcurement(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Procurement rejected", result)
}

// ============================================================
// REVISI
// PUT /api/transactions/procurement/revise?transaction_number=xxx
// ============================================================

func ReviseProcurement(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.ReviseProcurementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.ReviseProcurement(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Procurement revised successfully", result)
}

// ============================================================
// GET DETAIL WITH STAGE
// GET /api/transactions/procurement/detail?transaction_number=xxx
// ============================================================

func GetProcurementDetailWithStage(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	result, err := services.GetProcurementDetailWithStage(transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Procurement detail retrieved successfully", result)
}

// ============================================================
// GET APPROVAL STATUS
// GET /api/transactions/procurement/approval-status?transaction_number=xxx
// ============================================================

func GetProcurementApprovalStatus(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	result, err := services.GetTransactionApprovalStatus(transactionNumber, "PROCUREMENT")
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval status retrieved successfully", result)
}
