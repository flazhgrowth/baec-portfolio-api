package account

import "github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"

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
	MeResponse struct {
		AccountInfo entity.AccountInfo `json:"me"`
	}
)
