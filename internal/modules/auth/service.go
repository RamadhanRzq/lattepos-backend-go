package auth

import (
	"context"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ramadhanrzq/backend-go/internal/modules/users"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

// Service menangani autentikasi: verifikasi kredensial dan penerbitan token.
type Service struct {
	users users.Repository
	jwt   *appjwt.Manager
}

func NewService(userRepo users.Repository, jwtManager *appjwt.Manager) *Service {
	return &Service{users: userRepo, jwt: jwtManager}
}

// Login memvalidasi username & password, lalu menerbitkan token JWT.
// Error selalu users.ErrInvalidCredentials untuk kredensial salah —
// jangan bocorkan apakah username atau password yang keliru.
func (s *Service) Login(ctx context.Context, username, password string) (users.User, string, error) {
	username = strings.TrimSpace(strings.ToLower(username))

	u, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return users.User{}, "", users.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return users.User{}, "", users.ErrInvalidCredentials
	}

	token, err := s.jwt.Generate(u.ID, u.Username, u.Email, u.Name)
	if err != nil {
		return users.User{}, "", err
	}

	return *u, token, nil
}

// TokenExpiration mengembalikan durasi masa berlaku token yang diterbitkan.
func (s *Service) TokenExpiration() time.Duration {
	return s.jwt.Expiration()
}

// VerifyToken memvalidasi token JWT dan mengembalikan claims-nya.
// Method ini yang memenuhi port middleware.TokenVerifier.
func (s *Service) VerifyToken(tokenString string) (*appjwt.Claims, error) {
	return s.jwt.Verify(tokenString)
}
