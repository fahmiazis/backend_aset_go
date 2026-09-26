package dto

// ============================================================
// REPORT TRANSAKSI — procurement, mutation, disposal
// ============================================================

// TransactionReportFilter dipakai bersama oleh data JSON maupun export excel,
// supaya file yang diunduh selalu sama dengan tabel yang sedang dilihat.
type TransactionReportFilter struct {
	StartDate  string `form:"start_date"`  // YYYY-MM-DD, berdasarkan transaction_date
	EndDate    string `form:"end_date"`    // YYYY-MM-DD, inklusif
	BranchCode string `form:"branch_code"` // kosong = semua cabang yang boleh dilihat user
	Stage      string `form:"stage"`       // kosong = semua stage
	Search     string `form:"search"`      // nomor transaksi, catatan, nama barang / aset
}

type ReportStageCount struct {
	Stage string `json:"stage"`
	Count int    `json:"count"` // jumlah transaksi, bukan baris
}

// TransactionReportSummary — tidak semua field terpakai di tiap report:
// quantity hanya procurement, income hanya disposal.
type TransactionReportSummary struct {
	TotalTransactions int                `json:"total_transactions"`
	TotalRows         int                `json:"total_rows"`
	TotalQuantity     int                `json:"total_quantity"`
	TotalValue        float64            `json:"total_value"`  // procurement: total harga; disposal: nilai jual
	TotalIncome       float64            `json:"total_income"` // disposal: nilai pemasukan
	ByStage           []ReportStageCount `json:"by_stage"`     // dihitung sebelum filter stage, untuk tab
}

// Kolom yang sama di ketiga report
type ReportTransactionInfo struct {
	TransactionNumber string  `json:"transaction_number"`
	TransactionDate   string  `json:"transaction_date"`
	BranchCode        string  `json:"branch_code"` // cabang pengaju
	BranchName        string  `json:"branch_name"`
	CurrentStage      string  `json:"current_stage"`
	Status            string  `json:"status"`
	Notes             *string `json:"notes"`
	CreatedByName     string  `json:"created_by_name"`
}

type TransactionReportResponse[T any] struct {
	Branches    []BranchOption           `json:"branches"`     // pilihan dropdown cabang
	AllBranches bool                     `json:"all_branches"` // admin: tanpa pembatasan cabang
	Summary     TransactionReportSummary `json:"summary"`
	Rows        []T                      `json:"rows"`
}
