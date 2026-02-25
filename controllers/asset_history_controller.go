package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetAssetHistory(c *gin.Context) {
	assetNumber := c.Param("number")

	history, err := services.GetAssetHistory(assetNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset history retrieved successfully", history)
}

func GetAllAssetHistories(c *gin.Context) {
	var filter dto.AssetHistoryFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 10
	}

	histories, total, err := services.GetAllAssetHistories(filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"data":  histories,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset histories retrieved successfully", response)
}
