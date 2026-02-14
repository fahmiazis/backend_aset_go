package dto

import "time"

type CreateUserRequest struct {
	Username  string   `json:"username" binding:"required,min=3,max=50"`
	Fullname  string   `json:"fullname" binding:"required,min=3,max=100"`
	Email     string   `json:"email" binding:"required,email"`
	Password  string   `json:"password" binding:"required,min=6"`
	NIK       string   `json:"nik"`
	MPNNumber string   `json:"mpn_number"`
	Status    string   `json:"status" binding:"omitempty,oneof=active inactive"`
	RoleIDs   []string `json:"role_ids" binding:"required,min=1"`
}

type UpdateUserRequest struct {
	Fullname  string `json:"fullname" binding:"omitempty,min=3,max=100"`
	Email     string `json:"email" binding:"omitempty,email"`
	Password  string `json:"password" binding:"omitempty,min=6"`
	NIK       string `json:"nik"`
	MPNNumber string `json:"mpn_number"`
	Status    string `json:"status" binding:"omitempty,oneof=active inactive"`
}

type UserDetailResponse struct {
	ID        string         `json:"id"`
	Username  string         `json:"username"`
	Fullname  string         `json:"fullname"`
	Email     string         `json:"email"`
	NIK       *string        `json:"nik"`        // ← Pointer
	MPNNumber *string        `json:"mpn_number"` // ← Pointer
	Status    string         `json:"status"`
	Roles     []RoleResponse `json:"roles"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type RoleResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AssignRoleRequest struct {
	RoleIDs []string `json:"role_ids" binding:"required,min=1"`
}
