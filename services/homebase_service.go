package services

import (
	"backend-go/config"
	"backend-go/models"
	"errors"

	"gorm.io/gorm"
)

// GetUserActiveHomebase gets user's currently active homebase
func GetUserActiveHomebase(userID string) (*models.UserBranch, error) {
	var userBranch models.UserBranch

	err := config.DB.
		Preload("Branch").
		Where("user_id = ? AND branch_type = ? AND is_active = ?", userID, "homebase", true).
		First(&userBranch).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("user does not have active homebase branch")
		}
		return nil, err
	}

	if userBranch.Branch == nil {
		return nil, errors.New("branch data not found")
	}

	return &userBranch, nil
}

// GetUserActiveBranchCode gets branch code from user's active homebase
func GetUserActiveBranchCode(userID string) (string, error) {
	homebase, err := GetUserActiveHomebase(userID)
	if err != nil {
		return "", err
	}

	if homebase.Branch == nil {
		return "", errors.New("branch data not found")
	}

	return homebase.Branch.BranchCode, nil
}

// GetFirstHomebaseBranchCode gets branch code from first homebase (fallback)
func GetFirstHomebaseBranchCode(userID string) (string, error) {
	homebases, err := GetUserHomebaseBranches(userID)
	if err != nil {
		return "", err
	}

	if len(homebases) == 0 {
		return "", errors.New("user has no homebase branch")
	}

	// Find active homebase first
	for _, hb := range homebases {
		if hb.IsActive && hb.Branch != nil {
			return hb.Branch.BranchCode, nil
		}
	}

	// Fallback to first homebase if no active
	if homebases[0].Branch != nil {
		return homebases[0].Branch.BranchCode, nil
	}

	return "", errors.New("branch data not found")
}
