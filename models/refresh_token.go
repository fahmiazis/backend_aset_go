package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshToken struct {
	ID         string    `gorm:"type:char(36);primaryKey" json:"id"`
	UserID     string    `gorm:"type:char(36);not null;index" json:"user_id"`
	Token      string    `gorm:"type:text;not null" json:"token"`
	DeviceInfo string    `gorm:"type:varchar(255)" json:"device_info"`
	IPAddress  string    `gorm:"type:varchar(45)" json:"ip_address"`
	ExpiresAt  time.Time `gorm:"not null" json:"expires_at"`
	IsRevoked  bool      `gorm:"default:false" json:"is_revoked"`
	CreatedAt  time.Time `json:"created_at"`

	// Relation
	User User `gorm:"foreignKey:UserID;references:ID" json:"-"`
}

func (rt *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if rt.ID == "" {
		rt.ID = uuid.New().String()
	}
	return nil
}
