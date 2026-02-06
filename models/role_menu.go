package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StringArray custom type untuk handle JSON array of strings di MySQL
type StringArray []string

// Scan implements sql.Scanner — MySQL JSON → Go slice
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to cast value to []byte")
	}

	return json.Unmarshal(bytes, s)
}

// Value implements driver.Valuer — Go slice → MySQL JSON
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

type RoleMenu struct {
	ID          string      `gorm:"type:char(36);primaryKey" json:"id"`
	RoleID      string      `gorm:"type:char(36);not null" json:"role_id"`
	MenuID      string      `gorm:"type:char(36);not null" json:"menu_id"`
	Permissions StringArray `gorm:"type:json;not null;default:('[]')" json:"permissions"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`

	// Relations
	Role Role `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	Menu Menu `gorm:"foreignKey:MenuID" json:"menu,omitempty"`
}

func (rm *RoleMenu) BeforeCreate(tx *gorm.DB) error {
	if rm.ID == "" {
		rm.ID = uuid.New().String()
	}
	return nil
}

func (RoleMenu) TableName() string {
	return "role_menus"
}
