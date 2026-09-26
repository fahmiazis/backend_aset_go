package controllers

import (
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /notifications/waiting — isi lonceng navbar
func GetWaitingNotifications(c *gin.Context) {
	result, err := services.GetWaitingNotifications(c.GetString("user_id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Waiting notifications retrieved successfully", result)
}
