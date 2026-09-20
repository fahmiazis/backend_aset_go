package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ============================================================================
// BRANCH MEMBER SERVICE
//
// Pengelolaan anggota homebase dari sisi CABANG: pilih banyak user sekaligus
// untuk dijadikan homebase di satu cabang.
//
// Berbeda dengan SetActiveHomebase (transaction_number_service.go) yang bersifat
// self-service — user memilih homebase aktifnya sendiri berdasarkan token.
//
// Aturan: satu user boleh punya beberapa baris homebase, tapi hanya satu yang
// is_active. Saat user di-set ke cabang ini, homebase lamanya TIDAK dihapus,
// hanya dinonaktifkan.
// ============================================================================

const (
	branchTypeHomebase   = "homebase"
	branchTypeAssignment = "assignment"
	branchTypeTemporary  = "temporary"
)

// branchTypesAssignment — tipe yang dianggap "akses cabang tambahan".
// Dipisah dari homebase karena homebase menentukan kode cabang pada nomor
// transaksi, sedangkan assignment/temporary hanya memberi akses.
var branchTypesAssignment = []string{branchTypeAssignment, branchTypeTemporary}

// GetBranchHomebaseUsers mengambil user yang homebase-nya cabang ini.
func GetBranchHomebaseUsers(branchID string) ([]dto.HomebaseUserResponse, error) {
	var branch models.Branch
	if err := config.DB.First(&branch, "id = ?", branchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("branch not found")
		}
		return nil, err
	}

	var rows []dto.HomebaseUserResponse
	err := config.DB.
		Table("user_branchs AS ub").
		Select("u.id AS id, u.username AS username, u.fullname AS fullname, u.email AS email, u.status AS status, ub.is_active AS is_active").
		Joins("JOIN users u ON u.id = ub.user_id AND u.deleted_at IS NULL").
		Where("ub.branch_id = ? AND ub.branch_type = ?", branchID, branchTypeHomebase).
		Order("u.fullname ASC").
		Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	if rows == nil {
		rows = []dto.HomebaseUserResponse{}
	}

	return rows, nil
}

