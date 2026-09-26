package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ============================================================
// EMAIL TEMPLATE (master)
// ============================================================

func GetEmailTemplates(c *gin.Context) {
	templates, err := services.GetEmailTemplates(c.Query("transaction_type"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Email templates retrieved successfully", templates)
}

func GetEmailTemplateByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}

	tpl, err := services.GetEmailTemplateByID(uint(id))
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Email template retrieved successfully", tpl)
}

func CreateEmailTemplate(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.CreateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	tpl, err := services.CreateEmailTemplate(userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "Email template created successfully", tpl)
}

func UpdateEmailTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req dto.UpdateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	tpl, err := services.UpdateEmailTemplate(uint(id), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Email template updated successfully", tpl)
}

func DeleteEmailTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := services.DeleteEmailTemplate(uint(id)); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Email template deleted successfully", nil)
}

// ============================================================
// EMAIL TRANSAKSI (dialog sebelum aksi stage)
// ============================================================

func isAdminRequest(c *gin.Context) bool {
	roles, ok := c.Get("roles")
	if !ok {
		return false
	}
	list, ok := roles.([]string)
	if !ok {
		return false
	}
	for _, r := range list {
		if r == "admin" {
			return true
		}
	}
	return false
}

// GET /transaction-emails/preview?transaction_type=&transaction_number=&action=
// agreement baru: ?transaction_type=disposal_agreement&member_numbers=a,b
func PreviewTransactionEmail(c *gin.Context) {
	transactionType := c.Query("transaction_type")
	transactionNumber := c.Query("transaction_number")
	action := c.DefaultQuery("action", "proceed")

	// Saat membuat agreement nomornya belum ada — yang dikirim daftar
	// transaksi anggotanya (dipisah koma; nomor transaksi tidak mengandung koma)
	var memberNumbers []string
	for _, n := range strings.Split(c.Query("member_numbers"), ",") {
		if n = strings.TrimSpace(n); n != "" {
			memberNumbers = append(memberNumbers, n)
		}
	}

	if transactionType == "" || (transactionNumber == "" && len(memberNumbers) == 0) {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_type and transaction_number (or member_numbers) are required")
		return
	}

	preview, err := services.PreviewTransactionEmail(c.GetString("user_id"), transactionType, transactionNumber, action, memberNumbers)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Email preview retrieved successfully", preview)
}

// POST /transaction-emails/send — dipanggil setelah aksi stage berhasil.
// Gagal kirim tetap dibalas 200 dengan status FAILED: aksinya sudah terjadi,
// dan log-nya sudah tersimpan untuk dikirim ulang.
func SendTransactionEmail(c *gin.Context) {
	var req dto.SendTransactionEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.SendTransactionEmail(c.GetString("user_id"), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Email processed", result)
}

// GET /transaction-emails/logs?status=FAILED&transaction_number=
func GetTransactionEmailLogs(c *gin.Context) {
	filter := services.EmailLogFilter{
		Status:            c.Query("status"),
		TransactionNumber: c.Query("transaction_number"),
	}
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil {
		filter.Limit = limit
	}
	if !isAdminRequest(c) {
		filter.SentBy = c.GetString("user_id")
	}

	logs, err := services.GetTransactionEmailLogs(filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Email logs retrieved successfully", logs)
}

// POST /transaction-emails/logs/:id/resend
func ResendTransactionEmail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}

	result, err := services.ResendTransactionEmail(c.GetString("user_id"), isAdminRequest(c), uint(id))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Email processed", result)
}
