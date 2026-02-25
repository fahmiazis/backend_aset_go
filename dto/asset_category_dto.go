package dto

import "time"

type CreateAssetCategoryRequest struct {
	CategoryCode string  `json:"category_code" binding:"required,min=2,max=50"`
	CategoryName string  `json:"category_name" binding:"required,min=2,max=255"`
	Description  *string `json:"description"`
	IsActive     bool    `json:"is_active"`
}

type UpdateAssetCategoryRequest struct {
	CategoryCode *string `json:"category_code" binding:"omitempty,min=2,max=50"`
	CategoryName *string `json:"category_name" binding:"omitempty,min=2,max=255"`
	Description  *string `json:"description"`
	IsActive     *bool   `json:"is_active"`
}

type AssetCategoryResponse struct {
	ID           uint      `json:"id"`
	CategoryCode string    `json:"category_code"`
	CategoryName string    `json:"category_name"`
	Description  *string   `json:"description"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
