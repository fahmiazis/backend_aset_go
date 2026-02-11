package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    string    `gorm:"type:char(36);not null;index" json:"user_id"`
	RoleID    string    `gorm:"type:char(36);not null;index" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`

	// Relations
	User User  `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Role *Role `gorm:"foreignKey:RoleID;references:ID" json:"role"`
}

func (ur *UserRole) BeforeCreate(tx *gorm.DB) error {
	if ur.ID == "" {
		ur.ID = uuid.New().String()
	}
	return nil
}
