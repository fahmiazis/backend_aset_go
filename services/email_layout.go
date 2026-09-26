package services

import (
	"backend-go/config"
	"backend-go/models"
	"fmt"
	"html"
	"strings"
	"time"
)

// ============================================================
// Layout email notifikasi
//
// Template di Email Setting hanya berisi PESAN. Selebihnya — header, kartu
// info transaksi, daftar aset, tombol — dibangun di sini dari data transaksi,
// supaya semua email seragam dan daftar asetnya selalu sesuai isi ajuan.
//
// HTML-nya sengaja gaya email lama (table + inline style): Gmail/Outlook
// membuang <style>, flex, dan grid.
// ============================================================

// batas baris tabel aset — ajuan besar tetap terbaca, sisanya dirujuk ke
// halaman detail
const emailMaxAssetRows = 50

type emailSummary struct {
	typeLabel string
	// pasangan label → nilai di kartu info
	info  [][2]string
	notes string

	assetTitle string
	columns    []string
	// indeks kolom yang rata kanan (angka/nominal)
	rightAligned map[int]bool
	assetRows    [][]string
	// baris total opsional, panjangnya sama dengan columns
	totalRow []string
}

type emailLayout struct {
	action     string
	subject    string
	message    string
	additional string
	senderName string
	link       string
	summary    emailSummary
}

// ── format ──────────────────────────────────────────────────

func formatRupiahEmail(value float64) string {
	negative := value < 0
	if negative {
		value = -value
	}
	whole := fmt.Sprintf("%.0f", value)
	var b strings.Builder
	for i, ch := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(ch)
	}
	if negative {
		return "-Rp " + b.String()
	}
	return "Rp " + b.String()
}

func formatDateEmail(t time.Time) string {
	months := []string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
	return fmt.Sprintf("%02d %s %d", t.Day(), months[t.Month()-1], t.Year())
}

func strOr(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return *value
}

func branchLabel(code string) string {
	if code == "" || code == "ALL" {
		return "-"
	}
	var branch models.Branch
	if err := config.DB.Where("branch_code = ?", code).First(&branch).Error; err == nil && branch.BranchName != "" {
		return fmt.Sprintf("%s — %s", code, branch.BranchName)
	}
	return code
}

func transactionTypeLabel(txType string) string {
	switch txType {
	case TxProcurement:
		return "Procurement"
	case TxMutationFlow:
		return "Mutasi Aset"
	case TxDisposalFlow:
		return "Disposal Aset"
	case TxDisposalAgreement:
		return "Disposal Agreement"
	}
	return txType
}

func stageFlowLabel(stage, next string) string {
	if next == "" || next == stage {
		return stage
	}
	return stage + " → " + next
}

// ── ringkasan per jenis transaksi ───────────────────────────

func baseInfo(transaction *models.Transaction, ctx emailRenderContext) [][2]string {
	creator := strOr(resolveUserFullname(transaction.CreatedBy), "-")
	sender := strOr(resolveUserFullname(ctx.senderID), "-")

	return [][2]string{
		{"Nomor", transaction.TransactionNumber},
		{"Jenis", transactionTypeLabel(transaction.TransactionType)},
		{"Tahap", stageFlowLabel(ctx.stage, ctx.nextStage)},
		{"Cabang", branchLabel(GetCreatorBranchCode(transaction.CreatedBy))},
		{"Tanggal Ajuan", formatDateEmail(transaction.TransactionDate)},
		{"Diajukan oleh", creator},
		{"Diproses oleh", sender},
	}
}

