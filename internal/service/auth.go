package service

import (
	"context"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ramadhanrzq/backend-go/internal/domain/user"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

// AuthService menangani autentikasi: verifikasi kredensial dan penerbitan token.
type AuthService struct {
	users user.UserRepository
	jwt   *appjwt.Manager
}

func NewAuthService(users user.UserRepository, jwt *appjwt.Manager) *AuthService {
	return &AuthService{users: users, jwt: jwt}
}

// Login memvalidasi username & password, lalu menerbitkan token JWT.
// Error selalu ErrInvalidCredentials untuk kredensial salah —
// jangan bocorkan apakah username atau password yang keliru.
func (s *AuthService) Login(ctx context.Context, username, password string) (user.User, string, error) {
	username = strings.TrimSpace(strings.ToLower(username))

	u, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return user.User{}, "", user.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return user.User{}, "", user.ErrInvalidCredentials
	}

	token, err := s.jwt.Generate(u.ID, u.Username, u.Email, u.Name)
	if err != nil {
		return user.User{}, "", err
	}

	return *u, token, nil
}

// TokenExpiration mengembalikan durasi masa berlaku token yang diterbitkan.
func (s *AuthService) TokenExpiration() time.Duration {
	return s.jwt.Expiration()
}

// VerifyToken memvalidasi token JWT dan mengembalikan claims-nya.
func (s *AuthService) VerifyToken(tokenString string) (*appjwt.Claims, error) {
	return s.jwt.Verify(tokenString)
}
