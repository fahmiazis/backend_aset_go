package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"

	"gorm.io/gorm"
)

// GetAllBranchs retrieves all branchs
func GetAllBranchs() ([]dto.BranchResponse, error) {
	var branchs []models.Branch
	if err := config.DB.Order("branch_code ASC").Find(&branchs).Error; err != nil {
		return nil, err
	}

	response := make([]dto.BranchResponse, len(branchs))
	for i, branch := range branchs {
		response[i] = dto.BranchResponse{
			ID:         branch.ID,
			BranchCode: branch.BranchCode,
			BranchName: branch.BranchName,
			BranchType: branch.BranchType,
			Status:     branch.Status,
			CreatedAt:  branch.CreatedAt,
			UpdatedAt:  branch.UpdatedAt,
		}
	}

	return response, nil
}

// GetBranchByID retrieves a branch by ID
func GetBranchByID(id string) (*dto.BranchResponse, error) {
	var branch models.Branch
	if err := config.DB.First(&branch, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("branch not found")
		}
		return nil, err
	}

	response := &dto.BranchResponse{
		ID:         branch.ID,
		BranchCode: branch.BranchCode,
		BranchName: branch.BranchName,
		BranchType: branch.BranchType,
		Status:     branch.Status,
		CreatedAt:  branch.CreatedAt,
		UpdatedAt:  branch.UpdatedAt,
	}

	return response, nil
}

// GetBranchByKode retrieves a branch by branch_code
func GetBranchByKode(BranchCode string) (*dto.BranchResponse, error) {
	var branch models.Branch
	if err := config.DB.Where("branch_code = ?", BranchCode).First(&branch).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("branch not found")
		}
		return nil, err
	}

	response := &dto.BranchResponse{
		ID:         branch.ID,
		BranchCode: branch.BranchCode,
		BranchName: branch.BranchName,
		BranchType: branch.BranchType,
		Status:     branch.Status,
		CreatedAt:  branch.CreatedAt,
		UpdatedAt:  branch.UpdatedAt,
	}

	return response, nil
}

// CreateBranch creates a new branch
func CreateBranch(req dto.CreateBranchRequest) (*dto.BranchResponse, error) {
	// Set default status if not provided
	status := req.Status
	if status == "" {
		status = "active"
	}

	branch := models.Branch{
		BranchName: req.BranchName,
		BranchType: req.BranchType,
		Status:     status,
	}

	// branch_code will be auto-generated in BeforeCreate hook
	if err := config.DB.Create(&branch).Error; err != nil {
		return nil, err
	}

	response := &dto.BranchResponse{
		ID:         branch.ID,
		BranchCode: branch.BranchCode,
		BranchName: branch.BranchName,
		BranchType: branch.BranchType,
		Status:     branch.Status,
		CreatedAt:  branch.CreatedAt,
		UpdatedAt:  branch.UpdatedAt,
	}

	return response, nil
}

// UpdateBranch updates an existing branch
func UpdateBranch(id string, req dto.UpdateBranchRequest) (*dto.BranchResponse, error) {
	var branch models.Branch

	// Check if branch exists
	if err := config.DB.First(&branch, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("branch not found")
		}
		return nil, err
	}

	// Build update map
	updates := make(map[string]interface{})
	if req.BranchName != "" {
		updates["branch_name"] = req.BranchName
	}
	if req.BranchType != "" {
		updates["branch_type"] = req.BranchType
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	// Update branch
	if err := config.DB.Model(&branch).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Return updated branch
	return GetBranchByID(id)
}

// DeleteBranch soft deletes a branch
func DeleteBranch(id string) error {
	var branch models.Branch
	if err := config.DB.First(&branch, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("branch not found")
		}
		return err
	}

	return config.DB.Delete(&branch).Error
}

// AssignBranchsToUser assigns multiple branchs to a user
func AssignBranchsToUser(userID string, req dto.AssignBranchRequest) error {
	// Check if user exists
	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("user not found")
		}
		return err
	}

	// Verify all branchs exist
	var branchs []models.Branch
	if err := config.DB.Where("id IN ?", req.BranchIDs).Find(&branchs).Error; err != nil {
		return err
	}
	if len(branchs) != len(req.BranchIDs) {
		return errors.New("one or more branchs not found")
	}

	// Delete existing assignments
	if err := config.DB.Where("user_id = ?", userID).Delete(&models.UserBranch{}).Error; err != nil {
		return err
	}

	// Create new assignments
	for _, branchID := range req.BranchIDs {
		userBranch := models.UserBranch{
			UserID:   userID,
			BranchID: branchID,
		}
		if err := config.DB.Create(&userBranch).Error; err != nil {
			return err
		}
	}

	return nil
}

// GetUserBranchs retrieves all branchs assigned to a user
func GetUserBranchs(userID string) ([]dto.BranchResponse, error) {
	var branchs []models.Branch

	err := config.DB.Table("branchs").
		Joins("JOIN user_branchs ON user_branchs.branch_id = branchs.id").
		Where("user_branchs.user_id = ?", userID).
		Order("branchs.branch_code ASC").
		Find(&branchs).Error

	if err != nil {
		return nil, err
	}

	response := make([]dto.BranchResponse, len(branchs))
	for i, branch := range branchs {
		response[i] = dto.BranchResponse{
			ID:         branch.ID,
			BranchCode: branch.BranchCode,
			BranchName: branch.BranchName,
			BranchType: branch.BranchType,
			Status:     branch.Status,
			CreatedAt:  branch.CreatedAt,
			UpdatedAt:  branch.UpdatedAt,
		}
	}

	return response, nil
}

// GetBranchUsers retrieves all users assigned to a branch
func GetBranchUsers(branchID string) ([]dto.UserSimpleResponse, error) {
	var users []models.User

	err := config.DB.Table("users").
		Joins("JOIN user_branchs ON user_branchs.user_id = users.id").
		Where("user_branchs.branch_id = ?", branchID).
		Find(&users).Error

	if err != nil {
		return nil, err
	}

	response := make([]dto.UserSimpleResponse, len(users))
	for i, user := range users {
		response[i] = dto.UserSimpleResponse{
			ID:       user.ID,
			Username: user.Username,
			Fullname: user.Fullname,
			Email:    user.Email,
			Status:   user.Status,
		}
	}

	return response, nil
}

// RemoveBranchFromUser removes a specific branch from a user
func RemoveBranchFromUser(userID, branchID string) error {
	result := config.DB.Where("user_id = ? AND branch_id = ?", userID, branchID).
		Delete(&models.UserBranch{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("assignment not found")
	}

	return nil
}
