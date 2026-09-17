package user

import (
	"context"
	"time"

	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

// AuthService adalah kontrak untuk layanan autentikasi.
// Implementasi konkret ada di internal/service.
type AuthService interface {
	Login(ctx context.Context, username, password string) (User, string, error)
	VerifyToken(tokenString string) (*appjwt.Claims, error)
	TokenExpiration() time.Duration
}

// UserService adalah kontrak untuk layanan user.
type UserService interface {
	List(ctx context.Context) ([]User, error)
	GetByID(ctx context.Context, id string) (User, error)
	Create(ctx context.Context, username, name, email, password string) (User, error)
}
