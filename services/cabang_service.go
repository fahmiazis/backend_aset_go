package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"

	"gorm.io/gorm"
)

// GetAllCabangs retrieves all cabangs
func GetAllCabangs() ([]dto.CabangResponse, error) {
	var cabangs []models.Cabang
	if err := config.DB.Order("kode_cabang ASC").Find(&cabangs).Error; err != nil {
		return nil, err
	}

	response := make([]dto.CabangResponse, len(cabangs))
	for i, cabang := range cabangs {
		response[i] = dto.CabangResponse{
			ID:           cabang.ID,
			KodeCabang:   cabang.KodeCabang,
			NamaCabang:   cabang.NamaCabang,
			StatusCabang: cabang.StatusCabang,
			CreatedAt:    cabang.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    cabang.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return response, nil
}

// GetCabangByID retrieves a cabang by ID
func GetCabangByID(id string) (*dto.CabangResponse, error) {
	var cabang models.Cabang
	if err := config.DB.First(&cabang, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("cabang not found")
		}
		return nil, err
	}

	response := &dto.CabangResponse{
		ID:           cabang.ID,
		KodeCabang:   cabang.KodeCabang,
		NamaCabang:   cabang.NamaCabang,
		StatusCabang: cabang.StatusCabang,
		CreatedAt:    cabang.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    cabang.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	return response, nil
}

// GetCabangByKode retrieves a cabang by kode_cabang
func GetCabangByKode(kodeCabang string) (*dto.CabangResponse, error) {
	var cabang models.Cabang
	if err := config.DB.Where("kode_cabang = ?", kodeCabang).First(&cabang).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("cabang not found")
		}
		return nil, err
	}

	response := &dto.CabangResponse{
		ID:           cabang.ID,
		KodeCabang:   cabang.KodeCabang,
		NamaCabang:   cabang.NamaCabang,
		StatusCabang: cabang.StatusCabang,
		CreatedAt:    cabang.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    cabang.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	return response, nil
}

// CreateCabang creates a new cabang
func CreateCabang(req dto.CreateCabangRequest) (*dto.CabangResponse, error) {
	// Set default status if not provided
	statusCabang := req.StatusCabang
	if statusCabang == "" {
		statusCabang = "active"
	}

	cabang := models.Cabang{
		NamaCabang:   req.NamaCabang,
		StatusCabang: statusCabang,
	}

	// kode_cabang will be auto-generated in BeforeCreate hook
	if err := config.DB.Create(&cabang).Error; err != nil {
		return nil, err
	}

	response := &dto.CabangResponse{
		ID:           cabang.ID,
		KodeCabang:   cabang.KodeCabang,
		NamaCabang:   cabang.NamaCabang,
		StatusCabang: cabang.StatusCabang,
		CreatedAt:    cabang.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    cabang.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	return response, nil
}

// UpdateCabang updates an existing cabang
func UpdateCabang(id string, req dto.UpdateCabangRequest) (*dto.CabangResponse, error) {
	var cabang models.Cabang

	// Check if cabang exists
	if err := config.DB.First(&cabang, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("cabang not found")
		}
		return nil, err
	}

	// Build update map
	updates := make(map[string]interface{})
	if req.NamaCabang != "" {
		updates["nama_cabang"] = req.NamaCabang
	}
	if req.StatusCabang != "" {
		updates["status_cabang"] = req.StatusCabang
	}

	// Update cabang
	if err := config.DB.Model(&cabang).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Return updated cabang
	return GetCabangByID(id)
}

// DeleteCabang soft deletes a cabang
func DeleteCabang(id string) error {
	var cabang models.Cabang
	if err := config.DB.First(&cabang, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("cabang not found")
		}
		return err
	}

	return config.DB.Delete(&cabang).Error
}

// AssignCabangsToUser assigns multiple cabangs to a user
func AssignCabangsToUser(userID string, req dto.AssignCabangRequest) error {
	// Check if user exists
	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("user not found")
		}
		return err
	}

	// Verify all cabangs exist
	var cabangs []models.Cabang
	if err := config.DB.Where("id IN ?", req.CabangIDs).Find(&cabangs).Error; err != nil {
		return err
	}
	if len(cabangs) != len(req.CabangIDs) {
		return errors.New("one or more cabangs not found")
	}

	// Delete existing assignments
	if err := config.DB.Where("user_id = ?", userID).Delete(&models.UserCabang{}).Error; err != nil {
		return err
	}

	// Create new assignments
	for _, cabangID := range req.CabangIDs {
		userCabang := models.UserCabang{
			UserID:   userID,
			CabangID: cabangID,
		}
		if err := config.DB.Create(&userCabang).Error; err != nil {
			return err
		}
	}

	return nil
}

// GetUserCabangs retrieves all cabangs assigned to a user
func GetUserCabangs(userID string) ([]dto.CabangResponse, error) {
	var cabangs []models.Cabang

	err := config.DB.Table("cabangs").
		Joins("JOIN user_cabangs ON user_cabangs.cabang_id = cabangs.id").
		Where("user_cabangs.user_id = ?", userID).
		Order("cabangs.kode_cabang ASC").
		Find(&cabangs).Error

	if err != nil {
		return nil, err
	}

	response := make([]dto.CabangResponse, len(cabangs))
	for i, cabang := range cabangs {
		response[i] = dto.CabangResponse{
			ID:           cabang.ID,
			KodeCabang:   cabang.KodeCabang,
			NamaCabang:   cabang.NamaCabang,
			StatusCabang: cabang.StatusCabang,
			CreatedAt:    cabang.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    cabang.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return response, nil
}

// GetCabangUsers retrieves all users assigned to a cabang
func GetCabangUsers(cabangID string) ([]dto.UserSimpleResponse, error) {
	var users []models.User

	err := config.DB.Table("users").
		Joins("JOIN user_cabangs ON user_cabangs.user_id = users.id").
		Where("user_cabangs.cabang_id = ?", cabangID).
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

// RemoveCabangFromUser removes a specific cabang from a user
func RemoveCabangFromUser(userID, cabangID string) error {
	result := config.DB.Where("user_id = ? AND cabang_id = ?", userID, cabangID).
		Delete(&models.UserCabang{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("assignment not found")
	}

	return nil
}
