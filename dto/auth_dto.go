package dto

type RegisterRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Fullname  string `json:"fullname" binding:"required,min=3,max=100"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	NIK       string `json:"nik"`
	MPNNumber string `json:"mpn_number"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int64        `json:"expires_in"`
	User         UserResponse `json:"user"`
}

type UserResponse struct {
	ID        string   `json:"id"`
	Username  string   `json:"username"`
	Fullname  string   `json:"fullname"`
	Email     string   `json:"email"`
	NIK       *string  `json:"nik"`        // ← Pointer
	MPNNumber *string  `json:"mpn_number"` // ← Pointer
	Status    string   `json:"status"`
	Roles     []string `json:"roles"`
}
