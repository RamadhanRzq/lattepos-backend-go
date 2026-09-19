package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ramadhanrzq/backend-go/internal/modules/auth"
	"github.com/ramadhanrzq/backend-go/internal/modules/organizations"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

type stubUserRepo struct {
	users.Repository
	findFn func(ctx context.Context, username string) (*users.User, error)
}

func (s stubUserRepo) FindByUsername(ctx context.Context, username string) (*users.User, error) {
	return s.findFn(ctx, username)
}

type stubOrgs struct {
	orgs []organizations.Organization
	err  error
}

func (s stubOrgs) ListUserOrgs(ctx context.Context, userID string) ([]organizations.Organization, error) {
	return s.orgs, s.err
}

type memRTRepo struct {
	tokens map[string]*auth.RefreshToken
}

func newMemRTRepo() *memRTRepo {
	return &memRTRepo{tokens: make(map[string]*auth.RefreshToken)}
}

func (m *memRTRepo) Create(_ context.Context, rt *auth.RefreshToken) error {
	rt.ID = "rt-" + rt.TokenHash[:8]
	rt.CreatedAt = time.Now()
	rt.UpdatedAt = time.Now()
	m.tokens[rt.TokenHash] = rt
	return nil
}

func (m *memRTRepo) FindByHash(_ context.Context, hash string) (*auth.RefreshToken, error) {
	rt, ok := m.tokens[hash]
	if !ok {
		return nil, auth.ErrTokenInvalid
	}
	return rt, nil
}

func (m *memRTRepo) Revoke(_ context.Context, id string) error {
	for _, rt := range m.tokens {
		if rt.ID == id {
			now := time.Now()
			rt.RevokedAt = &now
			return nil
		}
	}
	return nil
}

func (m *memRTRepo) RevokeFamily(_ context.Context, familyID string) error {
	now := time.Now()
	for _, rt := range m.tokens {
		if rt.FamilyID == familyID {
			rt.RevokedAt = &now
		}
	}
	return nil
}

func (m *memRTRepo) RevokeAllByUser(_ context.Context, userID string) error {
	now := time.Now()
	for _, rt := range m.tokens {
		if rt.UserID == userID {
			rt.RevokedAt = &now
		}
	}
	return nil
}

func (m *memRTRepo) DeleteExpired(_ context.Context) error { return nil }

func newTestManager() *appjwt.Manager {
	return appjwt.NewManager("test-secret", time.Minute, "test-refresh-secret", 24*time.Hour)
}

func TestService_Login(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	stored := &users.User{
		ID: "user-1", Username: "admin", Email: "admin@lattepos.com",
		Name: "Administrator", PasswordHash: string(hash),
	}
	repo := stubUserRepo{
		findFn: func(ctx context.Context, username string) (*users.User, error) {
			if username == stored.Username {
				return stored, nil
			}
			return nil, users.ErrNotFound
		},
	}
	manager := newTestManager()
	rtRepo := newMemRTRepo()
	svc := auth.NewService(repo, users.NewService(repo), manager, stubOrgs{orgs: []organizations.Organization{
		{ID: "org-1", Name: "Kopi Nusantara", Slug: "kopi-nusantara"},
		{ID: "org-2", Name: "Kopi Gayo", Slug: "kopi-gayo"},
	}}, rtRepo)

	t.Run("kredensial valid menerbitkan token pair", func(t *testing.T) {
		u, pair, orgs, err := svc.Login(context.Background(), "  ADMIN ", "secret123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.ID != stored.ID {
			t.Fatalf("expected user %q, got %q", stored.ID, u.ID)
		}
		if pair.AccessToken == "" {
			t.Fatal("access token empty")
		}
		if pair.RefreshToken == "" {
			t.Fatal("refresh token empty")
		}
		if pair.ExpiresIn != 60 {
			t.Fatalf("expected expires_in 60, got %d", pair.ExpiresIn)
		}

		claims, err := manager.Verify(pair.AccessToken)
		if err != nil {
			t.Fatalf("token harus bisa diverifikasi: %v", err)
		}
		if claims.UserID != stored.ID {
			t.Fatalf("expected claim user_id %q, got %q", stored.ID, claims.UserID)
		}
		if len(orgs) != 2 || orgs[0].Slug != "kopi-nusantara" {
			t.Fatalf("expected 2 organizations dengan slug, got %+v", orgs)
		}
	})

	t.Run("user tidak dikenal dan password salah berbagi error yang sama", func(t *testing.T) {
		_, _, _, errUnknown := svc.Login(context.Background(), "ghost", "secret123")
		_, _, _, errWrong := svc.Login(context.Background(), "admin", "salah")

		if !errors.Is(errUnknown, users.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials untuk user tidak dikenal, got %v", errUnknown)
		}
		if !errors.Is(errWrong, users.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials untuk password salah, got %v", errWrong)
		}
		if errUnknown.Error() != errWrong.Error() {
			t.Fatalf("error harus seragam supaya tidak membocorkan user mana yang ada: %q vs %q", errUnknown, errWrong)
		}
	})
}

type registerRepo struct {
	users.Repository
	taken map[string]bool
}

func (r registerRepo) Create(ctx context.Context, u *users.User) error {
	if r.taken[u.Username] {
		return users.ErrUsernameTaken
	}
	u.ID = "user-2"
	return nil
}

