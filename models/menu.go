package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Menu struct {
	ID         string         `gorm:"type:char(36);primaryKey" json:"id"`
	ParentID   *string        `gorm:"type:char(36)" json:"parent_id"`
	Name       string         `gorm:"type:varchar(100);not null" json:"name"`
	Path       *string        `gorm:"type:varchar(255)" json:"path"`       // frontend route
	RoutePath  *string        `gorm:"type:varchar(255)" json:"route_path"` // backend API route
	IconName   *string        `gorm:"type:varchar(100)" json:"icon_name"`
	OrderIndex int            `gorm:"type:int;default:0" json:"order_index"`
	Status     string         `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Parent    *Menu      `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children  []Menu     `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	RoleMenus []RoleMenu `gorm:"foreignKey:MenuID" json:"role_menus,omitempty"`
}

func (m *Menu) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

func (Menu) TableName() string {
	return "menus"
}
