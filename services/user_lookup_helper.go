package services

import (
	"backend-go/config"
	"backend-go/models"
)

// ============================================================
// USER LOOKUP HELPER
// Resolve user_id (UUID) → fullname, dipakai untuk mengisi
// created_by_name di response transaksi (procurement, mutation, disposal).
// ============================================================

// resolveUserFullname mengembalikan fullname user, atau nil kalau
// userID kosong / user tidak ditemukan.
func resolveUserFullname(userID string) *string {
	if userID == "" {
		return nil
	}

	var user models.User
	if err := config.DB.
		Select("fullname").
		Where("id = ?", userID).
		First(&user).Error; err != nil {
		return nil
	}

	return &user.Fullname
}

// resolveUserFullnames melakukan lookup batch untuk sekumpulan userID,
// dipakai saat memetakan list agar tidak N+1 query.
func resolveUserFullnames(userIDs []string) map[string]string {
	result := make(map[string]string)

	unique := make([]string, 0, len(userIDs))
	seen := make(map[string]bool)
	for _, id := range userIDs {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}

	if len(unique) == 0 {
		return result
	}

	var users []models.User
	if err := config.DB.
		Select("id", "fullname").
		Where("id IN ?", unique).
		Find(&users).Error; err != nil {
		return result
	}

	for _, u := range users {
		result[u.ID] = u.Fullname
	}

	return result
}
