package auth

import (
	"context"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ramadhanrzq/backend-go/internal/modules/organizations"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

// OrgLister adalah port daftar organisasi user. Dipenuhi struktural oleh
// organizations.Service (method ListUserOrgs); nil berarti tanpa daftar org.
type OrgLister interface {
	ListUserOrgs(ctx context.Context, userID string) ([]organizations.Organization, error)
}

// Service menangani autentikasi: verifikasi kredensial, registrasi, dan penerbitan token.
// userSvc dipakai ulang untuk registrasi supaya aturan create tetap satu sumber (users.Service.Create).
// Ponytail: registrasi publik selalu role cashier, tanpa verifikasi email/rate-limit; tambah saat dibutuhkan.
type Service struct {
	users   users.Repository
	userSvc *users.Service
	jwt     *appjwt.Manager
	orgs    OrgLister
}

func NewService(userRepo users.Repository, userSvc *users.Service, jwtManager *appjwt.Manager, orgs OrgLister) *Service {
	return &Service{users: userRepo, userSvc: userSvc, jwt: jwtManager, orgs: orgs}
}

// Login memvalidasi username & password, menerbitkan token JWT, lalu melampirkan
// daftar organisasi user supaya client bisa redirect ke /org/{slug} masing-masing.
// Daftar org best-effort: gagal query tidak menggagalkan login.
// Error selalu users.ErrInvalidCredentials untuk kredensial salah —
// jangan bocorkan apakah username atau password yang keliru.
func (s *Service) Login(ctx context.Context, username, password string) (users.User, string, []organizations.Organization, error) {
	username = strings.TrimSpace(strings.ToLower(username))

	u, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return users.User{}, "", nil, users.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return users.User{}, "", nil, users.ErrInvalidCredentials
	}

	token, err := s.jwt.Generate(u.ID, u.Username, u.Email, u.Name)
	if err != nil {
		return users.User{}, "", nil, err
	}

	return *u, token, s.userOrgs(ctx, u.ID), nil
}

// Register mendaftarkan user baru lalu menerbitkan token (auto-login).
// User baru belum punya organisasi: organizations selalu slice kosong.
func (s *Service) Register(ctx context.Context, username, name, email, password string) (users.User, string, []organizations.Organization, error) {
	u, err := s.userSvc.Create(ctx, username, name, email, password)
	if err != nil {
		return users.User{}, "", nil, err
	}

	token, err := s.jwt.Generate(u.ID, u.Username, u.Email, u.Name)
	if err != nil {
		return users.User{}, "", nil, err
	}

	return u, token, []organizations.Organization{}, nil
}

// userOrgs mengembalikan organisasi user; nil bila port tak dipasang atau query gagal.
func (s *Service) userOrgs(ctx context.Context, userID string) []organizations.Organization {
	if s.orgs == nil {
		return []organizations.Organization{}
	}
	orgs, err := s.orgs.ListUserOrgs(ctx, userID)
	if err != nil {
		return []organizations.Organization{}
	}
	if orgs == nil {
		return []organizations.Organization{}
	}
	return orgs
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
