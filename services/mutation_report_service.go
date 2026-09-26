package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ============================================================
// MUTATION — satu baris per aset.
// Terlihat oleh cabang pengaju MAUPUN cabang tujuan.
// ============================================================

func GetMutationReport(filter dto.TransactionReportFilter, viewer AssetViewer) ([]dto.MutationReportRow, dto.TransactionReportSummary, error) {
	summary := dto.TransactionReportSummary{ByStage: []dto.ReportStageCount{}}
	scope, err := resolveReportScope(viewer, filter.BranchCode)
	if err != nil {
		return nil, summary, err
	}

	query := baseReportQuery(TxMutation, filter)
	if !scope.all {
		if len(scope.codes) == 0 {
			query = query.Where("1 = 0")
		} else {
			query = query.Where("("+reportBranchExpr+" IN ? OR transactions.mutation_to_branch_code IN ?)", scope.codes, scope.codes)
		}
	}
	if strings.TrimSpace(filter.Search) != "" {
		like := searchLike(filter.Search)
		query = query.Where(
			"transactions.transaction_number LIKE ? OR transactions.notes LIKE ? OR transactions.id IN (SELECT ma.transaction_id FROM transaction_mutation_assets ma JOIN assets a ON a.id = ma.asset_id WHERE ma.asset_number LIKE ? OR a.asset_name LIKE ?)",
			like, like, like, like)
	}

	transactions, byStage, err := fetchReportTransactions(query, filter.Stage)
	if err != nil {
		return nil, summary, err
	}
	summary.ByStage = byStage
	summary.TotalTransactions = len(transactions)

	var lines []models.TransactionMutationAsset
	if len(transactions) > 0 {
		if err := config.DB.Where("transaction_id IN ?", transactionIDs(transactions)).
			Order("id ASC").Find(&lines).Error; err != nil {
			return nil, summary, err
		}
	}
	linesByTx := map[uint][]models.TransactionMutationAsset{}
	assetIDs := make([]uint, 0, len(lines))
	for _, l := range lines {
		linesByTx[l.TransactionID] = append(linesByTx[l.TransactionID], l)
		assetIDs = append(assetIDs, l.AssetID)
	}

	assets := reportAssets(assetIDs)
	branchNames := reportBranchNames()
	categoryNames := reportCategoryNames()
	creators := reportCreators(transactions)

	rows := make([]dto.MutationReportRow, 0)
	for _, t := range transactions {
		info := reportTransactionInfo(t, branchNames, creators)
		txLines := linesByTx[t.ID]
		if len(txLines) == 0 {
			row := dto.MutationReportRow{ReportTransactionInfo: info, FromBranchCode: info.BranchCode, FromBranchName: info.BranchName}
			if t.MutationToBranchCode != nil {
				row.ToBranchCode = *t.MutationToBranchCode
				row.ToBranchName = branchNames[*t.MutationToBranchCode]
			}
			rows = append(rows, row)
			continue
		}
		for _, l := range txLines {
			asset := assets[l.AssetID]
			category := ""
			if asset.CategoryID != nil {
				category = categoryNames[*asset.CategoryID]
			}
			rows = append(rows, dto.MutationReportRow{
				ReportTransactionInfo: info,
				AssetNumber:           l.AssetNumber,
				AssetName:             asset.AssetName,
				CategoryName:          category,
				FromBranchCode:        l.FromBranchCode,
				FromBranchName:        branchNames[l.FromBranchCode],
				ToBranchCode:          l.ToBranchCode,
				ToBranchName:          branchNames[l.ToBranchCode],
				FromLocation:          l.FromLocation,
				ToLocation:            l.ToLocation,
				DocumentNumber:        l.DocumentNumber,
				AssetStatus:           l.Status,
			})
		}
	}
	summary.TotalRows = len(rows)
	return rows, summary, nil
}

func ExportMutationReport(filter dto.TransactionReportFilter, viewer AssetViewer) (*excelize.File, string, error) {
	rows, summary, err := GetMutationReport(filter, viewer)
	if err != nil {
		return nil, "", err
	}
	columns := append(append([]reportColumn{}, reportInfoColumns...),
		reportColumn{"No. Aset", 18, false},
		reportColumn{"Nama Aset", 28, false},
		reportColumn{"Kategori", 20, false},
		reportColumn{"Dari Cabang", 24, false},
		reportColumn{"Ke Cabang", 24, false},
		reportColumn{"Lokasi Asal", 20, false},
		reportColumn{"Lokasi Tujuan", 20, false},
		reportColumn{"No. Dokumen", 18, false},
		reportColumn{"Status Aset", 12, false},
		reportColumn{"Catatan Transaksi", 30, false},
	)
	data := make([][]interface{}, 0, len(rows))
	executed := 0
	for _, r := range rows {
		if r.AssetStatus == "EXECUTED" {
			executed++
		}
		data = append(data, append(reportInfoValues(r.ReportTransactionInfo),
			r.AssetNumber, r.AssetName, r.CategoryName,
			branchLabelWithName(r.FromBranchCode, r.FromBranchName),
			branchLabelWithName(r.ToBranchCode, r.ToBranchName),
			strOrEmpty(r.FromLocation), strOrEmpty(r.ToLocation),
			strOrEmpty(r.DocumentNumber), r.AssetStatus, strOrEmpty(r.Notes)))
	}
	f, err := buildReportWorkbook(reportExport{
		title:   "REPORT MUTATION",
		columns: columns,
		rows:    data,
		totals: [][2]interface{}{
			{"Jumlah Transaksi", summary.TotalTransactions},
			{"Jumlah Aset", summary.TotalRows},
			{"Aset Sudah Dieksekusi", executed},
		},
	}, filter, summary)
	if err != nil {
		return nil, "", err
	}
	return f, reportFilename("mutation-report", filter), nil
}
