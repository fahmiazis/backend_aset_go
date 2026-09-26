package dto

// Report mutation — satu baris per aset
type MutationReportRow struct {
	ReportTransactionInfo
	AssetNumber    string  `json:"asset_number"`
	AssetName      string  `json:"asset_name"`
	CategoryName   string  `json:"category_name"`
	FromBranchCode string  `json:"from_branch_code"`
	FromBranchName string  `json:"from_branch_name"`
	ToBranchCode   string  `json:"to_branch_code"`
	ToBranchName   string  `json:"to_branch_name"`
	FromLocation   *string `json:"from_location"`
	ToLocation     *string `json:"to_location"`
	DocumentNumber *string `json:"document_number"`
	AssetStatus    string  `json:"asset_status"` // status baris: PENDING / EXECUTED / CANCELLED
}
