package models

import "time"

// StockOpnameConfig adalah single-row settings buat modul stock opname.
// Kolom baru ditambahin di sini seiring butuh config lain (mis. hak akses
// per role, nyusul — sengaja belum digarap).
type StockOpnameConfig struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	SubmissionStartDay int       `json:"submission_start_day"`
	SubmissionEndDay   int       `json:"submission_end_day"`
	UpdatedBy          *string   `gorm:"size:100" json:"updated_by"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (StockOpnameConfig) TableName() string { return "stock_opname_configs" }
