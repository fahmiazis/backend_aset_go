package controllers

import (
	"backend-go/services"
	"backend-go/utils"

	"github.com/gin-gonic/gin"
)

// GetProcurementReport → GET /reports/procurement[?format=xlsx]
func GetProcurementReport(c *gin.Context) {
	filter := parseTransactionReportFilter(c)
	viewer := assetViewer(c)

	if isExcelReportRequest(c) {
		file, filename, err := services.ExportProcurementReport(filter, viewer)
		if err != nil {
			utils.ErrorResponse(c, reportErrorStatus(err), err.Error())
			return
		}
		writeReportExcel(c, file, filename)
		return
	}

	rows, summary, err := services.GetProcurementReport(filter, viewer)
	if err != nil {
		utils.ErrorResponse(c, reportErrorStatus(err), err.Error())
		return
	}
	respondTransactionReport(c, viewer, rows, summary)
}
