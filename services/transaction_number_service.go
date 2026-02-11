package services

import (
	"backend-go/config"
	"backend-go/models"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// TransactionType constants
const (
	TxProcurement = "procurement"  // -IO
	TxDisposal    = "disposal"     // -DPSL
	TxMutation    = "mutation"     // -MTI
	TxStockOpname = "stock_opname" // -OPNM
)

// GetTransactionSuffix returns suffix based on transaction type
func GetTransactionSuffix(txType string) string {
	switch txType {
	case TxProcurement:
		return "IO"
	case TxDisposal:
		return "DPSL"
	case TxMutation:
		return "MTI"
	case TxStockOpname:
		return "OPNM"
	default:
		return "TRX"
	}
}

// GetRomanMonth converts month number to Roman numeral
func GetRomanMonth(month int) string {
	romans := []string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}
	if month < 1 || month > 12 {
		return "I"
	}
	return romans[month-1]
}

// GenerateTransactionNumber generates transaction number
// Format: {sequence}/{branch_code}/{branch_name}/{month}/{year}-{suffix}
// Example: 0001/C00001/HO Jakarta/I/2025-IO
func GenerateTransactionNumber(userID string, transactionType string) (string, error) {
	// 1. Get user's active homebase
	var userBranch models.UserBranch
	err := config.DB.
		Preload("Branch").
		Where("user_id = ? AND branch_type = ? AND is_active = ?", userID, "homebase", true).
		First(&userBranch).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", errors.New("user does not have active homebase branch")
		}
		return "", err
	}

	if userBranch.Branch == nil {
		return "", errors.New("branch data not found")
	}

	branchCode := userBranch.Branch.BranchCode
	branchName := userBranch.Branch.BranchName

	// 2. Get current month and year
	now := time.Now()
	month := int(now.Month())
	year := now.Year()
	romanMonth := GetRomanMonth(month)

	// 3. Get last sequence number for this month and transaction type
	var lastReservoir models.Reservoir
	err = config.DB.
		Where("transaksi = ? AND kode_plant = ? AND MONTH(created_at) = ? AND YEAR(created_at) = ?",
			transactionType, branchCode, month, year).
		Order("created_at DESC").
		First(&lastReservoir).Error

	sequence := 1
	if err == nil {
		// Parse sequence from last transaction number
		// Format: 0001/CODE/NAME/I/2025-IO
		parts := strings.Split(lastReservoir.NoTransaksi, "/")
		if len(parts) > 0 {
			var lastSeq int
			fmt.Sscanf(parts[0], "%d", &lastSeq)
			sequence = lastSeq + 1
		}
	}

	// 4. Format sequence with leading zeros (4 digits)
	seqStr := fmt.Sprintf("%04d", sequence)

	// 5. Get suffix based on transaction type
	suffix := GetTransactionSuffix(transactionType)

	// 6. Generate transaction number
	// Format: {sequence}/{branch_code}/{branch_name}/{roman_month}/{year}-{suffix}
	transactionNumber := fmt.Sprintf("%s/%s/%s/%s/%d-%s",
		seqStr, branchCode, branchName, romanMonth, year, suffix)

	// 7. Create reservoir entry with status 'delayed'
	reservoir := models.Reservoir{
		NoTransaksi: transactionNumber,
		KodePlant:   branchCode,
		Transaksi:   transactionType,
		Tipe:        "area", // default tipe
		Status:      "delayed",
		BranchID:    &userBranch.BranchID,
		UserID:      &userID,
	}

	if err := config.DB.Create(&reservoir).Error; err != nil {
		return "", err
	}

	return transactionNumber, nil
}

// MarkTransactionAsUsed marks transaction number as used (when submitted)
func MarkTransactionAsUsed(transactionNumber string) error {
	var reservoir models.Reservoir
	if err := config.DB.Where("no_transaksi = ?", transactionNumber).First(&reservoir).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("transaction number not found in reservoir")
		}
		return err
	}

	if reservoir.Status == "used" {
		return errors.New("transaction number already used")
	}

	updates := map[string]interface{}{
		"status": "used",
	}

	return config.DB.Model(&reservoir).Updates(updates).Error
}

// MarkTransactionAsExpired marks transaction number as expired (when cancelled/replaced)
func MarkTransactionAsExpired(transactionNumber string) error {
	var reservoir models.Reservoir
	if err := config.DB.Where("no_transaksi = ?", transactionNumber).First(&reservoir).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("transaction number not found in reservoir")
		}
		return err
	}

	updates := map[string]interface{}{
		"status": "expired",
	}

	return config.DB.Model(&reservoir).Updates(updates).Error
}

// GetTransactionStatus gets status of a transaction number
func GetTransactionStatus(transactionNumber string) (string, error) {
	var reservoir models.Reservoir
	if err := config.DB.Where("no_transaksi = ?", transactionNumber).First(&reservoir).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", errors.New("transaction number not found")
		}
		return "", err
	}

	return reservoir.Status, nil
}

// GetUserHomebaseBranches returns all homebase branches for a user
func GetUserHomebaseBranches(userID string) ([]models.UserBranch, error) {
	var userBranches []models.UserBranch
	err := config.DB.
		Preload("Branch").
		Where("user_id = ? AND branch_type = ?", userID, "homebase").
		Find(&userBranches).Error

	if err != nil {
		return nil, err
	}

	return userBranches, nil
}

// SetActiveHomebase sets which homebase is active for the user
func SetActiveHomebase(userID, branchID string) error {
	// First, deactivate all homebases for this user
	err := config.DB.Model(&models.UserBranch{}).
		Where("user_id = ? AND branch_type = ?", userID, "homebase").
		Update("is_active", false).Error

	if err != nil {
		return err
	}

	// Then activate the selected homebase
	err = config.DB.Model(&models.UserBranch{}).
		Where("user_id = ? AND branch_id = ? AND branch_type = ?", userID, branchID, "homebase").
		Update("is_active", true).Error

	if err != nil {
		return err
	}

	return nil
}
