package auth

import (
	"github.com/ramadhanrzq/backend-go/internal/modules/organizations"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
)

// LoginResponse adalah payload sukses POST /api/v1/login dan POST /api/v1/register.
type LoginResponse struct {
	AccessToken  string                       `json:"access_token"`
	TokenType    string                       `json:"token_type"`
	ExpiresIn    int                          `json:"expires_in"`
	RefreshToken string                       `json:"refresh_token"`
	User         users.User                   `json:"user"`
	Organizations []organizations.Organization `json:"organizations"`
}

// RefreshResponse adalah payload sukses POST /api/v1/auth/refresh.
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// MeResponse adalah payload GET /api/v1/me.
type MeResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}
