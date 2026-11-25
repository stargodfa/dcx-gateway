package dto

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthUser struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type LoginResponse struct {
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int      `json:"expires_in"`
	User        AuthUser `json:"user"`
}

type MeResponse struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type LogoutResponse struct {
	OK bool `json:"ok"`
}
