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

// StockOpnameConditionSummary = chart "Kondisi aset" (donut Baik/Rusak/Tidak Ada/Belum Isi)
type StockOpnameConditionSummary struct {
	Baik     int64 `json:"baik"`      // found_condition GOOD/FAIR
	Rusak    int64 `json:"rusak"`     // found_condition POOR/BROKEN
	TidakAda int64 `json:"tidak_ada"` // found_physical_status MISSING
	BelumIsi int64 `json:"belum_isi"` // belum ada temuan (belum_submit)
}

type StockOpnameDashboardCharts struct {
	StatusPerGrouping []StockOpnameGroupingStatus `json:"status_per_grouping"`
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
// DETAIL REPORT (gambar 1 & 3 — tabel rekapitulasi)
// ============================================================

// StockOpnameRekapRow = satu baris di tabel rekapitulasi (kolom Acquis.val /
// Accum.dep / Book val / Eksekusi seperti di contoh). Supported=false
// dipakai buat baris yang belum bisa dihitung dari skema saat ini — FE
// nampilin badge "belum didukung" alih-alih angka 0 yang menyesatkan.
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

type StockOpnameDetailReportResponse struct {
	Period      StockOpnameReportPeriod `json:"period"`
	BranchCode  string                  `json:"branch_code"`
	Rekap       []StockOpnameRekapRow   `json:"rekap"`
	AreaSummary StockOpnameAreaSummary  `json:"area_summary"`
	Note        string                  `json:"note"`
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
