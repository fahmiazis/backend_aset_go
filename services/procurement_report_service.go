package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ============================================================
// PROCUREMENT — satu baris per barang
// ============================================================

func GetProcurementReport(filter dto.TransactionReportFilter, viewer AssetViewer) ([]dto.ProcurementReportRow, dto.TransactionReportSummary, error) {
	summary := dto.TransactionReportSummary{ByStage: []dto.ReportStageCount{}}
	scope, err := resolveReportScope(viewer, filter.BranchCode)
	if err != nil {
		return nil, summary, err
	}

	query := applyOriginBranchScope(baseReportQuery(TxProcurement, filter), scope)
	if strings.TrimSpace(filter.Search) != "" {
		like := searchLike(filter.Search)
		query = query.Where(
			"transactions.transaction_number LIKE ? OR transactions.notes LIKE ? OR transactions.id IN (SELECT transaction_id FROM transaction_procurements WHERE item_name LIKE ?)",
			like, like, like)
	}

	transactions, byStage, err := fetchReportTransactions(query, filter.Stage)
	if err != nil {
		return nil, summary, err
	}
	summary.ByStage = byStage
	summary.TotalTransactions = len(transactions)

	itemsByTx := map[uint][]models.TransactionProcurement{}
	if len(transactions) > 0 {
		var items []models.TransactionProcurement
		if err := config.DB.Preload("TransactionProcurementDetails").
			Where("transaction_id IN ?", transactionIDs(transactions)).
			Order("id ASC").Find(&items).Error; err != nil {
			return nil, summary, err
		}
		for _, it := range items {
			itemsByTx[it.TransactionID] = append(itemsByTx[it.TransactionID], it)
		}
	}

	branchNames := reportBranchNames()
	categoryNames := reportCategoryNames()
	creators := reportCreators(transactions)

	rows := make([]dto.ProcurementReportRow, 0)
	for _, t := range transactions {
		info := reportTransactionInfo(t, branchNames, creators)
		items := itemsByTx[t.ID]
		if len(items) == 0 {
			// transaksi tanpa barang tetap tampil supaya jumlahnya cocok dengan daftar
			rows = append(rows, dto.ProcurementReportRow{ReportTransactionInfo: info})
			continue
		}
		for _, it := range items {
			category := ""
			if it.CategoryID != nil {
				category = categoryNames[*it.CategoryID]
			}
			dist := make([]string, 0, len(it.TransactionProcurementDetails))
			for _, d := range it.TransactionProcurementDetails {
				dist = append(dist, fmt.Sprintf("%s (%d)", d.BranchCode, d.Quantity))
			}
			rows = append(rows, dto.ProcurementReportRow{
				ReportTransactionInfo: info,
				ItemName:              it.ItemName,
				CategoryName:          category,
				Quantity:              it.Quantity,
				UnitPrice:             it.UnitPrice,
				TotalPrice:            it.TotalPrice,
				Distribution:          strings.Join(dist, ", "),
				ItemNotes:             it.Notes,
			})
			summary.TotalQuantity += it.Quantity
			summary.TotalValue += it.TotalPrice
		}
	}
	summary.TotalRows = len(rows)
	return rows, summary, nil
}

func ExportProcurementReport(filter dto.TransactionReportFilter, viewer AssetViewer) (*excelize.File, string, error) {
	rows, summary, err := GetProcurementReport(filter, viewer)
	if err != nil {
		return nil, "", err
	}
	columns := append(append([]reportColumn{}, reportInfoColumns...),
		reportColumn{"Nama Barang", 28, false},
		reportColumn{"Kategori", 20, false},
		reportColumn{"Qty", 8, false},
		reportColumn{"Harga Satuan", 16, true},
		reportColumn{"Total Harga", 18, true},
		reportColumn{"Distribusi Cabang", 30, false},
		reportColumn{"Catatan Barang", 30, false},
		reportColumn{"Catatan Transaksi", 30, false},
	)
	data := make([][]interface{}, 0, len(rows))
	for _, r := range rows {
		data = append(data, append(reportInfoValues(r.ReportTransactionInfo),
			r.ItemName, r.CategoryName, r.Quantity, r.UnitPrice, r.TotalPrice,
			r.Distribution, strOrEmpty(r.ItemNotes), strOrEmpty(r.Notes)))
	}
	f, err := buildReportWorkbook(reportExport{
		title:   "REPORT PROCUREMENT",
		columns: columns,
		rows:    data,
		totals: [][2]interface{}{
			{"Jumlah Transaksi", summary.TotalTransactions},
			{"Jumlah Barang", summary.TotalRows},
			{"Total Qty", summary.TotalQuantity},
			{"Total Nilai", summary.TotalValue},
		},
	}, filter, summary)
	if err != nil {
		return nil, "", err
	}
	return f, reportFilename("procurement-report", filter), nil
}
