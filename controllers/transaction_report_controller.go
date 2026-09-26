package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// Dipakai bersama report procurement, mutation, disposal.
//
// Satu endpoint per report: JSON secara default, file excel kalau
// ?format=xlsx. Sengaja tidak dipisah ke /export supaya cukup satu baris menu
// (route_path) dan satu centang "read" di hak akses role.

func parseTransactionReportFilter(c *gin.Context) dto.TransactionReportFilter {
	return dto.TransactionReportFilter{
		StartDate:  c.Query("start_date"),
		EndDate:    c.Query("end_date"),
		BranchCode: c.Query("branch_code"),
		Stage:      c.Query("stage"),
		Search:     c.Query("search"),
	}
}

func isExcelReportRequest(c *gin.Context) bool {
	return c.Query("format") == "xlsx"
}

func reportErrorStatus(err error) int {
	if errors.Is(err, services.ErrReportBranchForbidden) {
		return http.StatusForbidden
	}
	return http.StatusInternalServerError
}

func writeReportExcel(c *gin.Context, file *excelize.File, filename string) {
	defer file.Close()
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Access-Control-Expose-Headers", "Content-Disposition")
	if err := file.Write(c.Writer); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to write excel file: "+err.Error())
	}
}

func respondTransactionReport[T any](c *gin.Context, viewer services.AssetViewer, rows []T, summary dto.TransactionReportSummary) {
	branches, all, err := services.GetReportBranchOptions(viewer)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Report retrieved successfully", dto.TransactionReportResponse[T]{
		Branches:    branches,
		AllBranches: all,
		Summary:     summary,
		Rows:        rows,
	})
}
