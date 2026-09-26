package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ============================================================
// DISPOSAL — satu baris per aset
// ============================================================

// disposalAgreementNumbers — nomor agreement aktif (bukan REJECTED) per
// transaksi. Agreement yang ditolak dilewati karena anggotanya bisa
// dikelompokkan ulang ke agreement lain.
func disposalAgreementNumbers(txIDs []uint) map[uint]string {
	result := map[uint]string{}
	if len(txIDs) == 0 {
		return result
	}
	type pair struct {
		TransactionID   uint
		AgreementNumber string
	}
	var pairs []pair
	config.DB.Table("disposal_agreement_items i").
		Select("i.transaction_id, a.agreement_number").
		Joins("JOIN disposal_agreements a ON a.id = i.agreement_id AND a.deleted_at IS NULL").
		Where("i.transaction_id IN ? AND a.current_stage <> ?", txIDs, "REJECTED").
		Order("a.id ASC").
		Scan(&pairs)
	for _, p := range pairs {
		result[p.TransactionID] = p.AgreementNumber
	}
	return result
}

func GetDisposalReport(filter dto.TransactionReportFilter, viewer AssetViewer) ([]dto.DisposalReportRow, dto.TransactionReportSummary, error) {
	summary := dto.TransactionReportSummary{ByStage: []dto.ReportStageCount{}}
	scope, err := resolveReportScope(viewer, filter.BranchCode)
	if err != nil {
		return nil, summary, err
	}

	query := applyOriginBranchScope(baseReportQuery(TxDisposal, filter), scope)
	if strings.TrimSpace(filter.Search) != "" {
		like := searchLike(filter.Search)
		query = query.Where(
			"transactions.transaction_number LIKE ? OR transactions.notes LIKE ? OR transactions.id IN (SELECT da.transaction_id FROM transaction_disposal_assets da JOIN assets a ON a.id = da.asset_id WHERE da.asset_number LIKE ? OR a.asset_name LIKE ?)",
			like, like, like, like)
	}

	transactions, byStage, err := fetchReportTransactions(query, filter.Stage)
	if err != nil {
		return nil, summary, err
	}
	summary.ByStage = byStage
	summary.TotalTransactions = len(transactions)

	ids := transactionIDs(transactions)
	var lines []models.TransactionDisposalAsset
	if len(ids) > 0 {
		if err := config.DB.Where("transaction_id IN ?", ids).
			Order("id ASC").Find(&lines).Error; err != nil {
			return nil, summary, err
		}
	}
	linesByTx := map[uint][]models.TransactionDisposalAsset{}
	assetIDs := make([]uint, 0, len(lines))
	for _, l := range lines {
		linesByTx[l.TransactionID] = append(linesByTx[l.TransactionID], l)
		assetIDs = append(assetIDs, l.AssetID)
	}

	assets := reportAssets(assetIDs)
	agreements := disposalAgreementNumbers(ids)
	branchNames := reportBranchNames()
	categoryNames := reportCategoryNames()
	creators := reportCreators(transactions)

	rows := make([]dto.DisposalReportRow, 0)
	for _, t := range transactions {
		info := reportTransactionInfo(t, branchNames, creators)
		var agreement *string
		if n, ok := agreements[t.ID]; ok {
			agreement = &n
		}
		disposalType := ""
		if t.DisposalType != nil {
			disposalType = *t.DisposalType
		}
		txLines := linesByTx[t.ID]
		if len(txLines) == 0 {
			rows = append(rows, dto.DisposalReportRow{ReportTransactionInfo: info, DisposalType: disposalType, AgreementNumber: agreement})
			continue
		}
		for _, l := range txLines {
			asset := assets[l.AssetID]
			category := ""
			if asset.CategoryID != nil {
				category = categoryNames[*asset.CategoryID]
			}
			var invoiceDate *string
			if l.InvoiceDate != nil {
				d := l.InvoiceDate.Format("2006-01-02")
				invoiceDate = &d
			}
			rows = append(rows, dto.DisposalReportRow{
				ReportTransactionInfo: info,
				DisposalType:          l.DisposalType,
				AgreementNumber:       agreement,
				AssetNumber:           l.AssetNumber,
				AssetName:             asset.AssetName,
				CategoryName:          category,
				DisposalReason:        l.DisposalReason,
				SaleValue:             l.SaleValue,
				IncomeValue:           l.IncomeValue,
				InvoiceNumber:         l.InvoiceNumber,
				InvoiceDate:           invoiceDate,
				DocumentNumber:        l.DocumentNumber,
				AssetStatus:           l.Status,
			})
			if l.Status != "CANCELLED" {
				if l.SaleValue != nil {
					summary.TotalValue += *l.SaleValue
				}
				if l.IncomeValue != nil {
					summary.TotalIncome += *l.IncomeValue
				}
			}
		}
	}
	summary.TotalRows = len(rows)
	return rows, summary, nil
}

func ExportDisposalReport(filter dto.TransactionReportFilter, viewer AssetViewer) (*excelize.File, string, error) {
	rows, summary, err := GetDisposalReport(filter, viewer)
	if err != nil {
		return nil, "", err
	}
	columns := append(append([]reportColumn{}, reportInfoColumns...),
		reportColumn{"Tipe Disposal", 12, false},
		reportColumn{"No. Agreement", 26, false},
		reportColumn{"No. Aset", 18, false},
		reportColumn{"Nama Aset", 28, false},
		reportColumn{"Kategori", 20, false},
		reportColumn{"Alasan", 30, false},
		reportColumn{"Nilai Jual", 16, true},
		reportColumn{"Nilai Pemasukan", 16, true},
		reportColumn{"No. Faktur", 18, false},
		reportColumn{"Tgl. Faktur", 12, false},
		reportColumn{"No. Dokumen", 18, false},
		reportColumn{"Status Aset", 12, false},
		reportColumn{"Catatan Transaksi", 30, false},
	)
	data := make([][]interface{}, 0, len(rows))
	for _, r := range rows {
		data = append(data, append(reportInfoValues(r.ReportTransactionInfo),
			r.DisposalType, strOrEmpty(r.AgreementNumber), r.AssetNumber, r.AssetName, r.CategoryName,
			strOrEmpty(r.DisposalReason), floatOrNil(r.SaleValue), floatOrNil(r.IncomeValue),
			strOrEmpty(r.InvoiceNumber), strOrEmpty(r.InvoiceDate), strOrEmpty(r.DocumentNumber),
			r.AssetStatus, strOrEmpty(r.Notes)))
	}
	f, err := buildReportWorkbook(reportExport{
		title:   "REPORT DISPOSAL",
		columns: columns,
		rows:    data,
		totals: [][2]interface{}{
			{"Jumlah Transaksi", summary.TotalTransactions},
			{"Jumlah Aset", summary.TotalRows},
			{"Total Nilai Jual", summary.TotalValue},
			{"Total Nilai Pemasukan", summary.TotalIncome},
		},
	}, filter, summary)
	if err != nil {
		return nil, "", err
	}
	return f, reportFilename("disposal-report", filter), nil
}
