package dto

import "time"

// CreateBranchRequest represents the request body for creating a branch
type CreateBranchRequest struct {
	BranchName string `json:"branch_name" binding:"required"`
	BranchType string `json:"branch_type" binding:"required"`
	Status     string `json:"status" binding:"omitempty,oneof=active inactive"`
}

// UpdateBranchRequest represents the request body for updating a branch
type UpdateBranchRequest struct {
	BranchName string `json:"branch_name" binding:"omitempty"`
	BranchType string `json:"branch_type" binding:"omitempty"`
	Status     string `json:"status" binding:"omitempty,oneof=active inactive"`
}

// BranchResponse represents the branch data in response
type BranchResponse struct {
	ID         string    `json:"id"`
	BranchCode string    `json:"branch_code"`
	BranchName string    `json:"branch_name"`
	BranchType string    `json:"branch_type"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// AssignBranchRequest represents the request to assign branchs to a user
type AssignBranchRequest struct {
	BranchIDs []string `json:"branch_ids" binding:"required,min=1"`
}

// UserSimpleResponse represents simplified user data for branch users
type UserSimpleResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Email    string `json:"email"`
	Status   string `json:"status"`
}

// UserBranchResponse represents user with their branchs
type UserBranchResponse struct {
	ID       string           `json:"id"`
	Username string           `json:"username"`
	Fullname string           `json:"fullname"`
	Email    string           `json:"email"`
	Branchs  []BranchResponse `json:"branchs"`
}

// ============================================================================
// HOMEBASE — pengelolaan anggota homebase dari sisi cabang
// ============================================================================

// AssignHomebaseUsersRequest — set banyak user sekaligus agar homebase-nya
// menjadi cabang ini.
type AssignHomebaseUsersRequest struct {
	UserIDs []string `json:"user_ids" binding:"required,min=1"`
}

// HomebaseUserResponse — satu user yang homebase-nya di cabang ini
type HomebaseUserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Email    string `json:"email"`
	Status   string `json:"status"`
	// IsActive menandai apakah cabang ini homebase yang sedang aktif bagi user
	// tersebut. User bisa punya beberapa homebase, tapi hanya satu yang aktif.
	IsActive bool `json:"is_active"`
}

// AssignBranchUsersRequest — beri banyak user sekaligus akses ke cabang ini
type AssignBranchUsersRequest struct {
	UserIDs []string `json:"user_ids" binding:"required,min=1"`
}

// BranchAssignedUserResponse — user yang di-assign ke cabang ini (non-homebase)
type BranchAssignedUserResponse struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	Fullname   string `json:"fullname"`
	Email      string `json:"email"`
	Status     string `json:"status"`
	BranchType string `json:"branch_type"`
}
