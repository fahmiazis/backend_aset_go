package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ============================================================
// Stock Opname Report
//
// Dibangun di atas data flow baru (services/stock_opname_flow_service.go).
// Beberapa kategori di layout referensi (dari contoh Excel/dashboard tim
// aset) TIDAK punya jalur data di skema saat ini — setiap tempat itu
// terjadi diberi komentar eksplisit supaya jelas mana angka nyata dan mana
// placeholder. Lihat juga doc-STOCK_OPNAME_FLOW_SUMMARY.md di root repo.
// ============================================================

// assetOpnameFact = satu baris gabungan "asset + temuan opname periode ini",
// dipakai bersama oleh dashboard, detail report, dan export excel supaya
// ketiganya selalu konsisten.
type assetOpnameFact struct {
	AssetID                 uint
	AssetNumber             string
	AssetName               string
	CategoryName            string
	Grouping                string
	BranchCode              string
	BranchName              string
	BranchType              string
	CurrentAssetStatus      string
	AcquisitionValue        float64
	AccumulatedDepreciation float64
	BookValue               float64

	HasOpnameThisPeriod bool
	OpnameStage         string
	TransactionNumber   string
	FoundPhysicalStatus string
	FoundCondition      string
	FoundAssetStatus    string
}

const (
	statusBucketFinish      = "finish"
	statusBucketInProgress  = "in_progress"
	statusBucketBelumSubmit = "belum_submit"
	statusBucketRejected    = "rejected"
	statusBucketDisposal    = "disposal"
)

func (f assetOpnameFact) isDisposed() bool {
	return f.CurrentAssetStatus == models.AssetStatusDisposed || f.CurrentAssetStatus == models.AssetStatusInDisposal
}

// statusBucket menentukan bucket Finish/InProgress/BelumSubmit/Rejected/Disposal
// untuk satu asset. "Revisi" sengaja tidak muncul di sini — lihat catatan di
// dto.StockOpnameStatusBreakdown.
func (f assetOpnameFact) statusBucket() string {
	if f.isDisposed() {
		return statusBucketDisposal
	}
	if !f.HasOpnameThisPeriod {
		return statusBucketBelumSubmit
	}
	switch f.OpnameStage {
	case models.StageFinished:
		return statusBucketFinish
	case models.StageRejected:
		return statusBucketRejected
	default: // DRAFT, APPROVAL, EXECUTE_STOCK_OPNAME
		return statusBucketInProgress
	}
}

