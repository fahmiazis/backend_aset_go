package scheduler

import (
	"backend-go/dto"
	"backend-go/services"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

var schedulerInstance *cron.Cron

// StartScheduler menjalankan semua scheduled jobs
// Dipanggil di main.go setelah setup routes
func StartScheduler() {
	schedulerInstance = cron.New(cron.WithSeconds())

	// Jalankan kalkulasi depresiasi tiap tanggal 1, jam 00:01:00
	// Format: second minute hour day month weekday
	schedulerInstance.AddFunc("0 1 0 1 * *", runMonthlyDepreciation)

	schedulerInstance.Start()
	fmt.Println("[Scheduler] Started - Monthly depreciation will run on the 1st of each month at 00:01")
}

// StopScheduler menghentikan scheduler — dipanggil saat shutdown
func StopScheduler() {
	if schedulerInstance != nil {
		schedulerInstance.Stop()
		fmt.Println("[Scheduler] Stopped")
	}
}

// runMonthlyDepreciation menjalankan kalkulasi depresiasi untuk bulan sebelumnya
// Dijalankan tanggal 1 → hitung bulan sebelumnya
// Contoh: dijalankan 1 April → hitung periode 2026-03
func runMonthlyDepreciation() {
	// Hitung bulan sebelumnya
	now := time.Now()
	lastMonth := now.AddDate(0, -1, 0)
	period := lastMonth.Format("2006-01")

	fmt.Printf("[Scheduler] Running monthly depreciation for period: %s\n", period)

	err := services.CalculateMonthlyDepreciation("system", dto.CalculateDepreciationRequest{
		Period: period,
	})

	if err != nil {
		fmt.Printf("[Scheduler] ERROR: Monthly depreciation failed for %s: %v\n", period, err)
		return
	}

	fmt.Printf("[Scheduler] Monthly depreciation completed for period: %s\n", period)

	// Auto lock setelah calculate berhasil
	if err := services.LockMonthlyDepreciation(period); err != nil {
		fmt.Printf("[Scheduler] WARNING: Failed to lock depreciation for %s: %v\n", period, err)
		return
	}

	fmt.Printf("[Scheduler] Monthly depreciation locked for period: %s\n", period)
}
