package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"sort"
	"time"
)

// ============================================================
// DASHBOARD
//
// Dibatasi cabang user dengan aturan yang sama dengan report
// (AssetBranchScope): aset lewat assets.branch_code, transaksi lewat kode
// cabang di nomor transaksinya.
// ============================================================

// dashboardTransactionTypes — jenis yang tampil di dashboard, urutan legend
var dashboardTransactionTypes = []string{TxProcurement, TxMutation, TxDisposal, TxHandover, TxStockOpname}

const dashboardRecentPerType = 10

func monthRange(t time.Time) (time.Time, time.Time) {
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	return start, start.AddDate(0, 1, -1)
}

func GetDashboardSummary(viewer AssetViewer) (*dto.DashboardSummaryResponse, error) {
	now := time.Now()
	scope := reportScope{}
	scope.codes, scope.all = AssetBranchScope(viewer)

	values, err := dashboardAssetValues(now, scope)
	if err != nil {
		return nil, err
	}
	flow, monthCounts, err := dashboardFlow(now, scope)
	if err != nil {
		return nil, err
	}
	recent, err := dashboardRecent(scope)
	if err != nil {
		return nil, err
	}

	return &dto.DashboardSummaryResponse{
		AssetValues: values,
		Flow:        flow,
		MonthCounts: monthCounts,
		Recent:      recent,
	}, nil
}

// dashboardAssetValues — asset_values berisi satu baris per aset per bulan
// (hasil hitung penyusutan bulanan), kadang lebih dari satu dalam sebulan
// (mis. nilai awal saat aset dibuat + hasil penyusutan). Yang dipakai baris
// terakhir tiap aset di bulan berjalan.
//
// Catatan: baris bulan berjalan baru ada setelah penyusutan bulan itu
// dihitung (POST /depreciation/calculate). Sebelum itu kartunya kecil/nol.
func dashboardAssetValues(now time.Time, scope reportScope) (dto.DashboardAssetValues, error) {
	start, end := monthRange(now)
	result := dto.DashboardAssetValues{Period: start.Format("2006-01")}

	if !scope.all && len(scope.codes) == 0 {
		return result, nil
	}

	query := config.DB.Table("asset_values av").
		Select("av.id, av.asset_id, av.effective_date, av.acquisition_value, av.book_value, av.accumulated_depreciation").
		Joins("JOIN assets a ON a.id = av.asset_id").
		Where("av.effective_date BETWEEN ? AND ?", start.Format("2006-01-02"), end.Format("2006-01-02")).
		Where("a.asset_status <> ? AND a.deleted_at IS NULL", "DISPOSED")
	if !scope.all {
		query = query.Where("a.branch_code IN ?", scope.codes)
	}

	var rows []models.AssetValue
	if err := query.Find(&rows).Error; err != nil {
		return result, err
	}

	latest := map[uint]models.AssetValue{}
	for _, r := range rows {
		cur, ok := latest[r.AssetID]
		if !ok || r.EffectiveDate.After(cur.EffectiveDate) ||
			(r.EffectiveDate.Equal(cur.EffectiveDate) && r.ID > cur.ID) {
			latest[r.AssetID] = r
		}
	}
	for _, r := range latest {
		result.AcquisitionValue += r.AcquisitionValue
		result.BookValue += r.BookValue
		result.AccumulatedDepreciation += r.AccumulatedDepreciation
	}
	result.TotalAssets = len(latest)
	return result, nil
}

func dashboardFlow(now time.Time, scope reportScope) ([]dto.DashboardFlowRow, []dto.DashboardTypeCount, error) {
	firstMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -5, 0)
	_, monthEnd := monthRange(now)
	currentMonth := now.Format("2006-01")

	query := applyOriginBranchScope(config.DB.Model(&models.Transaction{}), scope).
		Select("transactions.transaction_type, transactions.transaction_date, transactions.current_stage").
		Where("transactions.transaction_type IN ?", dashboardTransactionTypes).
		Where("transactions.transaction_date BETWEEN ? AND ?", firstMonth.Format("2006-01-02"), monthEnd.Format("2006-01-02"))

	var transactions []models.Transaction
	if err := query.Find(&transactions).Error; err != nil {
		return nil, nil, err
	}

	// semua kombinasi bulan × jenis diisi nol dulu, supaya grafik tidak bolong
	flow := make([]dto.DashboardFlowRow, 0, 6*len(dashboardTransactionTypes))
	index := map[string]int{}
	for i := 0; i < 6; i++ {
		month := firstMonth.AddDate(0, i, 0).Format("2006-01")
		for _, txType := range dashboardTransactionTypes {
			index[month+"|"+txType] = len(flow)
			flow = append(flow, dto.DashboardFlowRow{Month: month, TransactionType: txType})
		}
	}

	monthCount := map[string]int{}
	for _, t := range transactions {
		month := t.TransactionDate.Format("2006-01")
		i, ok := index[month+"|"+t.TransactionType]
		if !ok {
			continue
		}
		switch t.CurrentStage {
		case "FINISHED":
			flow[i].Finished++
		case "REJECTED":
			flow[i].Rejected++
		case "CANCELLED":
			flow[i].Cancelled++
		default:
			flow[i].InProgress++
		}
		if month == currentMonth {
			monthCount[t.TransactionType]++
		}
	}

	counts := make([]dto.DashboardTypeCount, 0, len(dashboardTransactionTypes))
	for _, txType := range dashboardTransactionTypes {
		counts = append(counts, dto.DashboardTypeCount{TransactionType: txType, Count: monthCount[txType]})
	}
	return flow, counts, nil
}

// dashboardRecent — transaksi terbaru per jenis. Diambil per jenis (bukan 10
// teratas gabungan) supaya filter jenis di frontend tetap berisi.
func dashboardRecent(scope reportScope) ([]dto.DashboardRecentTransaction, error) {
	var all []models.Transaction
	for _, txType := range dashboardTransactionTypes {
		var batch []models.Transaction
		if err := applyOriginBranchScope(config.DB.Model(&models.Transaction{}), scope).
			Where("transactions.transaction_type = ?", txType).
			Order("transactions.created_at DESC, transactions.id DESC").
			Limit(dashboardRecentPerType).
			Find(&batch).Error; err != nil {
			return nil, err
		}
		all = append(all, batch...)
	}

	sort.SliceStable(all, func(i, j int) bool {
		if !all[i].CreatedAt.Equal(all[j].CreatedAt) {
			return all[i].CreatedAt.After(all[j].CreatedAt)
		}
		return all[i].ID > all[j].ID
	})

	creators := reportCreators(all)
	result := make([]dto.DashboardRecentTransaction, 0, len(all))
	for _, t := range all {
		result = append(result, dto.DashboardRecentTransaction{
			TransactionNumber: t.TransactionNumber,
			TransactionType:   t.TransactionType,
			TransactionDate:   t.TransactionDate.Format("2006-01-02"),
			CurrentStage:      t.CurrentStage,
			Status:            t.Status,
			CreatedByName:     creators[t.CreatedBy],
		})
	}
	return result, nil
}