// AssignHomebaseUsers menjadikan cabang ini sebagai homebase aktif bagi
// sekumpulan user sekaligus. Homebase lama tiap user dinonaktifkan (tetap
// tersimpan), lalu baris untuk cabang ini dibuat/diaktifkan.
func AssignHomebaseUsers(branchID string, userIDs []string) error {
	var branch models.Branch
	if err := config.DB.First(&branch, "id = ?", branchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("branch not found")
		}
		return err
	}

	// Buang duplikat
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
		return errors.New("no users selected")
	}

	// Pastikan semua user ada
	var count int64
	if err := config.DB.Model(&models.User{}).
		Where("id IN ?", unique).
		Count(&count).Error; err != nil {
		return err
	}
	if int(count) != len(unique) {
		return errors.New("one or more users not found")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, userID := range unique {
		// Nonaktifkan homebase lain milik user ini (tidak dihapus)
		if err := tx.Model(&models.UserBranch{}).
			Where("user_id = ? AND branch_type = ? AND branch_id <> ?",
				userID, branchTypeHomebase, branchID).
			Update("is_active", false).Error; err != nil {
			tx.Rollback()
			return err
		}

		// Buat atau aktifkan baris untuk cabang ini
		var existing models.UserBranch
		err := tx.Where("user_id = ? AND branch_id = ?", userID, branchID).
			First(&existing).Error

		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Create(&models.UserBranch{
				UserID:     userID,
				BranchID:   branchID,
				BranchType: branchTypeHomebase,
				IsActive:   true,
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		case err == nil:
			// Baris sudah ada (mungkin bertipe temporary/assignment) —
			// naikkan jadi homebase aktif
			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"branch_type": branchTypeHomebase,
				"is_active":   true,
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		default:
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// RemoveHomebaseUser melepas satu user dari homebase cabang ini.
func RemoveHomebaseUser(branchID, userID string) error {
	result := config.DB.
		Where("user_id = ? AND branch_id = ? AND branch_type = ?",
			userID, branchID, branchTypeHomebase).
		Delete(&models.UserBranch{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user is not a homebase member of this branch")
	}

	return nil
}

// ============================================================================
// BRANCH ASSIGNMENT — akses cabang tambahan (bukan homebase)
//
// Berbeda dengan homebase yang hanya boleh satu yang aktif per user,
// assignment bersifat additive: satu user boleh punya akses ke banyak cabang.
//
// Catatan: approval_service.go memberi approver akses ke transaksi sebuah
// cabang berdasarkan SEMUA baris user_branchs apa pun tipenya — jadi anggota
// homebase cabang ini otomatis sudah punya akses dan tidak perlu di-assign lagi.
// Tabel user_branchs punya UNIQUE (user_id, branch_id), sehingga satu user
// hanya bisa punya satu baris per cabang.
// ============================================================================

// GetBranchAssignedUsers mengambil user yang di-assign ke cabang ini
// (tipe assignment/temporary, di luar anggota homebase).
func GetBranchAssignedUsers(branchID string) ([]dto.BranchAssignedUserResponse, error) {
	var branch models.Branch
	if err := config.DB.First(&branch, "id = ?", branchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("branch not found")
		}
		return nil, err
	}

	var rows []dto.BranchAssignedUserResponse
	err := config.DB.
		Table("user_branchs AS ub").
		Select("u.id AS id, u.username AS username, u.fullname AS fullname, u.email AS email, u.status AS status, ub.branch_type AS branch_type").
		Joins("JOIN users u ON u.id = ub.user_id AND u.deleted_at IS NULL").
		Where("ub.branch_id = ? AND ub.branch_type IN ?", branchID, branchTypesAssignment).
		Order("u.fullname ASC").
		Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	if rows == nil {
		rows = []dto.BranchAssignedUserResponse{}
	}

	return rows, nil
}

// AssignBranchUsers memberi sekumpulan user akses ke cabang ini.
// Tidak mengubah homebase siapa pun: user yang sudah punya baris di cabang ini
// (termasuk sebagai homebase) dilewati, karena aksesnya sudah ada.
func AssignBranchUsers(branchID string, userIDs []string) error {
	var branch models.Branch
	if err := config.DB.First(&branch, "id = ?", branchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("branch not found")
		}
		return err
	}

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
		return errors.New("no users selected")
	}

	var count int64
	if err := config.DB.Model(&models.User{}).
		Where("id IN ?", unique).
		Count(&count).Error; err != nil {
		return err
	}
	if int(count) != len(unique) {
		return errors.New("one or more users not found")
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, userID := range unique {
		var existing models.UserBranch
		err := tx.Where("user_id = ? AND branch_id = ?", userID, branchID).
			First(&existing).Error

		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			// is_active dibiarkan false: kolom itu hanya bermakna untuk homebase
			if err := tx.Create(&models.UserBranch{
				UserID:     userID,
				BranchID:   branchID,
				BranchType: branchTypeAssignment,
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		case err == nil:
			// Sudah punya baris di cabang ini — biarkan apa adanya.
			// Kalau tipenya homebase, menurunkannya jadi assignment akan
			// merusak penomoran transaksi user tersebut.
			continue
		default:
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// RemoveBranchUser mencabut akses cabang seorang user.
// Baris homebase tidak ikut terhapus — gunakan RemoveHomebaseUser untuk itu.
func RemoveBranchUser(branchID, userID string) error {
	result := config.DB.
		Where("user_id = ? AND branch_id = ? AND branch_type IN ?",
			userID, branchID, branchTypesAssignment).
		Delete(&models.UserBranch{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user is not assigned to this branch")
	}

	return nil
}
