package dto

// ============================================================
// REQUEST
// ============================================================

// StockOpnameReportFilter dipakai bersama oleh dashboard, detail report,
// maupun export excel supaya ketiganya selalu menampilkan angka yang konsisten
// untuk kombinasi period + branch yang sama.
type StockOpnameReportFilter struct {
	Month      int    `form:"month"`       // 1-12, default: bulan berjalan
	Year       int    `form:"year"`        // default: tahun berjalan
	BranchCode string `form:"branch_code"` // kosong / "ALL" = semua cabang ("Plant: Semua")
}

// ============================================================
// SHARED PIECES
// ============================================================

// StockOpnameStatusBreakdown adalah bucket status yang dipakai berulang di
// stats card, chart "status per grouping", dan chart "status submit".
type StockOpnameStatusBreakdown struct {
	Finish      int64 `json:"finish"`       // current_stage = FINISHED
	InProgress  int64 `json:"in_progress"`  // current_stage = DRAFT (belum pernah direvisi) / APPROVAL / EXECUTE_STOCK_OPNAME
	BelumSubmit int64 `json:"belum_submit"` // asset di scope belum pernah dimasukkan opname periode ini
	Rejected    int64 `json:"rejected"`     // current_stage = REJECTED
	Revisi      int64 `json:"revisi"`       // current_stage = DRAFT hasil ReviseStockOpname (belum di-submit ulang)
	Disposal    int64 `json:"disposal"`     // asset_status ACTIVE record = DISPOSED / IN_DISPOSAL (di luar cakupan opname normal)
}

// ============================================================
// DASHBOARD (gambar 2)
// ============================================================