func TestService_Register(t *testing.T) {
	manager := newTestManager()
	repo := registerRepo{taken: map[string]bool{"budi": true}}
	rtRepo := newMemRTRepo()
	svc := auth.NewService(repo, users.NewService(repo), manager, nil, rtRepo)

	t.Run("registrasi valid menerbitkan token pair untuk user baru", func(t *testing.T) {
		u, pair, orgs, err := svc.Register(context.Background(), "Ani", "Ani", "ani@lattepos.com", "secret123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(orgs) != 0 {
			t.Fatalf("expected organizations kosong untuk user baru, got %+v", orgs)
		}
		if u.Username != "ani" {
			t.Fatalf("expected username ternormalisasi %q, got %q", "ani", u.Username)
		}
		if pair.AccessToken == "" || pair.RefreshToken == "" {
			t.Fatal("token pair should not be empty")
		}
		claims, err := manager.Verify(pair.AccessToken)
		if err != nil {
			t.Fatalf("token harus bisa diverifikasi: %v", err)
		}
		if claims.UserID != u.ID {
			t.Fatalf("expected claim user_id %q, got %q", u.ID, claims.UserID)
		}
	})

	t.Run("username terpakai dan input kosong dipetakan ke error domain", func(t *testing.T) {
		_, _, _, errTaken := svc.Register(context.Background(), "BUDI", "Budi", "budi@lattepos.com", "secret123")
		if !errors.Is(errTaken, users.ErrUsernameTaken) {
			t.Fatalf("expected ErrUsernameTaken, got %v", errTaken)
		}
		_, _, _, errEmpty := svc.Register(context.Background(), "", "", "", "")
		if !errors.Is(errEmpty, users.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", errEmpty)
		}
	})
}

func TestService_Refresh(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
	stored := &users.User{ID: "user-1", Username: "admin", PasswordHash: string(hash)}
	repo := stubUserRepo{
		findFn: func(_ context.Context, username string) (*users.User, error) {
			if username == "admin" {
				return stored, nil
			}
			return nil, users.ErrNotFound
		},
	}
	manager := newTestManager()
	rtRepo := newMemRTRepo()
	svc := auth.NewService(repo, users.NewService(repo), manager, nil, rtRepo)

	t.Run("rotation success", func(t *testing.T) {
		_, pair, _, err := svc.Login(context.Background(), "admin", "pass")
		if err != nil {
			t.Fatal(err)
		}

		newPair, err := svc.Refresh(context.Background(), pair.RefreshToken)
		if err != nil {
			t.Fatalf("refresh failed: %v", err)
		}
		if newPair.AccessToken == "" || newPair.RefreshToken == "" {
			t.Fatal("new pair should not be empty")
		}
		if newPair.RefreshToken == pair.RefreshToken {
			t.Fatal("refresh token should rotate")
		}
	})

	t.Run("reuse detection revokes family", func(t *testing.T) {
		_, pair, _, err := svc.Login(context.Background(), "admin", "pass")
		if err != nil {
			t.Fatal(err)
		}
		oldRT := pair.RefreshToken

		// First refresh (ok)
		_, err = svc.Refresh(context.Background(), oldRT)
		if err != nil {
			t.Fatalf("first refresh should succeed: %v", err)
		}

		// Reuse old token
		_, err = svc.Refresh(context.Background(), oldRT)
		if !errors.Is(err, auth.ErrTokenReuse) {
			t.Fatalf("expected ErrTokenReuse, got %v", err)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := svc.Refresh(context.Background(), "invalid-token")
		if err == nil {
			t.Fatal("expected error for invalid token")
		}
	})
}

func TestService_Logout(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
	stored := &users.User{ID: "user-1", Username: "admin", PasswordHash: string(hash)}
	repo := stubUserRepo{
		findFn: func(_ context.Context, username string) (*users.User, error) {
			if username == "admin" {
				return stored, nil
			}
			return nil, users.ErrNotFound
		},
	}
	rtRepo := newMemRTRepo()
	svc := auth.NewService(repo, users.NewService(repo), newTestManager(), nil, rtRepo)

	t.Run("logout revokes token", func(t *testing.T) {
		_, pair, _, err := svc.Login(context.Background(), "admin", "pass")
		if err != nil {
			t.Fatal(err)
		}

		if err := svc.Logout(context.Background(), pair.RefreshToken); err != nil {
			t.Fatalf("logout failed: %v", err)
		}

		// Refresh should fail
		_, err = svc.Refresh(context.Background(), pair.RefreshToken)
		if err == nil {
			t.Fatal("refresh should fail after logout")
		}
	})
}

func TestService_LogoutAll(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
	stored := &users.User{ID: "user-1", Username: "admin", PasswordHash: string(hash)}
	repo := stubUserRepo{
		findFn: func(_ context.Context, username string) (*users.User, error) {
			if username == "admin" {
				return stored, nil
			}
			return nil, users.ErrNotFound
		},
	}
	rtRepo := newMemRTRepo()
	svc := auth.NewService(repo, users.NewService(repo), newTestManager(), nil, rtRepo)

	_, pair1, _, _ := svc.Login(context.Background(), "admin", "pass")
	_, pair2, _, _ := svc.Login(context.Background(), "admin", "pass")

	if err := svc.LogoutAll(context.Background(), "user-1"); err != nil {
		t.Fatalf("logout-all failed: %v", err)
	}

	for i, rt := range []string{pair1.RefreshToken, pair2.RefreshToken} {
		_, err := svc.Refresh(context.Background(), rt)
		if err == nil {
			t.Fatalf("token %d: refresh should fail after logout-all", i)
		}
	}
}
