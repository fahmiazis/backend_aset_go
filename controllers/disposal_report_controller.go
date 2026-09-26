package controllers

import (
	"backend-go/services"
	"backend-go/utils"

	"github.com/gin-gonic/gin"
)

// GetDisposalReport → GET /reports/disposal[?format=xlsx]
func GetDisposalReport(c *gin.Context) {
	filter := parseTransactionReportFilter(c)
	viewer := assetViewer(c)

	if isExcelReportRequest(c) {
		file, filename, err := services.ExportDisposalReport(filter, viewer)
		if err != nil {
			utils.ErrorResponse(c, reportErrorStatus(err), err.Error())
			return
		}
		writeReportExcel(c, file, filename)
		return
	}

	rows, summary, err := services.GetDisposalReport(filter, viewer)
	if err != nil {
		utils.ErrorResponse(c, reportErrorStatus(err), err.Error())
		return
	}
	respondTransactionReport(c, viewer, rows, summary)
}
