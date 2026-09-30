package account

import "time"

type (
	LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	LoginResponse struct {
		ID           string `json:"id"`
		Username     string `json:"username"`
		AccessToken  string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
)

type (
	RegisterRequest struct {
		LoginRequest
	}
)

type (
	// UserResponse is the contract's User: what GET /auth/me returns.
	UserResponse struct {
		ID        string    `json:"id"`
		Username  string    `json:"username"`
		CreatedAt time.Time `json:"created_at"`
	}
)
