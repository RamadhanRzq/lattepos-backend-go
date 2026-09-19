package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ramadhanrzq/backend-go/internal/modules/organizations"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

var (
	ErrTokenRevoked = errors.New("refresh token sudah direvoke")
	ErrTokenExpired = errors.New("refresh token sudah kedaluwarsa")
	ErrTokenInvalid = errors.New("refresh token tidak valid")
	ErrTokenReuse   = errors.New("reuse terdeteksi: seluruh token family direvoke")
)

// OrgLister adalah port daftar organisasi user. Dipenuhi struktural oleh
// organizations.Service (method ListUserOrgs); nil berarti tanpa daftar org.
type OrgLister interface {
	ListUserOrgs(ctx context.Context, userID string) ([]organizations.Organization, error)
}

// TokenPair adalah hasil login/refresh yang diberikan ke client.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// Service menangani autentikasi: verifikasi kredensial, registrasi,
// penerbitan token pair, refresh rotation, dan revokasi.
type Service struct {
	users    users.Repository
	userSvc  *users.Service
	jwt      *appjwt.Manager
	orgs     OrgLister
	rtRepo   RefreshTokenRepository
}

func NewService(
	userRepo users.Repository,
	userSvc *users.Service,
	jwtManager *appjwt.Manager,
	orgs OrgLister,
	rtRepo RefreshTokenRepository,
) *Service {
	return &Service{
		users:  userRepo,
		userSvc: userSvc,
		jwt:    jwtManager,
		orgs:   orgs,
		rtRepo: rtRepo,
	}
}

// Login memvalidasi username & password, menerbitkan token pair, lalu melampirkan
// daftar organisasi user.
func (s *Service) Login(ctx context.Context, username, password string) (users.User, TokenPair, []organizations.Organization, error) {
	username = strings.TrimSpace(strings.ToLower(username))

	u, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return users.User{}, TokenPair{}, nil, users.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return users.User{}, TokenPair{}, nil, users.ErrInvalidCredentials
	}

	pair, err := s.issueTokenPair(ctx, u.ID, "")
	if err != nil {
		return users.User{}, TokenPair{}, nil, err
	}

	return *u, pair, s.userOrgs(ctx, u.ID), nil
}

// Register mendaftarkan user baru lalu menerbitkan token pair (auto-login).
func (s *Service) Register(ctx context.Context, username, name, email, password string) (users.User, TokenPair, []organizations.Organization, error) {
	u, err := s.userSvc.Create(ctx, username, name, email, password)
	if err != nil {
		return users.User{}, TokenPair{}, nil, err
	}

	pair, err := s.issueTokenPair(ctx, u.ID, "")
	if err != nil {
		return users.User{}, TokenPair{}, nil, err
	}

	return u, pair, []organizations.Organization{}, nil
}

// Refresh melakukan token rotation: revoke old, issue new pair.
// Jika token yang sudah direvoke dipakai ulang, seluruh family direvoke.
func (s *Service) Refresh(ctx context.Context, rawRefreshToken string) (TokenPair, error) {
	hash := HashToken(rawRefreshToken)

	stored, err := s.rtRepo.FindByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TokenPair{}, ErrTokenInvalid
		}
		return TokenPair{}, err
	}

	// Reuse detection: token sudah direvoke = kemungkinan token theft.
	if stored.RevokedAt != nil {
		_ = s.rtRepo.RevokeFamily(ctx, stored.FamilyID)
		return TokenPair{}, ErrTokenReuse
	}

	if time.Now().After(stored.ExpiresAt) {
		return TokenPair{}, ErrTokenExpired
	}

	// Revoke token lama terlebih dahulu.
	if err := s.rtRepo.Revoke(ctx, stored.ID); err != nil {
		return TokenPair{}, err
	}

	// Issue new pair dalam family yang sama.
	pair, err := s.issueTokenPair(ctx, stored.UserID, stored.FamilyID)
	if err != nil {
		return TokenPair{}, err
	}

	return pair, nil
}

// Logout merevoke satu refresh token.
func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := HashToken(rawRefreshToken)

	stored, err := s.rtRepo.FindByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	return s.rtRepo.Revoke(ctx, stored.ID)
}

// LogoutAll merevoke seluruh refresh token milik user.
func (s *Service) LogoutAll(ctx context.Context, userID string) error {
	return s.rtRepo.RevokeAllByUser(ctx, userID)
}

// VerifyToken memvalidasi access token JWT dan mengembalikan claims-nya.
// Method ini memenuhi port middleware.TokenVerifier.
func (s *Service) VerifyToken(tokenString string) (*appjwt.Claims, error) {
	return s.jwt.Verify(tokenString)
}

// AccessTTL mengembalikan durasi masa berlaku access token.
func (s *Service) AccessTTL() time.Duration {
	return s.jwt.AccessTTL()
}

// issueTokenPair membuat access + refresh token dan menyimpan hash refresh token.
// familyID kosong = session baru (login/register); non-kosong = rotation.
func (s *Service) issueTokenPair(ctx context.Context, userID, familyID string) (TokenPair, error) {
	accessToken, err := s.jwt.GenerateAccess(userID)
	if err != nil {
		return TokenPair{}, err
	}

	rawRefresh, expiresAt, err := s.jwt.GenerateRefresh()
	if err != nil {
		return TokenPair{}, err
	}

	if familyID == "" {
		familyID = randomFamilyID()
	}

	rt := &RefreshToken{
		UserID:    userID,
		TokenHash: HashToken(rawRefresh),
		FamilyID:  familyID,
		ExpiresAt: expiresAt,
	}
	if err := s.rtRepo.Create(ctx, rt); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int(s.jwt.AccessTTL().Seconds()),
	}, nil
}

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

func randomFamilyID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
