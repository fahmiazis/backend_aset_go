package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ChangeMyPassword - PUT /auth/me/password
func ChangeMyPassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.ChangeOwnPassword(c.GetString("user_id"), req.CurrentPassword, req.NewPassword); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Password changed successfully", nil)
}

// UploadMyAvatar - POST /auth/me/avatar (multipart, field "file")
func UploadMyAvatar(c *gin.Context) {
	uploadAvatar(c, c.GetString("user_id"))
}

// DeleteMyAvatar - DELETE /auth/me/avatar
func DeleteMyAvatar(c *gin.Context) {
	deleteAvatar(c, c.GetString("user_id"))
}

// UploadUserAvatar - POST /users/:id/avatar (admin)
func UploadUserAvatar(c *gin.Context) {
	uploadAvatar(c, c.Param("id"))
}

// DeleteUserAvatar - DELETE /users/:id/avatar (admin)
func DeleteUserAvatar(c *gin.Context) {
	deleteAvatar(c, c.Param("id"))
}

// ServeUserAvatar - GET /users/:id/avatar — file foto profil
func ServeUserAvatar(c *gin.Context) {
	path, err := services.UserAvatarPath(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}
	c.Header("Cache-Control", "private, max-age=300")
	c.File(path)
}

func uploadAvatar(c *gin.Context, userID string) {
	file, err := c.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "file is required")
		return
	}

	if err := services.SaveUserAvatar(userID, file); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Avatar updated successfully", nil)
}

func deleteAvatar(c *gin.Context, userID string) {
	if err := services.RemoveUserAvatar(userID); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Avatar removed successfully", nil)
}
