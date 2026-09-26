package dto

// DashboardAssetValues — kartu atas dashboard. Diambil dari asset_values
// bulan berjalan saja (baris terakhir per aset di bulan itu), aset DISPOSED
// tidak dihitung.
type DashboardAssetValues struct {
	Period                  string  `json:"period"` // YYYY-MM
	TotalAssets             int     `json:"total_assets"`
	AcquisitionValue        float64 `json:"acquisition_value"`
	BookValue               float64 `json:"book_value"`
	AccumulatedDepreciation float64 `json:"accumulated_depreciation"`
}

// DashboardFlowRow — jumlah transaksi per bulan per jenis, dikelompokkan
// menurut hasil akhirnya. Frontend menjumlahkan sesuai jenis yang dipilih.
type DashboardFlowRow struct {
	Month           string `json:"month"` // YYYY-MM
	TransactionType string `json:"transaction_type"`
	Finished        int    `json:"finished"`
	InProgress      int    `json:"in_progress"`
	Rejected        int    `json:"rejected"`
	Cancelled       int    `json:"cancelled"`
}

type DashboardTypeCount struct {
	TransactionType string `json:"transaction_type"`
	Count           int    `json:"count"`
}

type DashboardRecentTransaction struct {
	TransactionNumber string `json:"transaction_number"`
	TransactionType   string `json:"transaction_type"`
	TransactionDate   string `json:"transaction_date"`
	CurrentStage      string `json:"current_stage"`
	Status            string `json:"status"`
	CreatedByName     string `json:"created_by_name"`
}

type DashboardSummaryResponse struct {
	AssetValues DashboardAssetValues `json:"asset_values"`
	// 6 bulan terakhir termasuk bulan berjalan
	Flow []DashboardFlowRow `json:"flow"`
	// jumlah transaksi bulan berjalan per jenis (donut)
	MonthCounts []DashboardTypeCount `json:"month_counts"`
	// transaksi terbaru, maksimal 10 per jenis — difilter per jenis di frontend
	Recent []DashboardRecentTransaction `json:"recent"`
}
