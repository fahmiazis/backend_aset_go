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
