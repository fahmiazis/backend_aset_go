package dto

// Report disposal — satu baris per aset
type DisposalReportRow struct {
	ReportTransactionInfo
	DisposalType    string   `json:"disposal_type"`
	AgreementNumber *string  `json:"agreement_number"`
	AssetNumber     string   `json:"asset_number"`
	AssetName       string   `json:"asset_name"`
	CategoryName    string   `json:"category_name"`
	DisposalReason  *string  `json:"disposal_reason"`
	SaleValue       *float64 `json:"sale_value"`
	IncomeValue     *float64 `json:"income_value"`
	InvoiceNumber   *string  `json:"invoice_number"`
	InvoiceDate     *string  `json:"invoice_date"`
	DocumentNumber  *string  `json:"document_number"`
	AssetStatus     string   `json:"asset_status"` // status baris: PENDING / DELETED / CANCELLED
}
