package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================================
// Permission & MenuPermission
//
// Hak akses TIDAK generik — tiap permission hanya berlaku pada
// route_path tertentu. middleware.RequirePermission mencari menu
// berdasarkan route_path (path request yang sudah dinormalkan:
// prefix /api/v1 dibuang, segment ID dilewati, maksimal 3 segment),
// lalu mencocokkan permission yang tersimpan di role_menus.permissions.
//
// Daftar permission dan pemetaannya ke menu dikelola lewat tabel
// (lihat migrations/20260501000001 & 20260501000002), bukan di kode,
// supaya bisa ditambah tanpa deploy ulang.
// ============================================================

// Permission — master daftar hak akses
type Permission struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	Value       string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"value"`
	Label       string    `gorm:"type:varchar(100);not null" json:"label"`
	Description *string   `gorm:"type:text" json:"description"`
	Module      *string   `gorm:"type:varchar(50);index" json:"module"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

func (Permission) TableName() string { return "permissions" }

// MenuPermission — permission mana saja yang relevan untuk sebuah menu.
// Ini hanya daftar PILIHAN yang valid; siapa yang punya akses tetap
// ditentukan oleh role_menus.permissions.
type MenuPermission struct {
	ID           string    `gorm:"type:char(36);primaryKey" json:"id"`
	MenuID       string    `gorm:"type:char(36);not null;index" json:"menu_id"`
	PermissionID string    `gorm:"type:char(36);not null;index" json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relations
	Menu       *Menu       `gorm:"foreignKey:MenuID" json:"menu,omitempty"`
	Permission *Permission `gorm:"foreignKey:PermissionID" json:"permission,omitempty"`
}

func (mp *MenuPermission) BeforeCreate(tx *gorm.DB) error {
	if mp.ID == "" {
		mp.ID = uuid.New().String()
	}
	return nil
}

func (MenuPermission) TableName() string { return "menu_permissions" }

// Module bawaan — dipakai untuk mengelompokkan di UI.
// Nilai lain tetap diterima, ini hanya urutan tampil yang diketahui.
const (
	PermissionModuleBasic       = "basic"
	PermissionModuleTransaction = "transaction"
	PermissionModuleProcurement = "procurement"
	PermissionModuleMutation    = "mutation"
	PermissionModuleDisposal    = "disposal"
	PermissionModuleApproval    = "approval"
	PermissionModuleAttachment  = "attachment"
)

// PermissionModuleLabels — label tampilan per module
var PermissionModuleLabels = map[string]string{
	PermissionModuleBasic:       "Akses Dasar",
	PermissionModuleTransaction: "Transaksi",
	PermissionModuleProcurement: "Procurement",
	PermissionModuleMutation:    "Mutasi",
	PermissionModuleDisposal:    "Disposal",
	PermissionModuleApproval:    "Approval",
	PermissionModuleAttachment:  "Dokumen",
}

// PermissionModuleOrder — urutan tampil module di UI
var PermissionModuleOrder = []string{
	PermissionModuleBasic,
	PermissionModuleTransaction,
	PermissionModuleProcurement,
	PermissionModuleMutation,
	PermissionModuleDisposal,
	PermissionModuleApproval,
	PermissionModuleAttachment,
}

// IsValidPermissionFormat — validasi BENTUK, bukan daftar putih.
// Selama masih development, permission bebas boleh ditambah; yang ditolak
// hanya string kosong, terlalu panjang, atau mengandung spasi — karena
// akan gagal dicocokkan middleware tanpa pesan error yang jelas.
func IsValidPermissionFormat(permission string) bool {
	if permission == "" || len(permission) > 100 {
		return false
	}
	return permission == strings.TrimSpace(permission) &&
		!strings.ContainsAny(permission, " \t\n\r")
}
