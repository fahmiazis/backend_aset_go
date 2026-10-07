package services

import (
	"backend-go/config"
	"backend-go/models"
	"fmt"

	"gorm.io/gorm"
)

// Pemetaan stage stock opname ke pengerjanya (dipakai email notifikasi):
//   - DRAFT → pembuatnya + semua user dengan homebase aktif di cabang
//     stock opname itu (draft milik cabang, lihat stock_opname_active.go)
//   - APPROVAL tidak terdaftar → ditentukan giliran step STOCK_OPNAME_APPROVAL
//   - EXECUTE_STOCK_OPNAME → execute_stock_opname di menu Stock Opname Execute
var stockOpnameWaiting = waitingConfig{
	txType:     TxStockOpnameFlow,
	draftStage: models.StageDraft,
	owners: map[string]stageOwner{
		models.StageStockOpnameExecute: {
			routePath:   "/transactions/stock-opname/execute",
			permissions: []string{"execute_stock_opname"},
		},
	},
	extraCondition: stockOpnameBranchDraftCondition,
	extraCheck:     isStockOpnameBranchDraft,
}

func applyStockOpnameWaitingFilter(query *gorm.DB, userID string) *gorm.DB {
	return applyWaitingFilter(query, userID, stockOpnameWaiting)
}

// IsStockOpnameWaitingForUser dipakai halaman detail stock opname.
func IsStockOpnameWaitingForUser(userID, transactionNumber string) bool {
	transaction, err := getStockOpnameTransaction(transactionNumber)
	if err != nil {
		return false
	}
	return isWaitingForUser(userID, transaction, stockOpnameWaiting)
}

// stockOpnameFindingSummary —ringkasan temuan per status fisik untuk email.
// Aset stock opname = seluruh aset cabang (bisa ratusan), jadi yang dikirim
// rekapnya, bukan daftar aset. Sebelum submit item masih di tabel draft.
func stockOpnameFindingSummary(transactionID uint) (title string, columns []string, rows [][]string) {
	type bucket struct {
		PhysicalStatus string
		Total          int
	}
	var buckets []bucket

	query := func(table string) {
		config.DB.Table(table).
			Select("physical_status, COUNT(*) AS total").
			Where("transaction_id = ?", transactionID).
			Group("physical_status").
			Order("total DESC").
			Scan(&buckets)
	}
	query(models.TransactionStockOpname{}.TableName())
	if len(buckets) == 0 {
		query(models.StockOpnameDraftItem{}.TableName())
	}

	total := 0
	for _, b := range buckets {
		total += b.Total
	}

	title = fmt.Sprintf("Ringkasan Temuan (%d aset)", total)
	columns = []string{"Status Fisik", "Jumlah Aset"}
	for _, b := range buckets {
		status := b.PhysicalStatus
		if status == "" {
			status = "Belum diisi"
		}
		rows = append(rows, []string{status, fmt.Sprint(b.Total)})
	}
	return title, columns, rows
}
