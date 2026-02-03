package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Cabang struct {
	ID           string         `gorm:"type:char(36);primaryKey" json:"id"`
	KodeCabang   string         `gorm:"type:varchar(10);uniqueIndex;not null" json:"kode_cabang"`
	NamaCabang   string         `gorm:"type:varchar(255);not null" json:"nama_cabang"`
	StatusCabang string         `gorm:"type:enum('active','inactive');default:'active'" json:"status_cabang"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

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
		var lastCabang Cabang
		err := tx.Unscoped().Order("kode_cabang DESC").First(&lastCabang).Error

		if err != nil {
			// First cabang, start with C00001
			c.KodeCabang = "C00001"
		} else {
			// Extract number from last kode_cabang and increment
			var lastNumber int
			_, err := fmt.Sscanf(lastCabang.KodeCabang, "C%05d", &lastNumber)
			if err != nil {
				c.KodeCabang = "C00001"
			} else {
				c.KodeCabang = fmt.Sprintf("C%05d", lastNumber+1)
			}
		}
	}

	return nil
}

// TableName overrides the table name
func (Cabang) TableName() string {
	return "cabangs"
}
