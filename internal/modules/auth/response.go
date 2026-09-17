package auth

import (
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/users"
)

// LoginResponse adalah payload sukses POST /login.
type LoginResponse struct {
	Token     string     `json:"token"`
	TokenType string     `json:"token_type"`
	ExpiresAt time.Time  `json:"expires_at"`
	User      users.User `json:"user"` // PasswordHash tersembunyi oleh tag json:"-"
}

// MeResponse adalah payload GET /me.
type MeResponse struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	ExpiresAt time.Time `json:"expires_at"`
}
