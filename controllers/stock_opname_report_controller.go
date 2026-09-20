package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func parseStockOpnameReportFilter(c *gin.Context) dto.StockOpnameReportFilter {
	month, _ := strconv.Atoi(c.Query("month"))
	year, _ := strconv.Atoi(c.Query("year"))
	return dto.StockOpnameReportFilter{
		Month:      month,
		Year:       year,
		BranchCode: c.Query("branch_code"),
	}
}

// GetStockOpnameReportDashboard → GET /transactions/stock-opname/report/dashboard
func GetStockOpnameReportDashboard(c *gin.Context) {
	filter := parseStockOpnameReportFilter(c)

	result, err := services.GetStockOpnameReportDashboard(filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opname dashboard retrieved successfully", result)
}

// GetStockOpnameReportDetail → GET /transactions/stock-opname/report/detail
func GetStockOpnameReportDetail(c *gin.Context) {
	filter := parseStockOpnameReportFilter(c)

	result, err := services.GetStockOpnameReportDetail(filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Stock opname detail report retrieved successfully", result)
}

// ExportStockOpnameReport → GET /transactions/stock-opname/report/export
func ExportStockOpnameReport(c *gin.Context) {
	month, _ := strconv.Atoi(c.Query("month"))
	year, _ := strconv.Atoi(c.Query("year"))
	req := dto.StockOpnameExportRequest{
		Month:      month,
		Year:       year,
		BranchCode: c.Query("branch_code"),
	}

	file, filename, err := services.ExportStockOpnameReportExcel(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err := file.Write(c.Writer); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to write excel file: "+err.Error())
		return
	}
}
