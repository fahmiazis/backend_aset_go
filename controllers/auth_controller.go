package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Register - POST /auth/register
func Register(c *gin.Context) {
	var req dto.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	user, err := services.Register(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "User registered successfully", gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

// Login - POST /auth/login
func Login(c *gin.Context) {
	var req dto.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	// Get device info and IP
	deviceInfo := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	response, err := services.Login(req, deviceInfo, ipAddress)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Login successful", response)
}

// RefreshToken - POST /auth/refresh
func RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	response, err := services.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Token refreshed successfully", response)
}

// Logout - POST /auth/logout
func Logout(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.Logout(userID, req.RefreshToken); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to logout")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Logout successful", nil)
}

// LogoutAllDevices - POST /auth/logout-all
func LogoutAllDevices(c *gin.Context) {
	userID := c.GetString("user_id")

	if err := services.LogoutAllDevices(userID); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to logout from all devices")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Logged out from all devices successfully", nil)
}

// GetProfile - GET /auth/me
func GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	user, err := services.GetUserByID(userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Profile retrieved successfully", user)
}
