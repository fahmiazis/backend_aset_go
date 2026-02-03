package dto

// CreateCabangRequest represents the request body for creating a cabang
type CreateCabangRequest struct {
	NamaCabang  string `json:"nama_cabang" binding:"required"`
	JenisCabang string `json:"jenis_cabang" binding:"required"`
	Status      string `json:"status" binding:"omitempty,oneof=active inactive"`
}

// UpdateCabangRequest represents the request body for updating a cabang
type UpdateCabangRequest struct {
	NamaCabang  string `json:"nama_cabang" binding:"omitempty"`
	JenisCabang string `json:"jenis_cabang" binding:"omitempty"`
	Status      string `json:"status" binding:"omitempty,oneof=active inactive"`
}

// CabangResponse represents the cabang data in response
type CabangResponse struct {
	ID          string `json:"id"`
	KodeCabang  string `json:"kode_cabang"`
	NamaCabang  string `json:"nama_cabang"`
	JenisCabang string `json:"jenis_cabang"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// AssignCabangRequest represents the request to assign cabangs to a user
type AssignCabangRequest struct {
	CabangIDs []string `json:"cabang_ids" binding:"required,min=1"`
}

// UserSimpleResponse represents simplified user data for cabang users
type UserSimpleResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Email    string `json:"email"`
	Status   string `json:"status"`
}

// UserCabangResponse represents user with their cabangs
type UserCabangResponse struct {
	ID       string           `json:"id"`
	Username string           `json:"username"`
	Fullname string           `json:"fullname"`
	Email    string           `json:"email"`
	Cabangs  []CabangResponse `json:"cabangs"`
}