func buildTransactionSummary(transaction *models.Transaction, ctx emailRenderContext) emailSummary {
	summary := emailSummary{
		typeLabel:    transactionTypeLabel(transaction.TransactionType),
		info:         baseInfo(transaction, ctx),
		notes:        strOr(transaction.Notes, ""),
		rightAligned: map[int]bool{},
	}

	switch transaction.TransactionType {
	case TxProcurement:
		if transaction.IONumber != nil && *transaction.IONumber != "" {
			summary.info = append(summary.info, [2]string{"Nomor IO", *transaction.IONumber})
		}

		var items []models.TransactionProcurement
		config.DB.Where("transaction_id = ?", transaction.ID).Order("id").Find(&items)

		summary.assetTitle = fmt.Sprintf("Barang yang Diajukan (%d)", len(items))
		summary.columns = []string{"No", "Nama Barang", "Qty", "Harga Satuan", "Total"}
		summary.rightAligned = map[int]bool{2: true, 3: true, 4: true}

		var qty int
		var total float64
		for i, item := range items {
			qty += item.Quantity
			total += item.TotalPrice
			summary.assetRows = append(summary.assetRows, []string{
				fmt.Sprint(i + 1),
				item.ItemName,
				fmt.Sprint(item.Quantity),
				formatRupiahEmail(item.UnitPrice),
				formatRupiahEmail(item.TotalPrice),
			})
		}
		if len(items) > 0 {
			summary.totalRow = []string{"", "Total", fmt.Sprint(qty), "", formatRupiahEmail(total)}
		}

	case TxMutationFlow:
		if transaction.MutationToBranchCode != nil {
			summary.info = append(summary.info, [2]string{"Cabang Tujuan", branchLabel(*transaction.MutationToBranchCode)})
		}

		var rows []models.TransactionMutationAsset
		config.DB.Preload("Asset").
			Where("transaction_id = ? AND status <> ?", transaction.ID, "CANCELLED").
			Order("id").Find(&rows)

		summary.assetTitle = fmt.Sprintf("Aset yang Dimutasi (%d)", len(rows))
		summary.columns = []string{"No", "Nomor Aset", "Nama Aset", "Dari", "Ke"}
		for i, row := range rows {
			name := "-"
			if row.Asset != nil {
				name = row.Asset.AssetName
			}
			to := row.ToBranchCode
			if row.ToLocation != nil && *row.ToLocation != "" {
				to += " · " + *row.ToLocation
			}
			summary.assetRows = append(summary.assetRows, []string{
				fmt.Sprint(i + 1), row.AssetNumber, name, row.FromBranchCode, to,
			})
		}

	case TxDisposalFlow:
		disposalType := strOr(transaction.DisposalType, models.DisposalTypeDispose)
		summary.info = append(summary.info, [2]string{"Tipe Disposal", disposalType})

		var rows []models.TransactionDisposalAsset
		config.DB.Preload("Asset").
			Where("transaction_id = ? AND status <> ?", transaction.ID, "CANCELLED").
			Order("id").Find(&rows)

		summary.assetTitle = fmt.Sprintf("Aset yang Didisposal (%d)", len(rows))
		sell := disposalType == models.DisposalTypeSell
		summary.columns = []string{"No", "Nomor Aset", "Nama Aset", "Alasan"}
		if sell {
			summary.columns = append(summary.columns, "Nilai Jual")
			summary.rightAligned = map[int]bool{4: true}
		}

		var total float64
		for i, row := range rows {
			name := "-"
			if row.Asset != nil {
				name = row.Asset.AssetName
			}
			line := []string{fmt.Sprint(i + 1), row.AssetNumber, name, strOr(row.DisposalReason, "-")}
			if sell {
				value := "-"
				if row.SaleValue != nil {
					value = formatRupiahEmail(*row.SaleValue)
					total += *row.SaleValue
				}
				line = append(line, value)
			}
			summary.assetRows = append(summary.assetRows, line)
		}
		if sell && len(rows) > 0 {
			summary.totalRow = []string{"", "", "", "Total", formatRupiahEmail(total)}
		}
	}

	return summary
}

