package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ============================================================
// SATU STOCK OPNAME AKTIF PER CABANG
// ============================================================
//
// Satu cabang cuma boleh punya satu stock opname yang masih berjalan (belum
// FINISHED / REJECTED). Draft baru baru bisa dibuat setelah yang lama selesai
// atau ditolak. Karena draftnya milik cabang, semua user dengan homebase
// aktif di cabang itu boleh mengisi temuan & submit — bukan cuma pembuatnya.
// Scan QR di aplikasi mobile langsung diarahkan ke stock opname aktif ini.
//
// Cabang stock opname = segmen kedua nomor transaksi (reportBranchExpr).

// ErrStockOpnameActiveExists — cabang masih punya stock opname berjalan.
type ErrStockOpnameActiveExists struct {
	BranchCode        string
	TransactionNumber string
	Stage             string
}

func (e *ErrStockOpnameActiveExists) Error() string {
	return fmt.Sprintf(
		"cabang %s masih punya stock opname yang berjalan (%s, tahap %s) — selesaikan atau tunggu sampai FINISHED/REJECTED sebelum membuat yang baru",
		e.BranchCode, e.TransactionNumber, e.Stage,
	)
}

// activeStockOpnameQuery — stock opname cabang ini yang belum FINISHED /
// REJECTED. Baris dari endpoint lama /old ikut tercatat sebagai
// stock_opname dengan current_stage DRAFT selamanya, tapi tidak pernah punya
// stock_opname_draft_items — jadi DRAFT tanpa item draft tidak dihitung.
func activeStockOpnameQuery(db *gorm.DB, branchCode string) *gorm.DB {
	return db.Model(&models.Transaction{}).
		Where("transactions.transaction_type = ?", TxStockOpnameFlow).
		Where(reportBranchExpr+" = ?", branchCode).
		Where("transactions.current_stage NOT IN ?", []string{models.StageFinished, models.StageRejected}).
		Where(
			"(transactions.current_stage <> ? OR EXISTS (SELECT 1 FROM stock_opname_draft_items d WHERE d.transaction_id = transactions.id))",
			models.StageDraft,
		)
}

// findActiveStockOpname — nil kalau cabang tidak punya stock opname berjalan.
// Data lama bisa saja punya lebih dari satu; yang terbaru dipakai.
func findActiveStockOpname(db *gorm.DB, branchCode string) (*models.Transaction, error) {
	var transaction models.Transaction
	err := activeStockOpnameQuery(db, branchCode).Order("transactions.created_at DESC").First(&transaction).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

// GetMyActiveStockOpname — stock opname berjalan di cabang homebase aktif
// user (GET /transactions/stock-opname/active). nil = belum ada.
func GetMyActiveStockOpname(userID string) (*dto.StockOpnameFlowDetailResponse, error) {
	branchCode, err := GetUserActiveBranchCode(userID)
	if err != nil {
		return nil, errors.New("anda belum memiliki homebase aktif, silakan set homebase terlebih dahulu")
	}
	transaction, err := findActiveStockOpname(config.DB, branchCode)
	if err != nil || transaction == nil {
		return nil, err
	}
	return GetStockOpnameFlowDetail(transaction.TransactionNumber)
}

// ensureStockOpnameBranchMember — draft stock opname milik cabang, jadi yang
// boleh mengubah/submit adalah user dengan homebase aktif di cabang itu.
func ensureStockOpnameBranchMember(userID string, transaction *models.Transaction) error {
	if transaction.CreatedBy == userID {
		return nil
	}
	branchCode := transactionBranchCode(transaction.TransactionNumber)
	userBranch, err := GetUserActiveBranchCode(userID)
	if err != nil || branchCode == "" || userBranch != branchCode {
		return fmt.Errorf("stock opname ini milik cabang %s — hanya user dengan homebase aktif di cabang tersebut yang bisa mengubahnya", branchCode)
	}
	return nil
}

// stockOpnameBranchDraftCondition / isStockOpnameBranchDraft — draft cabang
// homebase aktif user ikut "menunggu saya" (dipakai stockOpnameWaiting).
func stockOpnameBranchDraftCondition(userID string) *gorm.DB {
	branchCode, err := GetUserActiveBranchCode(userID)
	if err != nil {
		return config.DB.Where("1 = 0")
	}
	return config.DB.Where("current_stage = ? AND "+reportBranchExpr+" = ?", models.StageDraft, branchCode)
}

func isStockOpnameBranchDraft(userID string, transaction *models.Transaction) bool {
	if transaction.CurrentStage != models.StageDraft {
		return false
	}
	branchCode, err := GetUserActiveBranchCode(userID)
	return err == nil && branchCode == transactionBranchCode(transaction.TransactionNumber)
}
