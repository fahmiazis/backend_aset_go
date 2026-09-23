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

func CreateStockOpnameDraft(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.CreateStockOpnameDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.CreateStockOpnameDraft(userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Stock opname draft created successfully", result)
}

func GetStockOpnameFlowDetail(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	result, err := services.GetStockOpnameFlowDetail(transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opname detail retrieved successfully", result)
}

func GetAllStockOpnamesFlow(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	filter := services.StockOpnameListFilter{
		Page:  page,
		Limit: limit,
	}

	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if stage := c.Query("current_stage"); stage != "" {
		filter.CurrentStage = &stage
	}
	if createdBy := c.Query("created_by"); createdBy != "" {
		filter.CreatedBy = &createdBy
	}
	if startDate := c.Query("start_date"); startDate != "" {
		filter.StartDate = &startDate
	}
	if endDate := c.Query("end_date"); endDate != "" {
		filter.EndDate = &endDate
	}

	results, total, err := services.GetAllStockOpnameDrafts(filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"data":  results,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opnames retrieved successfully", response)
}

func UpdateStockOpnameFinding(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.UpdateStockOpnameFindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.UpdateStockOpnameFinding(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opname finding updated successfully", result)
}

func BulkUpdateStockOpnameFinding(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.BulkUpdateStockOpnameFindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.BulkUpdateStockOpnameFinding(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opname findings saved successfully", result)
}

// ============================================================
// EXCEL TEMPLATE (download + bulk update via upload)
// ============================================================

func DownloadStockOpnameTemplate(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	file, filename, err := services.GenerateStockOpnameTemplateExcel(userID, transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err := file.Write(c.Writer); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to write excel file: "+err.Error())
		return
	}
}

// DownloadStockOpnameDocumentation menghasilkan file excel dokumentasi foto
// bukti fisik (1 aset = 1 baris + foto) — cuma bisa diunduh setelah stock
// opname disubmit (bukan DRAFT lagi), karena baru saat itu foto per aset
// dijamin lengkap & tervalidasi.
func DownloadStockOpnameDocumentation(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	file, filename, err := services.GenerateStockOpnameDocumentationExcel(userID, transactionNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err := file.Write(c.Writer); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to write excel file: "+err.Error())
		return
	}
}

func UploadStockOpnameTemplate(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	result, err := services.ProcessStockOpnameTemplateUpload(userID, transactionNumber, file)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opname template processed successfully", result)
}

// ============================================================
// FLOW ACTIONS
// ============================================================

func SubmitStockOpname(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.SubmitStockOpnameRequest
	_ = c.ShouldBindJSON(&req)

	result, err := services.SubmitStockOpname(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opname submitted successfully", result)
}

func InitiateStockOpnameApproval(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.InitiateApprovalRequest
	_ = c.ShouldBindJSON(&req)

	if err := services.InitiateStockOpnameApproval(userID, transactionNumber, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval initiated successfully", nil)
}

func GetStockOpnameApprovalStatus(c *gin.Context) {
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	result, err := services.GetTransactionApprovalStatus(transactionNumber, services.TxStockOpnameFlow)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval status retrieved successfully", result)
}

func ExecuteStockOpname(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.ExecuteStockOpnameRequest
	_ = c.ShouldBindJSON(&req)

	result, err := services.ExecuteStockOpname(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opname executed successfully", result)
}

func RejectStockOpname(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	var req dto.RejectStockOpnameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.RejectStockOpname(userID, transactionNumber, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opname rejected", result)
}

// ============================================================
// FOTO BUKTI FISIK PER ASSET
// ============================================================

func UploadStockOpnameAssetPhoto(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	assetID, err := strconv.ParseUint(c.PostForm("asset_id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "asset_id is required and must be numeric")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	// file_modified_at = File.lastModified (epoch ms) dari browser — opsional,
	// dipakai buat validasi "gak lebih dari 10 hari" tanpa bergantung ke EXIF.
	clientModifiedAtMs, _ := strconv.ParseInt(c.PostForm("file_modified_at"), 10, 64)

	result, err := services.UploadStockOpnameAssetPhoto(userID, transactionNumber, uint(assetID), file, header, clientModifiedAtMs)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Photo uploaded successfully", result)
}

func ServeStockOpnameAssetPhoto(c *gin.Context) {
	photoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid photo id")
		return
	}

	path, _, err := services.GetStockOpnameAssetPhotoFilePath(uint(photoID))
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	// Inline (bukan attachment) biar bisa langsung dipakai sebagai <img src>
	// buat thumbnail di modal/grid, bukan trigger download.
	c.File(path)
}

func UploadStockOpnameBorrowDocument(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionNumber := c.Query("transaction_number")
	if transactionNumber == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "transaction_number is required")
		return
	}

	assetID, err := strconv.ParseUint(c.PostForm("asset_id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "asset_id is required and must be numeric")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	result, err := services.UploadStockOpnameBorrowDocument(userID, transactionNumber, uint(assetID), file, header)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Borrow document uploaded successfully", result)
}

func GetStockOpnameConfig(c *gin.Context) {
	result, err := services.GetStockOpnameConfig()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Stock opname config retrieved successfully", result)
}

func UpdateStockOpnameConfig(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.UpdateStockOpnameConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.UpdateStockOpnameConfig(userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opname config updated successfully", result)
}

func ServeStockOpnameBorrowDocument(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid document id")
		return
	}

	path, fileName, err := services.GetStockOpnameBorrowDocumentFilePath(uint(docID))
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	// Inline biar bisa langsung dibuka di tab baru, bukan trigger download.
	c.Header("Content-Disposition", "inline; filename=\""+fileName+"\"")
	c.File(path)
}
