package models

import "time"

// TransactionIONumber menyimpan IO number per branch per transaksi
// 1 transaksi bisa punya N IO numbers sesuai jumlah branch yang terlibat
type TransactionIONumber struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	TransactionID     uint      `gorm:"not null;index" json:"transaction_id"`
	TransactionNumber string    `gorm:"size:100;not null;index" json:"transaction_number"`
	BranchCode        string    `gorm:"size:50;not null;index" json:"branch_code"`
	IONumber          string    `gorm:"size:50;not null;uniqueIndex" json:"io_number"`
	ProcessedBy       string    `gorm:"size:100;not null" json:"processed_by"` // UUID PIC Budget
	ProcessedAt       time.Time `gorm:"not null" json:"processed_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// Relations
	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
}

func (TransactionIONumber) TableName() string { return "transaction_io_numbers" }
