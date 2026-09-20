package models

// ============================================================
// Stock Opname Flow — constants
// Tidak butuh struct/tabel baru: header pakai Transaction generik
// (transaction_type = TxStockOpnameFlow di services/stock_opname_flow_service.go),
// item pakai TransactionStockOpname yang sudah ada (models/transaction_stock_opname.go).
// ============================================================

// Stage khusus stock opname — sisanya reuse StageDraft/StageApproval/
// StageFinished/StageRejected dari models/procurement_flow.go (pola yang sama
// dipakai mutation flow).
const (
	StageStockOpnameExecute = "EXECUTE_STOCK_OPNAME"
)

// FlowStockOpnameApproval adalah flow_code yang dicari lewat
// GetApprovalFlowByCodeAndBranch (branch-aware, fallback ke ALL).
const FlowStockOpnameApproval = "STOCK_OPNAME_APPROVAL"
