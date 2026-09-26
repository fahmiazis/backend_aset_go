package controllers

import (
	"backend-go/dto"
	"backend-go/models"
	"backend-go/services"
	"backend-go/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ============================================================
// DRAFT MANAGEMENT
// ============================================================

func CreateDisposalDraft(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.CreateDisposalDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.CreateDisposalDraft(userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Disposal draft created successfully", result)
}

func GetDisposalDetail(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	result, err := services.GetDisposalDetail(transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	result.WaitingForMe = services.IsDisposalWaitingForUser(
		c.GetString("user_id"), transactionNumber)

	utils.SuccessResponse(c, http.StatusOK, "Disposal detail retrieved successfully", result)
}

func GetAllDisposals(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	filter := dto.DisposalListFilter{
		Page:  page,
		Limit: limit,
		// dibaca dari token, bukan query — supaya tidak bisa mengintip
		// daftar tugas user lain
		ViewerUserID: c.GetString("user_id"),
		WaitingForMe: c.Query("waiting_for_me") == "true",
	}

	if v := c.Query("disposal_type"); v != "" {
		filter.DisposalType = &v
	}
	if v := c.Query("status"); v != "" {
		filter.Status = &v
	}
	if v := c.Query("current_stage"); v != "" {
		filter.CurrentStage = &v
	}
	if v := c.Query("created_by"); v != "" {
		filter.CreatedBy = &v
	}
	if v := c.Query("search"); v != "" {
		filter.Search = &v
	}
	if v := c.Query("start_date"); v != "" {
		filter.StartDate = &v
	}
	if v := c.Query("end_date"); v != "" {
		filter.EndDate = &v
	}

	results, total, err := services.GetAllDisposals(filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Disposals retrieved successfully", map[string]interface{}{
		"data":  results,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	})
}

func AddAssetToDisposal(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.AddDisposalAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.AddAssetToDisposal(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset added to disposal successfully", result)
}

func RemoveAssetFromDisposal(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.RemoveDisposalAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.RemoveAssetFromDisposal(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset removed from disposal successfully", result)
}

// ============================================================
// FLOW ACTIONS
// ============================================================

func SubmitDisposal(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.SubmitDisposalRequest
	_ = c.ShouldBindJSON(&req)

	result, err := services.SubmitDisposal(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Disposal submitted successfully", result)
}

// SetDisposalSaleValues — SELL only, dilakukan oleh purchasing
func SetDisposalSaleValues(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.SetDisposalSaleValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.SetDisposalSaleValues(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Sale values set successfully", result)
}

// SetDisposalIncomeValues — FINANCE, isi nilai pemasukan per aset
func SetDisposalIncomeValues(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.SetDisposalIncomeValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.SetDisposalIncomeValues(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Income values saved successfully", result)
}

// SetDisposalInvoices — TAX, isi nomor & tanggal faktur per aset
func SetDisposalInvoices(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.SetDisposalInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.SetDisposalInvoices(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Invoice data saved successfully", result)
}

func InitiateDisposalApprovalRequest(c *gin.Context) {
	userID := c.GetString("user_id")
	_ = userID
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.InitiateDisposalApprovalRequest
	_ = c.ShouldBindJSON(&req)

	if err := services.InitiateDisposalApprovalRequest(userID, transactionNumber, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval request initiated successfully", nil)
}

func GetDisposalApprovalRequestStatus(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	result, err := services.GetTransactionApprovalStatusByFlowCode(
		transactionNumber, services.TxDisposalFlow, models.FlowDisposalApprovalRequest)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval request status retrieved successfully", result)
}

func InitiateDisposalApprovalAgreement(c *gin.Context) {
	userID := c.GetString("user_id")
	_ = userID
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.InitiateDisposalApprovalRequest
	_ = c.ShouldBindJSON(&req)

	if err := services.InitiateDisposalApprovalAgreement(userID, transactionNumber, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval agreement initiated successfully", nil)
}

func GetDisposalApprovalAgreementStatus(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	result, err := services.GetTransactionApprovalStatusByFlowCode(
		transactionNumber, services.TxDisposalFlow, models.FlowDisposalApprovalAgreement)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval agreement status retrieved successfully", result)
}

func ExecuteDisposal(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.ExecuteDisposalRequest
	_ = c.ShouldBindJSON(&req)

	result, err := services.ExecuteDisposal(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Disposal executed successfully", result)
}

// ConfirmDisposalFinance — SELL only
func ConfirmDisposalFinance(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.ConfirmDisposalFinanceRequest
	_ = c.ShouldBindJSON(&req)

	result, err := services.ConfirmDisposalFinance(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Finance stage confirmed successfully", result)
}

// ConfirmDisposalTax — SELL only
func ConfirmDisposalTax(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.ConfirmDisposalTaxRequest
	_ = c.ShouldBindJSON(&req)

	result, err := services.ConfirmDisposalTax(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Tax stage confirmed successfully", result)
}

func ConfirmDisposalAssetDeletion(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.ConfirmDisposalAssetDeletionRequest
	_ = c.ShouldBindJSON(&req)

	result, err := services.ConfirmDisposalAssetDeletion(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset deletion confirmed successfully", result)
}

func RejectDisposal(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.RejectDisposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.RejectDisposal(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Disposal rejected", result)
}

// ReviseDisposal - POST /transactions/disposal/revise?transaction_number=xxx
// Approver step berjalan mengembalikan transaksi ke DRAFT
func ReviseDisposal(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.ReviseDisposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.ReviseDisposal(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Disposal returned to draft for revision", result)
}

// CancelDisposal - POST /transactions/disposal/cancel?transaction_number=xxx
// Pembatalan oleh pengaju transaksi (bukan reject oleh approver)
func CancelDisposal(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.CancelDisposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.CancelDisposal(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Disposal cancelled", result)
}

// GetDisposalAttachmentFile - GET /transactions/disposal/attachments/:id/file
// Stream file attachment (preview & download). Tetap lewat auth, jadi dokumen
// tidak bisa diambil hanya dengan menebak URL.
func GetDisposalAttachmentFile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid attachment id")
		return
	}

	att, absolutePath, err := services.GetDisposalAttachmentFile(uint(id))
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	contentType := "application/octet-stream"
	if att.MimeType != nil && *att.MimeType != "" {
		contentType = *att.MimeType
	}

	// inline supaya bisa dipreview; frontend tetap bisa memaksa download
	disposition := "inline"
	if c.Query("download") == "1" {
		disposition = "attachment"
	}

	c.Header("Content-Disposition", disposition+"; filename=\""+att.FileName+"\"")
	c.Header("Content-Type", contentType)
	c.File(absolutePath)
}

// ============================================================
// ATTACHMENT
// ============================================================

func UploadDisposalAttachment(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	disposalAssetIDStr := c.PostForm("transaction_disposal_asset_id")
	disposalAssetID, err := strconv.ParseUint(disposalAssetIDStr, 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid transaction_disposal_asset_id")
		return
	}

	configIDStr := c.PostForm("attachment_config_id")
	configID, err := strconv.ParseUint(configIDStr, 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid attachment_config_id")
		return
	}

	stage := c.PostForm("stage")
	if stage == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "stage is required")
		return
	}

	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	result, err := services.UploadDisposalAttachment(
		userID, transactionNumber, uint(disposalAssetID),
		uint(configID), stage, file, fileHeader,
	)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Attachment uploaded successfully", result)
}

func ReviewDisposalAttachment(c *gin.Context) {
	reviewerID := c.GetString("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req dto.ReviewDisposalAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.ReviewDisposalAttachment(reviewerID, uint(id), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Attachment reviewed successfully", result)
}

func GetDisposalAttachmentStatus(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	stage := c.Query("stage")
	if stage == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "stage is required")
		return
	}

	detail, err := services.GetDisposalDetail(transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	branchCode := services.GetCreatorBranchCode(detail.Transaction.CreatedBy)
	result, err := services.GetDisposalAttachmentStatus(transactionNumber, detail.Transaction.ID, stage, branchCode)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Attachment status retrieved successfully", result)
}
