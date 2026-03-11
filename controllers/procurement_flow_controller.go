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
// POST /api/transactions/procurements/:transaction_number/submit
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
// POST /api/transactions/procurements/:transaction_number/verify
// ============================================================

func VerifyProcurement(c *gin.Context) {
	userID := c.GetString("user_id")
	branchCode := c.GetString("branch_code") // dari JWT middleware, branch aktif PIC Asset
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
// POST /api/transactions/procurements/:transaction_number/initiate-approval
// ============================================================

func InitiateProcurementApproval(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.InitiateApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.InitiateProcurementApproval(userID, transactionNumber, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval initiated successfully", nil)
}

// ============================================================
// COMPLETE APPROVAL (dipanggil setelah semua step approved)
// POST /api/transactions/procurements/:transaction_number/complete-approval
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
// POST /api/transactions/procurements/:transaction_number/process-budget
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
// POST /api/transactions/procurements/:transaction_number/execute
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
// POST /api/transactions/procurements/:transaction_number/gr
// ============================================================

func CreateAssetGR(c *gin.Context) {
	userID := c.GetString("user_id")
	userBranchCode := c.GetString("branch_code") // dari JWT middleware
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
// GET GR STATUS per transaksi
// GET /api/transactions/procurements/:transaction_number/gr
// ============================================================

func GetProcurementGRStatus(c *gin.Context) {
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

	utils.SuccessResponse(c, http.StatusOK, "GR status retrieved successfully", result.GRStatus)
}

// ============================================================
// REJECT
// POST /api/transactions/procurements/:transaction_number/reject
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
// PUT /api/transactions/procurements/:transaction_number/revise
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
// GET /api/transactions/procurements/:transaction_number/detail
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
// GET /api/transactions/procurements/:transaction_number/approval-status
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
