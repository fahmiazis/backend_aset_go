package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserCabang struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    string    `gorm:"type:char(36);not null;index:idx_user_cabang" json:"user_id"`
	CabangID  string    `gorm:"type:char(36);not null;index:idx_user_cabang" json:"cabang_id"`
	CreatedAt time.Time `json:"created_at"`

	// Relations
	User   User   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Cabang Cabang `gorm:"foreignKey:CabangID;constraint:OnDelete:CASCADE" json:"cabang,omitempty"`
}

// BeforeCreate will set a UUID
func (uc *UserCabang) BeforeCreate(tx *gorm.DB) error {
	if uc.ID == "" {
		uc.ID = uuid.New().String()
	}
	return nil
}

// TableName overrides the table name
func (UserCabang) TableName() string {
	return "user_cabangs"
}
