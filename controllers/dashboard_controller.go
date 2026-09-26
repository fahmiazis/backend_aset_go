package controllers

import (
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetDashboardSummary → GET /dashboard/summary
func GetDashboardSummary(c *gin.Context) {
	result, err := services.GetDashboardSummary(assetViewer(c))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Dashboard summary retrieved successfully", result)
}