func addToBreakdown(b *dto.StockOpnameStatusBreakdown, bucket string) {
	switch bucket {
	case statusBucketFinish:
		b.Finish++
	case statusBucketInProgress:
		b.InProgress++
	case statusBucketBelumSubmit:
		b.BelumSubmit++
	case statusBucketRejected:
		b.Rejected++
	case statusBucketDisposal:
		b.Disposal++
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// groupingLabel returns the trimmed grouping value, or "" when unset — left
// for the caller (FE) to render/translate as it sees fit, since this is a
// display concern, not backend business data.
func groupingLabel(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

// resolveReportPeriod: default ke bulan/tahun berjalan kalau tidak dikirim.
func resolveReportPeriod(month, year int) (time.Time, time.Time) {
	now := time.Now()
	if month < 1 || month > 12 {
		month = int(now.Month())
	}
	if year == 0 {
		year = now.Year()
	}
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)
	return start, end
}

// gatherStockOpnameFacts memuat semua asset dalam scope (difilter branch bila
// diminta) beserta nilai aktif (asset_values) dan temuan opname periode
// berjalan (kalau ada). Satu query per sumber data (assets, branches,
// asset_values, transaction_stock_opnames) — dihindari N+1 per baris.
//
// Selain facts (per-asset), juga mengembalikan scopedBranches: daftar branch
// yang harus tetap muncul di report (rekap per-branch, cost center, sheet
// "tidak kirim") walaupun branch itu belum/tidak punya asset sama sekali —
// sebelumnya branch seperti ini hilang total dari report karena semua
// agregasi per-branch dibangun dari facts (yang sumbernya tabel assets).
func gatherStockOpnameFacts(month, year int, branchCode string) (facts []assetOpnameFact, scopedBranches []models.Branch, start time.Time, end time.Time, err error) {
	start, end = resolveReportPeriod(month, year)

	assetQuery := config.DB.Model(&models.Asset{}).Preload("Category")
	branchCode = strings.TrimSpace(branchCode)
	filterSingleBranch := branchCode != "" && !strings.EqualFold(branchCode, "ALL")
	if filterSingleBranch {
		assetQuery = assetQuery.Where("branch_code = ?", branchCode)
	}

	var assets []models.Asset
	if err := assetQuery.Find(&assets).Error; err != nil {
		return nil, nil, start, end, fmt.Errorf("failed to load assets: %w", err)
	}

	var branches []models.Branch
	if err := config.DB.Find(&branches).Error; err != nil {
		return nil, nil, start, end, fmt.Errorf("failed to load branches: %w", err)
	}
	branchByCode := make(map[string]models.Branch, len(branches))
	for _, b := range branches {
		branchByCode[b.BranchCode] = b
	}

	if filterSingleBranch {
		for _, b := range branches {
			if strings.EqualFold(b.BranchCode, branchCode) {
				scopedBranches = []models.Branch{b}
				break
			}
		}
	} else {
		scopedBranches = branches
	}

	assetIDs := make([]uint, len(assets))
	for i, a := range assets {
		assetIDs[i] = a.ID
	}

	valueByAsset := make(map[uint]models.AssetValue, len(assets))
	if len(assetIDs) > 0 {
		var values []models.AssetValue
		if err := config.DB.Where("asset_id IN ? AND is_active = ?", assetIDs, true).Find(&values).Error; err != nil {
			return nil, nil, start, end, fmt.Errorf("failed to load asset values: %w", err)
		}
		for _, v := range values {
			valueByAsset[v.AssetID] = v
		}
	}

	type itemRow struct {
		AssetID           uint
		PhysicalStatus    string
		Condition         string
		AssetStatus       string
		TransactionID     uint
		TransactionNumber string
		CurrentStage      string
		CreatedAt         time.Time
	}
	// Item stock opname sekarang tersebar di 3 tabel tergantung stage
	// (draft/aktif/history — lihat services/stock_opname_flow_service.go),
	// jadi digabung lewat UNION ALL biar laporan tetap lihat semua item di
	// periode ini gak peduli lagi ada di tabel mana.
	itemTables := []string{"stock_opname_draft_items", "transaction_stock_opnames", "stock_opname_item_history"}
	unionParts := make([]string, len(itemTables))
	args := make([]interface{}, 0, len(itemTables)*3)
	for i, table := range itemTables {
		unionParts[i] = fmt.Sprintf(`
			SELECT tso.asset_id AS asset_id, tso.physical_status AS physical_status,
			       tso.`+"`condition`"+` AS `+"`condition`"+`, tso.asset_status AS asset_status,
			       tso.transaction_id AS transaction_id, tso.transaction_number AS transaction_number,
			       t.current_stage AS current_stage, t.created_at AS created_at
			FROM %s tso
			JOIN transactions t ON t.id = tso.transaction_id
			WHERE t.transaction_type = ? AND t.transaction_date BETWEEN ? AND ?
		`, table)
		args = append(args, TxStockOpnameFlow, start.Format("2006-01-02"), end.Format("2006-01-02"))
	}
	unionQuery := strings.Join(unionParts, " UNION ALL ") + " ORDER BY created_at ASC"

	var items []itemRow
	if err := config.DB.Raw(unionQuery, args...).Scan(&items).Error; err != nil {
		return nil, nil, start, end, fmt.Errorf("failed to load stock opname items: %w", err)
	}

	// Map keyed by asset_id; karena di-order ASC, penulisan terakhir yang
	// menang = temuan paling baru untuk asset itu di periode ini.
	itemByAsset := make(map[uint]itemRow, len(items))
	for _, it := range items {
		itemByAsset[it.AssetID] = it
	}

	facts = make([]assetOpnameFact, 0, len(assets))
	for _, a := range assets {
		v := valueByAsset[a.ID]
		branch := branchByCode[derefStr(a.BranchCode)]

		f := assetOpnameFact{
			AssetID:                 a.ID,
			AssetNumber:             a.AssetNumber,
			AssetName:               a.AssetName,
			Grouping:                groupingLabel(a.Grouping),
			BranchCode:              derefStr(a.BranchCode),
			BranchName:              branch.BranchName,
			BranchType:              branch.BranchType,
			CurrentAssetStatus:      a.AssetStatus,
			AcquisitionValue:        v.AcquisitionValue,
			AccumulatedDepreciation: v.AccumulatedDepreciation,
			BookValue:               v.BookValue,
		}
		if a.Category != nil {
			f.CategoryName = a.Category.CategoryName
		}
		if it, ok := itemByAsset[a.ID]; ok {
			f.HasOpnameThisPeriod = true
			f.OpnameStage = it.CurrentStage
			f.TransactionNumber = it.TransactionNumber
			f.FoundPhysicalStatus = it.PhysicalStatus
			f.FoundCondition = it.Condition
			f.FoundAssetStatus = it.AssetStatus
		}
		facts = append(facts, f)
	}

	return facts, scopedBranches, start, end, nil
}

// isAreaBranch: "AREA" di layout referensi = semua branch selain HO
// (mencakup branch_type "SUB BRANCH" dan variasi lain yang ada di data).
func isAreaBranch(branchType string) bool {
	return !strings.EqualFold(strings.TrimSpace(branchType), "HO")
}

// ============================================================
// DASHBOARD
// ============================================================

func GetStockOpnameReportDashboard(filter dto.StockOpnameReportFilter) (*dto.StockOpnameDashboardResponse, error) {
	facts, _, start, end, err := gatherStockOpnameFacts(filter.Month, filter.Year, filter.BranchCode)
	if err != nil {
		return nil, err
	}

	var overall dto.StockOpnameStatusBreakdown
	var acquisitionTotal, bookTotal float64

	groupingMap := map[string]*dto.StockOpnameGroupingStatus{}
	var groupingOrder []string

	var physical dto.StockOpnamePhysicalVsSystem
	var condition dto.StockOpnameConditionSummary

	for _, f := range facts {
		bucket := f.statusBucket()
		addToBreakdown(&overall, bucket)
		acquisitionTotal += f.AcquisitionValue
		bookTotal += f.BookValue

		g, ok := groupingMap[f.Grouping]
		if !ok {
			g = &dto.StockOpnameGroupingStatus{Grouping: f.Grouping}
			groupingMap[f.Grouping] = g
			groupingOrder = append(groupingOrder, f.Grouping)
		}
		addToBreakdown(&g.Status, bucket)
		g.Total++

		if f.isDisposed() {
			continue
		}
		if !f.HasOpnameThisPeriod || f.FoundCondition == "" {
			condition.BelumIsi++
			continue
		}
		if f.FoundPhysicalStatus == models.PhysicalStatusMissing {
			physical.PhysicalTidakAda++
			condition.TidakAda++
			continue
		}
		physical.PhysicalAda++
		switch f.FoundCondition {
		case models.ConditionGood, models.ConditionFair:
			condition.Baik++
		case models.ConditionPoor, models.ConditionBroken:
			condition.Rusak++
		}
	}
	physical.SystemAda = physical.PhysicalAda + physical.PhysicalTidakAda
	physical.SystemTidakAda = 0 // lihat catatan di dto.StockOpnamePhysicalVsSystem

	sort.Strings(groupingOrder)
	groupings := make([]dto.StockOpnameGroupingStatus, 0, len(groupingOrder))
	for _, key := range groupingOrder {
		g := groupingMap[key]
		groupings = append(groupings, *g)
	}

	totalAsset := int64(len(facts))
	finishPct := 0.0
	if totalAsset > 0 {
		finishPct = math.Round((float64(overall.Finish)/float64(totalAsset))*10000) / 100
	}

	resolvedBranch := strings.TrimSpace(filter.BranchCode)
	if resolvedBranch == "" {
		resolvedBranch = "ALL"
	}

	return &dto.StockOpnameDashboardResponse{
		Period: dto.StockOpnameReportPeriod{
			Month:     int(start.Month()),
			Year:      start.Year(),
			StartDate: start.Format("2006-01-02"),
			EndDate:   end.Format("2006-01-02"),
		},
		BranchCode: resolvedBranch,
		Stats: dto.StockOpnameDashboardStats{
			TotalAsset:       totalAsset,
			Finish:           overall.Finish,
			FinishPercentage: finishPct,
			InProgress:       overall.InProgress,
			BelumSubmit:      overall.BelumSubmit,
			Rejected:         overall.Rejected,
			Revisi:           overall.Revisi,
			Disposal:         overall.Disposal,
			AcquisitionValue: math.Round(acquisitionTotal*100) / 100,
			BookValue:        math.Round(bookTotal*100) / 100,
		},
		Charts: dto.StockOpnameDashboardCharts{
			StatusPerGrouping: groupings,
			PhysicalVsSystem:  physical,
			ConditionSummary:  condition,
			StatusSubmit:      overall,
		},
	}, nil
}

// ============================================================
// DETAIL REPORT (rekap table + cost center)
// ============================================================

func GetStockOpnameReportDetail(filter dto.StockOpnameReportFilter) (*dto.StockOpnameDetailReportResponse, error) {
	facts, scopedBranches, start, end, err := gatherStockOpnameFacts(filter.Month, filter.Year, filter.BranchCode)
	if err != nil {
		return nil, err
	}

	rekap, areaSummary := buildRekapRows(facts, scopedBranches)
	costCenters := buildCostCenterRows(facts, scopedBranches, 10)

	resolvedBranch := strings.TrimSpace(filter.BranchCode)
	if resolvedBranch == "" {
		resolvedBranch = "ALL"
	}

	return &dto.StockOpnameDetailReportResponse{
		Period: dto.StockOpnameReportPeriod{
			Month:     int(start.Month()),
			Year:      start.Year(),
			StartDate: start.Format("2006-01-02"),
			EndDate:   end.Format("2006-01-02"),
		},
		BranchCode:  resolvedBranch,
		Rekap:       rekap,
		AreaSummary: areaSummary,
		CostCenters: costCenters,
		Note: "Baris bertanda supported=false belum bisa dihitung dari skema data saat ini " +
			"(tidak ada jalur input aset fisik yang belum terdaftar di sistem, dan tidak ada flag kendaraan di kategori aset). " +
			"Lihat doc-STOCK_OPNAME_FLOW_SUMMARY.md untuk detail gap-nya.",
	}, nil
}

func buildRekapRows(facts []assetOpnameFact, scopedBranches []models.Branch) ([]dto.StockOpnameRekapRow, dto.StockOpnameAreaSummary) {
	var sapFisikArea, sapFisikHO, sapAdaFisikTidak, assetMutasi dto.StockOpnameRekapRow
	sapFisikArea.Label = "SAP = FISIK AREA"
	sapFisikArea.Supported = true
	sapFisikHO.Label = "SAP = FISIK HO"
	sapFisikHO.Supported = true
	sapAdaFisikTidak.Label = "SAP ADA FISIK TIDAK"
	sapAdaFisikTidak.Supported = true
	assetMutasi.Label = "ASSET MUTASI"
	assetMutasi.Supported = true

	var idleCount, rusakCount, hilangCount int64
	var assetBalance dto.StockOpnameRekapRow
	assetBalance.Label = fmt.Sprintf("Asset Balance SAP per %s", time.Now().Format("2 January 2006"))
	assetBalance.Supported = true

	// per-branch activity, untuk baris "Total Area yang tidak kirim" +
	// Area Clear/Tidak Clear.
	type branchAgg struct {
		branchType   string
		branchName   string
		totalAsset   int64
		opnamedAsset int64
		fullyClearOK bool // true kalau SEMUA asset di branch ini FINISH dan tidak ada MISSING
		acquisition  float64
		bookValue    float64
		accumDep     float64
	}
	branchAggs := map[string]*branchAgg{}
	// Seed dengan SEMUA branch dalam scope dulu (walau belum/tidak punya
	// asset sama sekali), supaya branch itu tetap muncul di "Total Area yang
	// tidak kirim" dan hitungan Area Clear/Tidak Clear dengan nilai 0 —
	// sebelumnya branch tanpa asset hilang total karena map ini cuma dibangun
	// dari facts (asset yang benar-benar ada).
	for _, b := range scopedBranches {
		branchAggs[b.BranchCode] = &branchAgg{branchType: b.BranchType, branchName: b.BranchName, fullyClearOK: true}
	}

	for _, f := range facts {
		assetBalance.AcquisitionValue += f.AcquisitionValue
		assetBalance.AccumulatedDepreciation += f.AccumulatedDepreciation
		assetBalance.BookValue += f.BookValue
		assetBalance.UnitCount++

		if f.CurrentAssetStatus == models.AssetStatusInMutation {
			assetMutasi.AcquisitionValue += f.AcquisitionValue
			assetMutasi.AccumulatedDepreciation += f.AccumulatedDepreciation
			assetMutasi.BookValue += f.BookValue
			assetMutasi.UnitCount++
		}

		if strings.Contains(strings.ToUpper(f.Grouping), "IDLE") {
			idleCount++
		}

		ba, ok := branchAggs[f.BranchCode]
		if !ok {
			ba = &branchAgg{branchType: f.BranchType, branchName: f.BranchName, fullyClearOK: true}
			branchAggs[f.BranchCode] = ba
		}
		ba.totalAsset++
		ba.acquisition += f.AcquisitionValue
		ba.bookValue += f.BookValue
		ba.accumDep += f.AccumulatedDepreciation

		if f.HasOpnameThisPeriod {
			ba.opnamedAsset++
		}

		if f.HasOpnameThisPeriod && f.FoundCondition != "" {
			switch f.FoundCondition {
			case models.ConditionPoor, models.ConditionBroken:
				rusakCount++
			}
			if f.FoundPhysicalStatus == models.PhysicalStatusMissing {
				hilangCount++
				sapAdaFisikTidak.AcquisitionValue += f.AcquisitionValue
				sapAdaFisikTidak.AccumulatedDepreciation += f.AccumulatedDepreciation
				sapAdaFisikTidak.BookValue += f.BookValue
				sapAdaFisikTidak.UnitCount++
			} else if f.OpnameStage == models.StageFinished {
				if isAreaBranch(f.BranchType) {
					sapFisikArea.AcquisitionValue += f.AcquisitionValue
					sapFisikArea.AccumulatedDepreciation += f.AccumulatedDepreciation
					sapFisikArea.BookValue += f.BookValue
					sapFisikArea.UnitCount++
				} else {
					sapFisikHO.AcquisitionValue += f.AcquisitionValue
					sapFisikHO.AccumulatedDepreciation += f.AccumulatedDepreciation
					sapFisikHO.BookValue += f.BookValue
					sapFisikHO.UnitCount++
				}
			}
		}

		// Area dianggap "clear" hanya kalau semua asset FINISH & fisik ketemu.
		if !f.HasOpnameThisPeriod || f.OpnameStage != models.StageFinished || f.FoundPhysicalStatus == models.PhysicalStatusMissing {
			ba.fullyClearOK = false
		}
	}

	totalRekonsiliasi := dto.StockOpnameRekapRow{
		Label:                   "Total Rekonsiliasi Hasil Opname AREA NASIONAL & HO TGR NON IT",
		AcquisitionValue:        sapFisikArea.AcquisitionValue + sapFisikHO.AcquisitionValue + sapAdaFisikTidak.AcquisitionValue + assetMutasi.AcquisitionValue,
		AccumulatedDepreciation: sapFisikArea.AccumulatedDepreciation + sapFisikHO.AccumulatedDepreciation + sapAdaFisikTidak.AccumulatedDepreciation + assetMutasi.AccumulatedDepreciation,
		BookValue:               sapFisikArea.BookValue + sapFisikHO.BookValue + sapAdaFisikTidak.BookValue + assetMutasi.BookValue,
		UnitCount:               sapFisikArea.UnitCount + sapFisikHO.UnitCount + sapAdaFisikTidak.UnitCount + assetMutasi.UnitCount,
		Supported:               true,
	}

	// "Total Area yang tidak kirim": branch (HO maupun Area) yang belum ada
	// satupun asset ter-opname periode ini.
	var tidakKirim dto.StockOpnameRekapRow
	tidakKirim.Label = "Total Area yang tidak kirim"
	tidakKirim.Supported = true
	var areaNotSubmitted, hoNotSubmitted int64

	var areaClear, areaNotClear, totalArea int64
	branchCodes := make([]string, 0, len(branchAggs))
	for code := range branchAggs {
		branchCodes = append(branchCodes, code)
	}
	sort.Strings(branchCodes)
	for _, code := range branchCodes {
		ba := branchAggs[code]
		if ba.opnamedAsset == 0 {
			tidakKirim.AcquisitionValue += ba.acquisition
			tidakKirim.AccumulatedDepreciation += ba.accumDep
			tidakKirim.BookValue += ba.bookValue
			tidakKirim.UnitCount += ba.totalAsset
			if isAreaBranch(ba.branchType) {
				areaNotSubmitted++
			} else {
				hoNotSubmitted++
			}
		}

		if isAreaBranch(ba.branchType) {
			totalArea++
			if ba.opnamedAsset > 0 && ba.fullyClearOK {
				areaClear++
			} else {
				areaNotClear++
			}
		}
	}
	tidakKirim.ExtraInfo = fmt.Sprintf("%d area; %d ho", areaNotSubmitted, hoNotSubmitted)

	balanceDelta := assetBalance.BookValue - (totalRekonsiliasi.BookValue + tidakKirim.BookValue)
	balanceChecked := dto.StockOpnameRekapRow{
		Label:            "Balance Checked",
		AcquisitionValue: assetBalance.AcquisitionValue - (totalRekonsiliasi.AcquisitionValue + tidakKirim.AcquisitionValue),
		BookValue:        balanceDelta,
		Supported:        true,
		ExtraInfo:        "-",
	}
	if math.Abs(balanceDelta) < 0.01 {
		balanceChecked.ExtraInfo = "OK"
	} else {
		balanceChecked.ExtraInfo = "SELISIH (ada asset belum ter-opname di branch yang sudah kirim sebagian)"
	}

	rows := []dto.StockOpnameRekapRow{
		sapFisikArea,
		sapFisikHO,
		sapAdaFisikTidak,
		{Label: "SAP ADA FISIK TIDAK - KENDARAAN", Supported: false},
		assetMutasi,
		totalRekonsiliasi,
		tidakKirim,
		assetBalance,
		balanceChecked,
		{Label: "IDLE", UnitCount: idleCount, Supported: true},
		{Label: "RUSAK", UnitCount: rusakCount, Supported: true},
		{Label: "FISIK ADA SAP TIDAK ADA - Asset Belum di GR", Supported: false},
		{Label: "FISIK ADA SAP TIDAK ADA - Asset Belum Terdaftar", Supported: false},
		{Label: "FISIK ADA SAP TIDAK ADA - Tidak ada nomor Asset", Supported: false},
		{Label: "HILANG", UnitCount: hilangCount, Supported: true},
	}

	areaClearPct, areaNotClearPct := 0.0, 0.0
	if totalArea > 0 {
		areaClearPct = math.Round((float64(areaClear)/float64(totalArea))*10000) / 100
		areaNotClearPct = math.Round((float64(areaNotClear)/float64(totalArea))*10000) / 100
	}

	return rows, dto.StockOpnameAreaSummary{
		AreaClearCount:         areaClear,
		AreaClearPercentage:    areaClearPct,
		AreaNotClearCount:      areaNotClear,
		AreaNotClearPercentage: areaNotClearPct,
		TotalArea:              totalArea,
	}
}

func buildCostCenterRows(facts []assetOpnameFact, scopedBranches []models.Branch, topN int) []dto.StockOpnameCostCenterRow {
	type agg struct {
		branchCode, branchName      string
		acquisitionValue, bookValue float64
		unitCount                   int64
	}
	byBranch := map[string]*agg{}
	// Seed dulu dengan semua branch dalam scope supaya branch tanpa asset
	// (nilai 0) tetap kebawa, bukan cuma branch yang punya asset di facts.
	for _, b := range scopedBranches {
		byBranch[b.BranchCode] = &agg{branchCode: b.BranchCode, branchName: b.BranchName}
	}
	for _, f := range facts {
		if f.BranchCode == "" {
			continue
		}
		a, ok := byBranch[f.BranchCode]
		if !ok {
			a = &agg{branchCode: f.BranchCode, branchName: f.BranchName}
			byBranch[f.BranchCode] = a
		}
		a.acquisitionValue += f.AcquisitionValue
		a.bookValue += f.BookValue
		a.unitCount++
	}

	rows := make([]dto.StockOpnameCostCenterRow, 0, len(byBranch))
	for _, a := range byBranch {
		rows = append(rows, dto.StockOpnameCostCenterRow{
			BranchCode:       a.branchCode,
			BranchName:       a.branchName,
			AcquisitionValue: math.Round(a.acquisitionValue*100) / 100,
			BookValue:        math.Round(a.bookValue*100) / 100,
			UnitCount:        a.unitCount,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].AcquisitionValue > rows[j].AcquisitionValue })
	if len(rows) > topN {
		rows = rows[:topN]
	}
	return rows
}

// ============================================================
// EXPORT EXCEL
// ============================================================

func ExportStockOpnameReportExcel(filter dto.StockOpnameExportRequest) (*excelize.File, string, error) {
	facts, scopedBranches, start, end, err := gatherStockOpnameFacts(filter.Month, filter.Year, filter.BranchCode)
	if err != nil {
		return nil, "", err
	}

	rekap, areaSummary := buildRekapRows(facts, scopedBranches)
	costCenters := buildCostCenterRows(facts, scopedBranches, 10)

	f := excelize.NewFile()
	defer func() {
		// excelize.NewFile() sudah bikin sheet "Sheet1" default; kita drop
		// setelah semua sheet lain siap.
	}()

	monthName := start.Format("January")

	writeSummarySheet(f, rekap, areaSummary, monthName, start.Year())
	writeDetailListSheet(f, "SAP=FISIK", facts, func(x assetOpnameFact) bool {
		return x.HasOpnameThisPeriod && x.OpnameStage == models.StageFinished && x.FoundPhysicalStatus != models.PhysicalStatusMissing
	})
	writeDetailListSheet(f, "SAP ADA FISIK TDK", facts, func(x assetOpnameFact) bool {
		return x.HasOpnameThisPeriod && x.FoundPhysicalStatus == models.PhysicalStatusMissing
	})
	writeDetailListSheet(f, "MUTASI", facts, func(x assetOpnameFact) bool {
		return x.CurrentAssetStatus == models.AssetStatusInMutation
	})
	writeNotSubmittedBranchSheet(f, "TDK KIRIM", facts, scopedBranches)
	writeUnsupportedSheet(f, "SAP TDK ADA FISIK ADA",
		"Belum didukung skema saat ini: tidak ada jalur untuk mencatat aset yang ditemukan "+
			"secara fisik tapi belum terdaftar di sistem (endpoint add-asset stock opname mensyaratkan asset_id yang sudah ada).")
	writeCostCenterSheet(f, "COST CENTER", costCenters)
	writeDetailListSheet(f, "DETAIL", facts, func(x assetOpnameFact) bool { return true })

	f.DeleteSheet("Sheet1")
	if _, err := f.NewSheet("SUMMARY"); err == nil {
		// no-op, dijaga supaya SUMMARY selalu ada walau writeSummarySheet gagal bikin sheet baru
	}
	f.SetActiveSheet(0)

	filename := fmt.Sprintf("stock_opname_report_%s_%d.xlsx", strings.ToLower(monthName), start.Year())
	_ = end
	return f, filename, nil
}

func writeSummarySheet(f *excelize.File, rekap []dto.StockOpnameRekapRow, area dto.StockOpnameAreaSummary, monthName string, year int) {
	sheet := "SUMMARY"
	index, _ := f.NewSheet(sheet)

	boldStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	titleStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	moneyStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 3}) // #,##0

	f.SetCellValue(sheet, "A1", fmt.Sprintf("OPNAME ASSET %s %d", strings.ToUpper(monthName), year))
	f.SetCellStyle(sheet, "A1", "A1", titleStyle)
	f.MergeCell(sheet, "A1", "E1")

	headers := []string{"REKAPITULASI", "Acquis.val.", "Accum.dep.", "Book val.", "Unit / Info"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, boldStyle)
	}

	row := 4
	for _, r := range rekap {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.Label)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.AcquisitionValue)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.AccumulatedDepreciation)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r.BookValue)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("D%d", row), moneyStyle)

		info := r.ExtraInfo
		if info == "" && r.UnitCount > 0 {
			info = fmt.Sprintf("%d unit", r.UnitCount)
		}
		if !r.Supported {
			info = "belum didukung skema saat ini"
		}
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), info)
		row++
	}

	row++
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Area Clear")
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("%d (%.0f%%)", area.AreaClearCount, area.AreaClearPercentage))
	row++
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Area Tidak Clear")
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("%d (%.0f%%)", area.AreaNotClearCount, area.AreaNotClearPercentage))
	row++
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Total Area Nasional")
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("%d (100%%)", area.TotalArea))

	f.SetColWidth(sheet, "A", "A", 55)
	f.SetColWidth(sheet, "B", "D", 18)
	f.SetColWidth(sheet, "E", "E", 30)
	f.SetActiveSheet(index)
}

