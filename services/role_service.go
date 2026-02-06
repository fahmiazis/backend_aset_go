package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"

	"gorm.io/gorm"
)

// GetAllRoles retrieves all roles
func GetAllRoles() ([]dto.RoleDetailResponse, error) {
	var roles []models.Role
	if err := config.DB.Order("name ASC").Find(&roles).Error; err != nil {
		return nil, err
	}

	response := make([]dto.RoleDetailResponse, len(roles))
	for i, role := range roles {
		response[i] = dto.RoleDetailResponse{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
			CreatedAt:   role.CreatedAt,
			UpdatedAt:   role.UpdatedAt,
		}
	}

	return response, nil
}

// GetRoleByID retrieves a role by ID
func GetRoleByID(id string) (*dto.RoleDetailResponse, error) {
	var role models.Role
	if err := config.DB.First(&role, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("role not found")
		}
		return nil, err
	}

	response := &dto.RoleDetailResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}

	return response, nil
}

// CreateRole creates a new role
func CreateRole(req dto.CreateRoleRequest) (*dto.RoleDetailResponse, error) {
	// Check if role name already exists
	var existing models.Role
	if err := config.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return nil, errors.New("role name already exists")
	}

	role := models.Role{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := config.DB.Create(&role).Error; err != nil {
		return nil, err
	}

	response := &dto.RoleDetailResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}

	return response, nil
}

// UpdateRole updates an existing role
func UpdateRole(id string, req dto.UpdateRoleRequest) (*dto.RoleDetailResponse, error) {
	var role models.Role
	if err := config.DB.First(&role, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("role not found")
		}
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Name != "" {
		// Check if name is taken by another role
		var existing models.Role
		if err := config.DB.Where("name = ? AND id != ?", req.Name, id).First(&existing).Error; err == nil {
			return nil, errors.New("role name already exists")
		}
		updates["name"] = req.Name
	}

	if req.Description != "" {
		updates["description"] = req.Description
	}

	if len(updates) > 0 {
		if err := config.DB.Model(&role).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return GetRoleByID(id)
}

// DeleteRole soft deletes a role
func DeleteRole(id string) error {
	var role models.Role
	if err := config.DB.First(&role, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("role not found")
		}
		return err
	}

	// Check if role is assigned to any users
	var count int64
	if err := config.DB.Model(&models.UserRole{}).Where("role_id = ?", id).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return errors.New("cannot delete role that is assigned to users")
	}

	return config.DB.Delete(&role).Error
}
