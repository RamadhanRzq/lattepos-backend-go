package dto

import (
	"time"

	"github.com/ramadhanrzq/backend-go/internal/domain/user"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	TokenType string    `json:"token_type"`
	ExpiresAt time.Time `json:"expires_at"`
	User      user.User `json:"user"` // PasswordHash tersembunyi oleh tag json:"-"
}

type MeResponse struct {
	UserID    string       `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	ExpiresAt time.Time `json:"expires_at"`
}