func writeDetailListSheet(f *excelize.File, sheet string, facts []assetOpnameFact, include func(assetOpnameFact) bool) {
	index, _ := f.NewSheet(sheet)
	boldStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})

	headers := []string{
		"Asset Number", "Asset Name", "Category", "Branch Code", "Branch Name",
		"Grouping", "Current Asset Status", "Opname Stage", "Transaction Number",
		"Found Physical Status", "Found Condition", "Found Asset Status",
		"Acquisition Value", "Book Value",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, boldStyle)
	}

	row := 2
	for _, fact := range facts {
		if !include(fact) {
			continue
		}
		values := []interface{}{
			fact.AssetNumber, fact.AssetName, fact.CategoryName, fact.BranchCode, fact.BranchName,
			fact.Grouping, fact.CurrentAssetStatus, fact.OpnameStage, fact.TransactionNumber,
			fact.FoundPhysicalStatus, fact.FoundCondition, fact.FoundAssetStatus,
			fact.AcquisitionValue, fact.BookValue,
		}
		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellValue(sheet, cell, v)
		}
		row++
	}
	f.SetColWidth(sheet, "A", "N", 18)
	f.SetActiveSheet(index)
}

func writeNotSubmittedBranchSheet(f *excelize.File, sheet string, facts []assetOpnameFact, scopedBranches []models.Branch) {
	index, _ := f.NewSheet(sheet)
	boldStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})

	type agg struct {
		branchCode, branchName, branchType string
		totalAsset, opnamedAsset           int64
	}
	byBranch := map[string]*agg{}
	// Seed dulu dengan semua branch dalam scope supaya branch yang belum
	// (atau tidak punya) asset sama sekali tetap masuk sheet ini dengan
	// Total Asset = 0, bukan hilang total.
	for _, b := range scopedBranches {
		byBranch[b.BranchCode] = &agg{branchCode: b.BranchCode, branchName: b.BranchName, branchType: b.BranchType}
	}
	for _, fact := range facts {
		if fact.BranchCode == "" {
			continue
		}
		a, ok := byBranch[fact.BranchCode]
		if !ok {
			a = &agg{branchCode: fact.BranchCode, branchName: fact.BranchName, branchType: fact.BranchType}
			byBranch[fact.BranchCode] = a
		}
		a.totalAsset++
		if fact.HasOpnameThisPeriod {
			a.opnamedAsset++
		}
	}

	headers := []string{"Branch Code", "Branch Name", "Branch Type", "Total Asset"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, boldStyle)
	}

	codes := make([]string, 0, len(byBranch))
	for code := range byBranch {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	row := 2
	for _, code := range codes {
		a := byBranch[code]
		if a.opnamedAsset > 0 {
			continue
		}
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), a.branchCode)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), a.branchName)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), a.branchType)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), a.totalAsset)
		row++
	}
	f.SetColWidth(sheet, "A", "D", 20)
	f.SetActiveSheet(index)
}

func writeUnsupportedSheet(f *excelize.File, sheet string, note string) {
	index, _ := f.NewSheet(sheet)
	f.SetCellValue(sheet, "A1", note)
	f.SetColWidth(sheet, "A", "A", 100)
	f.SetActiveSheet(index)
}

func writeCostCenterSheet(f *excelize.File, sheet string, rows []dto.StockOpnameCostCenterRow) {
	index, _ := f.NewSheet(sheet)
	boldStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})

	headers := []string{"Branch Code", "Branch Name", "Acquisition Value", "Book Value", "Unit Count"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, boldStyle)
	}

	row := 2
	for _, r := range rows {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.BranchCode)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.BranchName)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.AcquisitionValue)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r.BookValue)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), r.UnitCount)
		row++
	}
	f.SetColWidth(sheet, "A", "E", 22)
	f.SetActiveSheet(index)
}