// buildAgreementSummary — agreement menampilkan aset dari transaksi anggota
// (`numbers`). Untuk revisi yang dikirim adalah anggota yang dikeluarkan.
func buildAgreementSummary(scope *agreementEmailScope, numbers []string, ctx emailRenderContext, revise bool) emailSummary {
	creator := strOr(resolveUserFullname(scope.creatorID), "-")
	sender := strOr(resolveUserFullname(ctx.senderID), "-")

	branches := make([]string, 0, len(scope.branchCodes))
	for _, code := range scope.branchCodes {
		branches = append(branches, branchLabel(code))
	}

	summary := emailSummary{
		typeLabel: transactionTypeLabel(TxDisposalAgreement),
		info: [][2]string{
			{"Nomor Agreement", ctx.transaction.TransactionNumber},
			{"Jenis", transactionTypeLabel(TxDisposalAgreement)},
			{"Tahap", stageFlowLabel(ctx.stage, ctx.nextStage)},
			{"Cabang Anggota", strings.Join(branches, ", ")},
			{"Jumlah Transaksi", fmt.Sprint(len(numbers))},
			{"Dibuat oleh", creator},
			{"Diproses oleh", sender},
		},
		columns:      []string{"No", "Nomor Transaksi", "Nomor Aset", "Nama Aset", "Tipe", "Nilai Jual"},
		rightAligned: map[int]bool{5: true},
	}
	if revise {
		summary.info[4] = [2]string{"Transaksi Dikembalikan", fmt.Sprint(len(numbers))}
	}

	var transactions []models.Transaction
	if len(numbers) > 0 {
		config.DB.Where("transaction_number IN ?", numbers).Order("transaction_number").Find(&transactions)
	}

	var total float64
	var hasSale bool
	for _, trx := range transactions {
		var rows []models.TransactionDisposalAsset
		config.DB.Preload("Asset").
			Where("transaction_id = ? AND status <> ?", trx.ID, "CANCELLED").
			Order("id").Find(&rows)

		for _, row := range rows {
			name := "-"
			if row.Asset != nil {
				name = row.Asset.AssetName
			}
			value := "-"
			if row.SaleValue != nil {
				value = formatRupiahEmail(*row.SaleValue)
				total += *row.SaleValue
				hasSale = true
			}
			summary.assetRows = append(summary.assetRows, []string{
				fmt.Sprint(len(summary.assetRows) + 1),
				trx.TransactionNumber, row.AssetNumber, name, row.DisposalType, value,
			})
		}
	}

	if revise {
		summary.assetTitle = fmt.Sprintf("Aset pada Transaksi yang Dikembalikan (%d)", len(summary.assetRows))
	} else {
		summary.assetTitle = fmt.Sprintf("Aset dalam Agreement (%d)", len(summary.assetRows))
	}
	if hasSale {
		summary.totalRow = []string{"", "", "", "", "Total", formatRupiahEmail(total)}
	}
	return summary
}

// ── HTML ────────────────────────────────────────────────────

type actionStyle struct {
	label  string
	color  string // warna utama
	soft   string // latar lembut
	button string
}

func emailActionStyle(action string) actionStyle {
	switch action {
	case models.EmailActionReject:
		return actionStyle{"Ditolak", "#dc2626", "#fef2f2", "Lihat Detail"}
	case models.EmailActionRevise:
		return actionStyle{"Perlu Revisi", "#d97706", "#fffbeb", "Perbaiki Sekarang"}
	case models.EmailActionCancel:
		return actionStyle{"Dibatalkan", "#6b7280", "#f3f4f6", "Lihat Detail"}
	}
	return actionStyle{"Menunggu Tindakan", "#4f46e5", "#eef2ff", "Buka Transaksi"}
}

func esc(s string) string { return html.EscapeString(s) }

