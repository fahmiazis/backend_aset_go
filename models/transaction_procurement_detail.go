package models

import (
	"time"
)

type TransactionProcurementDetail struct {
	ID                       uint      `gorm:"primaryKey" json:"id"`
	TransactionProcurementID uint      `gorm:"not null;index" json:"transaction_procurement_id"`
	BranchCode               string    `gorm:"size:50;not null;index" json:"branch_code"`
	Quantity                 int       `gorm:"not null" json:"quantity"`
	RequesterName            *string   `gorm:"size:255" json:"requester_name"` // NULLABLE
	Notes                    *string   `gorm:"type:text" json:"notes"`         // NULLABLE
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`

	// Relations
	TransactionProcurement *TransactionProcurement `gorm:"foreignKey:TransactionProcurementID" json:"transaction_procurement,omitempty"`
}

func (TransactionProcurementDetail) TableName() string {
	return "transaction_procurement_details"
}
