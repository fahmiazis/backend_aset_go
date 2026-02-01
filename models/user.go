package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        string         `gorm:"type:char(36);primaryKey" json:"id"`
	Username  string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	Fullname  string         `gorm:"type:varchar(100);not null" json:"fullname"`
	Email     string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"type:varchar(255);not null" json:"-"`
	NIK       *string        `gorm:"type:varchar(20);uniqueIndex" json:"nik"` // ← Jadi pointer
	MPNNumber *string        `gorm:"type:varchar(50)" json:"mpn_number"`      // ← Jadi pointer
	Status    string         `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID" json:"-"`
	UserRoles     []UserRole     `gorm:"foreignKey:UserID" json:"-"`
	FCMTokens     []FCMToken     `gorm:"foreignKey:UserID" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}
