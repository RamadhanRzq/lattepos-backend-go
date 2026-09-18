package auth

import (
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/organizations"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
)

// LoginResponse adalah payload sukses POST /api/v1/login dan POST /api/v1/register.
// Organizations memuat org user supaya client bisa redirect ke /org/{slug}
// masing-masing; kosong bila user belum punya organisasi.
type LoginResponse struct {
	Token         string                       `json:"token"`
	TokenType     string                       `json:"token_type"`
	ExpiresAt     time.Time                    `json:"expires_at"`
	User          users.User                   `json:"user"` // PasswordHash tersembunyi oleh tag json:"-"
	Organizations []organizations.Organization `json:"organizations"`
}

// MeResponse adalah payload GET /api/v1/me.
type MeResponse struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	ExpiresAt time.Time `json:"expires_at"`
}
