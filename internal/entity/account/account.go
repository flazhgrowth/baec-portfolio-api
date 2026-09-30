package account

import (
	"regexp"
	"time"
	"unicode/utf8"
)

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

const (
	PasswordMinLength = 6
	PasswordMaxLength = 72
)

// usernamePattern is 3-20 letters, digits or underscores. It is matched against the
// already-lowercased username, so uppercase input is fine and is normalised first.
var usernamePattern = regexp.MustCompile(`^[a-z0-9_]{3,20}$`)

type (
	RegisterRequest struct {
		LoginRequest
	}
)

// Validate returns a human-readable reason, or "" when the request is valid. The
// username must already be lowercased (the service does that first).
func (args *RegisterRequest) Validate() string {
	if !usernamePattern.MatchString(args.Username) {
		return "username must be 3-20 characters: letters, digits or underscores"
	}
	if n := utf8.RuneCountInString(args.Password); n < PasswordMinLength || n > PasswordMaxLength {
		return "password must be 6-72 characters"
	}

	return ""
}

type (
	// UserResponse is the contract's User: what GET /auth/me returns.
	UserResponse struct {
		ID        string    `json:"id"`
		Username  string    `json:"username"`
		CreatedAt time.Time `json:"created_at"`
	}
)