type StockOpnameReportPeriod struct {
	Month     int    `json:"month"`
	Year      int    `json:"year"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type StockOpnameDashboardStats struct {
	TotalAsset       int64   `json:"total_asset"`
	Finish           int64   `json:"finish"`
	FinishPercentage float64 `json:"finish_percentage"`
	InProgress       int64   `json:"in_progress"`
	BelumSubmit      int64   `json:"belum_submit"`
	Rejected         int64   `json:"rejected"`
	Revisi           int64   `json:"revisi"`
	Disposal         int64   `json:"disposal"`
	AcquisitionValue float64 `json:"acquisition_value"`
	BookValue        float64 `json:"book_value"`
}

// StockOpnameGroupingStatus adalah satu batang di chart "Status per grouping".
// Grouping diambil apa adanya dari assets.grouping (free-text, diisi manual
// oleh tim aset) — bukan enum tetap. Grouping bisa string kosong ("") kalau
// asset belum dikelompokkan — caller (FE) yang menerjemahkan/menampilkan
// label untuk kasus ini, bukan backend.
type StockOpnameGroupingStatus struct {
	Grouping string                     `json:"grouping"`
	Status   StockOpnameStatusBreakdown `json:"status"`
	Total    int64                      `json:"total"`
}

// StockOpnamePhysicalVsSystem = chart "Status fisik vs SAP". Dihitung hanya
// dari asset yang sudah punya temuan (finish/in_progress/rejected), karena
// asset "belum_submit" memang belum punya data fisik untuk dibandingkan.
//
// Catatan: SystemTidakAda akan SELALU 0. Skema saat ini tidak punya jalur
// untuk mencatat "aset ditemukan secara fisik tapi tidak terdaftar di
// sistem" (endpoint add-asset selalu mensyaratkan asset_id yang sudah
// terdaftar) — lihat juga kolom "FISIK ADA SAP TIDAK ADA" di detail report.
type StockOpnamePhysicalVsSystem struct {
	PhysicalAda      int64 `json:"physical_ada"`       // found_physical_status = EXISTS
	PhysicalTidakAda int64 `json:"physical_tidak_ada"` // found_physical_status = MISSING
	SystemAda        int64 `json:"system_ada"`         // selalu = physical_ada + physical_tidak_ada (lihat catatan)
	SystemTidakAda   int64 `json:"system_tidak_ada"`   // selalu 0, lihat catatan
}

// StockOpnameConditionSummary = chart "Kondisi aset" (donut Baik/Rusak/Tidak Ada/Belum Isi)
type StockOpnameConditionSummary struct {
	Baik     int64 `json:"baik"`      // found_condition GOOD/FAIR
	Rusak    int64 `json:"rusak"`     // found_condition POOR/BROKEN
	TidakAda int64 `json:"tidak_ada"` // found_physical_status MISSING
	BelumIsi int64 `json:"belum_isi"` // belum ada temuan (belum_submit)
}

type StockOpnameDashboardCharts struct {
	StatusPerGrouping []StockOpnameGroupingStatus `json:"status_per_grouping"`
	PhysicalVsSystem  StockOpnamePhysicalVsSystem `json:"physical_vs_system"`
	ConditionSummary  StockOpnameConditionSummary `json:"condition_summary"`
	StatusSubmit      StockOpnameStatusBreakdown  `json:"status_submit"`
}

type StockOpnameDashboardResponse struct {
	Period     StockOpnameReportPeriod    `json:"period"`
	BranchCode string                     `json:"branch_code"` // "ALL" kalau tidak difilter
	Stats      StockOpnameDashboardStats  `json:"stats"`
	Charts     StockOpnameDashboardCharts `json:"charts"`
}

// ============================================================
// DETAIL REPORT (gambar 1 & 3 — tabel rekapitulasi + gambar 4 — cost center)
// ============================================================

// StockOpnameRekapRow = satu baris di tabel rekapitulasi (kolom Acquis.val /
// Accum.dep / Book val / Eksekusi seperti di contoh). Beberapa baris di
// contoh referensi (mis. "SAP ADA FISIK TIDAK - KENDARAAN", tiga baris
// "FISIK ADA SAP TIDAK ADA - ...") tidak punya jalur data di skema saat ini
// (tidak ada flag kendaraan di kategori aset, tidak ada jalur input aset
// fisik yang belum terdaftar di sistem) — baris-baris itu tetap dikirim
// dengan value 0 dan Supported=false supaya FE bisa menampilkan badge
// "belum didukung" alih-alih diam-diam menampilkan 0 yang menyesatkan.
type StockOpnameRekapRow struct {
	Label                   string  `json:"label"`
	AcquisitionValue        float64 `json:"acquisition_value"`
	AccumulatedDepreciation float64 `json:"accumulated_depreciation"`
	BookValue               float64 `json:"book_value"`
	UnitCount               int64   `json:"unit_count"`
	ExtraInfo               string  `json:"extra_info,omitempty"` // mis. "36 area; 13 ho" untuk baris "Total Area yang tidak kirim"
	Supported               bool    `json:"supported"`            // false = baris ini belum bisa dihitung dari skema saat ini
}

type StockOpnameAreaSummary struct {
	AreaClearCount         int64   `json:"area_clear_count"`
	AreaClearPercentage    float64 `json:"area_clear_percentage"`
	AreaNotClearCount      int64   `json:"area_not_clear_count"`
	AreaNotClearPercentage float64 `json:"area_not_clear_percentage"`
	TotalArea              int64   `json:"total_area"`
}

// StockOpnameCostCenterRow = satu baris chart "Nilai aset per cost center".
// Tidak ada tabel cost center terpisah di skema saat ini — "cost center"
// pada laporan ini adalah Branch (branch_code + branch_name), dipakai
// sebagai proxy terdekat yang tersedia.
type StockOpnameCostCenterRow struct {
	BranchCode       string  `json:"branch_code"`
	BranchName       string  `json:"branch_name"`
	AcquisitionValue float64 `json:"acquisition_value"`
	BookValue        float64 `json:"book_value"`
	UnitCount        int64   `json:"unit_count"`
}

type StockOpnameDetailReportResponse struct {
	Period      StockOpnameReportPeriod    `json:"period"`
	BranchCode  string                     `json:"branch_code"`
	Rekap       []StockOpnameRekapRow      `json:"rekap"`
	AreaSummary StockOpnameAreaSummary     `json:"area_summary"`
	CostCenters []StockOpnameCostCenterRow `json:"cost_centers_top10"`
	Note        string                     `json:"note"`
}

// ============================================================
// EXPORT EXCEL
// ============================================================

// StockOpnameExportRequest — sama seperti StockOpnameReportFilter, dipisah
// supaya query binding di controller untuk endpoint export tidak tercampur
// dengan endpoint JSON (meski isinya sama saat ini).
type StockOpnameExportRequest struct {
	Month      int    `form:"month"`
	Year       int    `form:"year"`
	BranchCode string `form:"branch_code"`
}
