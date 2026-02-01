package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func GetAllUsers() ([]dto.UserDetailResponse, error) {
	var users []models.User
	if err := config.DB.Preload("UserRoles.Role").Find(&users).Error; err != nil {
		return nil, err
	}

	response := make([]dto.UserDetailResponse, len(users))
	for i, user := range users {
		roles := make([]dto.RoleResponse, len(user.UserRoles))
		for j, ur := range user.UserRoles {
			roles[j] = dto.RoleResponse{
				ID:          ur.Role.ID,
				Name:        ur.Role.Name,
				Description: ur.Role.Description,
			}
		}

		response[i] = dto.UserDetailResponse{
			ID:        user.ID,
			Username:  user.Username,
			Fullname:  user.Fullname,
			Email:     user.Email,
			NIK:       user.NIK,
			MPNNumber: user.MPNNumber,
			Status:    user.Status,
			Roles:     roles,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		}
	}

	return response, nil
}

func GetUserByID(id string) (*dto.UserDetailResponse, error) {
	var user models.User
	if err := config.DB.Preload("UserRoles.Role").First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	roles := make([]dto.RoleResponse, len(user.UserRoles))
	for i, ur := range user.UserRoles {
		roles[i] = dto.RoleResponse{
			ID:          ur.Role.ID,
			Name:        ur.Role.Name,
			Description: ur.Role.Description,
		}
	}

	response := &dto.UserDetailResponse{
		ID:        user.ID,
		Username:  user.Username,
		Fullname:  user.Fullname,
		Email:     user.Email,
		NIK:       user.NIK,
		MPNNumber: user.MPNNumber,
		Status:    user.Status,
		Roles:     roles,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	return response, nil
}

func UpdateUser(id string, req dto.UpdateUserRequest) (*dto.UserDetailResponse, error) {
	var user models.User
	if err := config.DB.First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Fullname != "" {
		updates["fullname"] = req.Fullname
	}
	if req.Email != "" {
		// Check if email already used by another user
		var existingUser models.User
		if err := config.DB.Where("email = ? AND id != ?", req.Email, id).First(&existingUser).Error; err == nil {
			return nil, errors.New("email already in use")
		}
		updates["email"] = req.Email
	}
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, errors.New("failed to hash password")
		}
		updates["password"] = string(hashedPassword)
	}
	if req.NIK != "" {
		updates["nik"] = req.NIK
	}
	if req.MPNNumber != "" {
		updates["mpn_number"] = req.MPNNumber
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}

	return GetUserByID(id)
}

func DeleteUser(id string) error {
	var user models.User
	if err := config.DB.First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("user not found")
		}
		return err
	}

	return config.DB.Delete(&user).Error
}

func AssignRoles(userID string, req dto.AssignRoleRequest) error {
	// Check if user exists
	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("user not found")
		}
		return err
	}

	// Delete existing roles
	if err := config.DB.Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
		return err
	}

	// Assign new roles
	for _, roleID := range req.RoleIDs {
		// Check if role exists
		var role models.Role
		if err := config.DB.First(&role, "id = ?", roleID).Error; err != nil {
			continue // Skip invalid role IDs
		}

		userRole := models.UserRole{
			UserID: userID,
			RoleID: roleID,
		}
		if err := config.DB.Create(&userRole).Error; err != nil {
			return err
		}
	}

	return nil
}
