package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Cabang struct {
	ID          string         `gorm:"type:char(36);primaryKey" json:"id"`
	KodeCabang  string         `gorm:"type:varchar(10);uniqueIndex;not null" json:"kode_cabang"`
	NamaCabang  string         `gorm:"type:varchar(255);not null" json:"nama_cabang"`
	JenisCabang string         `gorm:"type:varchar(50);not null" json:"jenis_cabang"`
	Status      string         `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	UserCabangs []UserCabang `gorm:"foreignKey:CabangID" json:"user_cabangs,omitempty"`
}

// BeforeCreate will set a UUID rather than numeric ID
func (c *Cabang) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}

	// Auto-generate kode_cabang if not provided
	if c.KodeCabang == "" {
		var kodeCabang string

		// Single atomic query: ambil max number dan increment dalam satu go
		// COALESCE handles the case when table is empty (returns 0)
		tx.Raw(`SELECT CONCAT('C', LPAD(COALESCE(MAX(CAST(SUBSTRING(kode_cabang, 2) AS UNSIGNED)), 0) + 1, 5, '0')) FROM cabangs`).Scan(&kodeCabang)

		c.KodeCabang = kodeCabang
	}

	return nil
}

// TableName overrides the table name
func (Cabang) TableName() string {
	return "cabangs"
}
