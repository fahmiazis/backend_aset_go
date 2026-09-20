package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"time"
)

// ============================================================
// CONFIG STOCK OPNAME
//
// Single-row settings (id=1 selalu). Dipakai buat nentuin apakah sebuah
// submit "on schedule" (lihat isWithinSubmissionWindow) — TIDAK PERNAH
// dipakai buat memblokir submit itu sendiri, semua stock opname tetap bisa
// disubmit kapan pun (lihat SubmitStockOpname).
//
// Belum ada pengaturan hak akses siapa yang boleh ubah config ini (nyusul)
// — untuk sekarang cukup harus login (AuthMiddleware di routes), gak ada
// pengecekan permission tambahan.
// ============================================================

func getOrCreateStockOpnameConfig() (*models.StockOpnameConfig, error) {
	var cfg models.StockOpnameConfig
	if err := config.DB.First(&cfg).Error; err == nil {
		return &cfg, nil
	}
	// Belum ada baris sama sekali (mis. DB baru tanpa data seed) -> bikin
	// default 25 s/d 8 biar app tetap jalan tanpa perlu migrasi ulang.
	cfg = models.StockOpnameConfig{SubmissionStartDay: 25, SubmissionEndDay: 8}
	if err := config.DB.Create(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func GetStockOpnameConfig() (*dto.StockOpnameConfigResponse, error) {
	cfg, err := getOrCreateStockOpnameConfig()
	if err != nil {
		return nil, err
	}
	return &dto.StockOpnameConfigResponse{
		SubmissionStartDay: cfg.SubmissionStartDay,
		SubmissionEndDay:   cfg.SubmissionEndDay,
		UpdatedBy:          cfg.UpdatedBy,
		UpdatedAt:          cfg.UpdatedAt,
	}, nil
}

func UpdateStockOpnameConfig(userID string, req dto.UpdateStockOpnameConfigRequest) (*dto.StockOpnameConfigResponse, error) {
	cfg, err := getOrCreateStockOpnameConfig()
	if err != nil {
		return nil, err
	}
	if err := config.DB.Model(cfg).Updates(map[string]interface{}{
		"submission_start_day": req.SubmissionStartDay,
		"submission_end_day":   req.SubmissionEndDay,
		"updated_by":           userID,
	}).Error; err != nil {
		return nil, err
	}
	return GetStockOpnameConfig()
}

// isWithinSubmissionWindow ngecek apakah tanggal `t` jatuh di jendela
// submit [startDay, endDay] (berdasarkan tanggal-di-bulan aja, bukan bulan
// spesifik). Kalau startDay > endDay berarti jendelanya wrap ke bulan
// berikutnya (mis. 25 s/d 8: tanggal 25-31/30/29/28 ATAU 1-8 dianggap valid).
func isWithinSubmissionWindow(t time.Time, startDay, endDay int) bool {
	day := t.Day()
	if startDay <= endDay {
		return day >= startDay && day <= endDay
	}
	return day >= startDay || day <= endDay
}
