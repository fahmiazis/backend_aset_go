package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FCMToken struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    string    `gorm:"type:char(36);not null;index" json:"user_id"`
	Token     string    `gorm:"type:text;not null" json:"token"`
	DeviceID  string    `gorm:"type:varchar(255)" json:"device_id"`
	Platform  string    `gorm:"type:enum('android','ios','web')" json:"platform"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relation
	User User `gorm:"foreignKey:UserID;references:ID" json:"-"`
}

func (f *FCMToken) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	return nil
}