func textToHTML(text string) string {
	escaped := html.EscapeString(strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n")))
	return strings.ReplaceAll(escaped, "\n", "<br>")
}

func buildEmailHTML(l emailLayout) string {
	style := emailActionStyle(l.action)
	s := l.summary
	var b strings.Builder
	w := func(format string, args ...interface{}) { fmt.Fprintf(&b, format, args...) }

	w(`<!DOCTYPE html><html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title></head>`, esc(l.subject))
	w(`<body style="margin:0;padding:0;background:#f3f4f6;font-family:Arial,Helvetica,sans-serif;color:#1f2937">`)
	w(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f3f4f6;padding:24px 12px"><tr><td align="center">`)
	w(`<table role="presentation" width="640" cellpadding="0" cellspacing="0" style="width:100%%;max-width:640px;background:#ffffff;border-radius:14px;overflow:hidden;border:1px solid #e5e7eb">`)

	// header
	w(`<tr><td style="background:%s;padding:22px 28px">`, style.color)
	w(`<div style="font-size:12px;letter-spacing:1px;text-transform:uppercase;color:rgba(255,255,255,0.8)">Asset Management System</div>`)
	w(`<div style="margin-top:6px;font-size:20px;font-weight:bold;color:#ffffff">%s</div>`, esc(s.typeLabel))
	w(`<div style="margin-top:10px"><span style="display:inline-block;padding:4px 12px;border-radius:999px;background:#ffffff;color:%s;font-size:12px;font-weight:bold">%s</span></div>`, style.color, esc(style.label))
	w(`</td></tr>`)

	// pesan dari template
	w(`<tr><td style="padding:26px 28px 6px 28px;font-size:14px;line-height:1.65">%s</td></tr>`, textToHTML(l.message))

	// tambahan dari pengirim
	if strings.TrimSpace(l.additional) != "" {
		w(`<tr><td style="padding:12px 28px 4px 28px">`)
		w(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:%s;border-left:4px solid %s;border-radius:6px"><tr><td style="padding:12px 16px;font-size:13px;line-height:1.6">`, style.soft, style.color)
		w(`<div style="font-size:11px;font-weight:bold;color:%s;text-transform:uppercase;letter-spacing:0.5px;margin-bottom:4px">Pesan dari %s</div>`, style.color, esc(l.senderName))
		w(`%s</td></tr></table></td></tr>`, textToHTML(l.additional))
	}

	// kartu info
	if len(s.info) > 0 {
		w(`<tr><td style="padding:20px 28px 4px 28px">`)
		w(`<div style="font-size:13px;font-weight:bold;color:#111827;margin-bottom:8px">Informasi Ajuan</div>`)
		w(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="border:1px solid #e5e7eb;border-radius:8px;font-size:13px">`)
		for i, row := range s.info {
			bg := "#ffffff"
			if i%2 == 0 {
				bg = "#f9fafb"
			}
			w(`<tr style="background:%s"><td style="padding:8px 12px;color:#6b7280;width:38%%;vertical-align:top">%s</td><td style="padding:8px 12px;color:#111827;font-weight:bold;word-break:break-word">%s</td></tr>`, bg, esc(row[0]), esc(row[1]))
		}
		if s.notes != "" {
			w(`<tr><td style="padding:8px 12px;color:#6b7280;vertical-align:top">Catatan</td><td style="padding:8px 12px;color:#111827">%s</td></tr>`, textToHTML(s.notes))
		}
		w(`</table></td></tr>`)
	}

	// tabel aset
	if len(s.columns) > 0 {
		w(`<tr><td style="padding:20px 28px 4px 28px">`)
		w(`<div style="font-size:13px;font-weight:bold;color:#111827;margin-bottom:8px">%s</div>`, esc(s.assetTitle))
		if len(s.assetRows) == 0 {
			w(`<div style="padding:14px;border:1px dashed #d1d5db;border-radius:8px;font-size:13px;color:#9ca3af;text-align:center">Tidak ada data aset</div>`)
		} else {
			align := func(i int) string {
				if s.rightAligned[i] {
					return "right"
				}
				return "left"
			}
			w(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="border:1px solid #e5e7eb;border-radius:8px;font-size:12px;border-collapse:separate">`)
			w(`<tr style="background:#f3f4f6">`)
			for i, col := range s.columns {
				w(`<th style="padding:8px 10px;text-align:%s;color:#374151;font-weight:bold;border-bottom:1px solid #e5e7eb;white-space:nowrap">%s</th>`, align(i), esc(col))
			}
			w(`</tr>`)

			rows := s.assetRows
			if len(rows) > emailMaxAssetRows {
				rows = rows[:emailMaxAssetRows]
			}
			for r, row := range rows {
				bg := "#ffffff"
				if r%2 == 1 {
					bg = "#f9fafb"
				}
				w(`<tr style="background:%s">`, bg)
				for i, cell := range row {
					w(`<td style="padding:8px 10px;text-align:%s;color:#1f2937;border-bottom:1px solid #f3f4f6;vertical-align:top;word-break:break-word">%s</td>`, align(i), esc(cell))
				}
				w(`</tr>`)
			}
			if extra := len(s.assetRows) - len(rows); extra > 0 {
				w(`<tr><td colspan="%d" style="padding:8px 10px;color:#6b7280;font-style:italic">+ %d baris lainnya — lihat detail di aplikasi</td></tr>`, len(s.columns), extra)
			}
			if len(s.totalRow) == len(s.columns) {
				w(`<tr style="background:#f3f4f6">`)
				for i, cell := range s.totalRow {
					w(`<td style="padding:9px 10px;text-align:%s;color:#111827;font-weight:bold">%s</td>`, align(i), esc(cell))
				}
				w(`</tr>`)
			}
			w(`</table>`)
		}
		w(`</td></tr>`)
	}

	// tombol
	if l.link != "" {
		w(`<tr><td align="center" style="padding:26px 28px 8px 28px">`)
		w(`<a href="%s" style="display:inline-block;padding:12px 28px;background:%s;color:#ffffff;text-decoration:none;border-radius:8px;font-size:14px;font-weight:bold">%s</a>`, esc(l.link), style.color, esc(style.button))
		w(`</td></tr>`)
	}

	// footer
	w(`<tr><td style="padding:22px 28px;border-top:1px solid #f3f4f6;font-size:11px;line-height:1.6;color:#9ca3af;text-align:center">`)
	w(`Email ini dikirim otomatis oleh Asset Management System pada %s.<br>Mohon tidak membalas email ini.`, esc(time.Now().Format("02/01/2006 15:04")))
	w(`</td></tr>`)

	w(`</table></td></tr></table></body></html>`)
	return b.String()
}
