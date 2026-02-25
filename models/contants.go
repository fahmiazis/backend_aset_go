package models

// TransactionStatus constants (shared across all transaction tables)
const (
	TransactionStatusDraft    = "DRAFT"
	TransactionStatusApproved = "APPROVED"
	TransactionStatusRejected = "REJECTED"
)

const (
	TransactionTypeAcquisition  = "ACQUISITION"
	TransactionTypeMutation     = "MUTATION"
	TransactionTypeDisposal     = "DISPOSAL"
	TransactionTypeStockOpname  = "STOCK_OPNAME"
	TransactionTypeProcurement  = "PROCUREMENT"
	TransactionTypeDepreciation = "DEPRECIATION"
	TransactionTypeValueUpdate  = "VALUE_UPDATE"
)

const (
	DisposalMethodSale     = "SALE"
	DisposalMethodScrap    = "SCRAP"
	DisposalMethodDonate   = "DONATE"
	DisposalMethodWriteOff = "WRITE_OFF"
)
