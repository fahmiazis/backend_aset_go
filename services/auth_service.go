package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"backend-go/utils"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Register(req dto.RegisterRequest) (*models.User, error) {
	// Check if username exists
	var existingUser models.User
	if err := config.DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		return nil, errors.New("username already exists")
	}

	// Check if email exists
	if err := config.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return nil, errors.New("email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// Create user
	user := models.User{
		Username:  req.Username,
		Fullname:  req.Fullname,
		Email:     req.Email,
		Password:  string(hashedPassword),
		NIK:       stringToPointer(req.NIK),       // ← Helper function
		MPNNumber: stringToPointer(req.MPNNumber), // ← Helper function
		Status:    "active",
	}

	if err := config.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	// Assign default role (user)
	var defaultRole models.Role
	if err := config.DB.Where("name = ?", "user").First(&defaultRole).Error; err == nil {
		userRole := models.UserRole{
			UserID: user.ID,
			RoleID: defaultRole.ID,
		}
		config.DB.Create(&userRole)
	}

	return &user, nil
}

func Login(req dto.LoginRequest, deviceInfo, ipAddress string) (*dto.LoginResponse, error) {
	// Find user by username
	var user models.User
	if err := config.DB.Where("username = ? OR email = ?", req.Username, req.Username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	// Check if user is active
	if user.Status != "active" {
		return nil, errors.New("account is inactive")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Get user roles
	roles, err := GetUserRoles(user.ID)
	if err != nil {
		roles = []string{}
	}

	// Generate access token
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Username, user.Email, roles)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshTokenString, expiresAt, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	// Save refresh token to database
	refreshToken := models.RefreshToken{
		UserID:     user.ID,
		Token:      refreshTokenString,
		DeviceInfo: deviceInfo,
		IPAddress:  ipAddress,
		ExpiresAt:  time.Unix(expiresAt, 0),
		IsRevoked:  false,
	}

	if err := config.DB.Create(&refreshToken).Error; err != nil {
		return nil, err
	}

	// Build response
	response := &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 minutes in seconds
		User: dto.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Fullname:  user.Fullname,
			Email:     user.Email,
			NIK:       user.NIK,
			MPNNumber: user.MPNNumber,
			Status:    user.Status,
			Roles:     roles,
		},
	}

	return response, nil
}

func RefreshAccessToken(refreshTokenString string) (*dto.LoginResponse, error) {
	// Validate refresh token
	userID, err := utils.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Check if refresh token exists and not revoked
	var refreshToken models.RefreshToken
	if err := config.DB.Where("token = ? AND user_id = ? AND is_revoked = ?", refreshTokenString, userID, false).First(&refreshToken).Error; err != nil {
		return nil, errors.New("refresh token not found or revoked")
	}

	// Check if token expired
	if time.Now().After(refreshToken.ExpiresAt) {
		return nil, errors.New("refresh token expired")
	}

	// Get user
	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	// Check if user is active
	if user.Status != "active" {
		return nil, errors.New("account is inactive")
	}

	// Get user roles
	roles, err := GetUserRoles(user.ID)
	if err != nil {
		roles = []string{}
	}

	// Generate new access token
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Username, user.Email, roles)
	if err != nil {
		return nil, err
	}

	// Build response
	response := &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		User: dto.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Fullname:  user.Fullname,
			Email:     user.Email,
			NIK:       user.NIK,
			MPNNumber: user.MPNNumber,
			Status:    user.Status,
			Roles:     roles,
		},
	}

	return response, nil
}

func Logout(userID, refreshTokenString string) error {
	// Revoke refresh token
	return config.DB.Model(&models.RefreshToken{}).
		Where("user_id = ? AND token = ?", userID, refreshTokenString).
		Update("is_revoked", true).Error
}

func LogoutAllDevices(userID string) error {
	// Revoke all refresh tokens for user
	return config.DB.Model(&models.RefreshToken{}).
		Where("user_id = ?", userID).
		Update("is_revoked", true).Error
}

func GetUserRoles(userID string) ([]string, error) {
	var userRoles []models.UserRole
	if err := config.DB.Preload("Role").Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, err
	}

	roles := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roles[i] = ur.Role.Name
	}

	return roles, nil
}

func stringToPointer(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
