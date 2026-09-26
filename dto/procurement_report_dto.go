package dto

// Report procurement — satu baris per barang
type ProcurementReportRow struct {
	ReportTransactionInfo
	ItemName     string  `json:"item_name"`
	CategoryName string  `json:"category_name"`
	Quantity     int     `json:"quantity"`
	UnitPrice    float64 `json:"unit_price"`
	TotalPrice   float64 `json:"total_price"`
	Distribution string  `json:"distribution"` // "BC000005 (10), C00001 (5)"
	ItemNotes    *string `json:"item_notes"`
}
