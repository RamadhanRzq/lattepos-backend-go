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
	manager := appjwt.NewManager("test-secret", time.Minute)
	svc := auth.NewService(repo, users.NewService(repo), manager, stubOrgs{orgs: []organizations.Organization{
		{ID: "org-1", Name: "Kopi Nusantara", Slug: "kopi-nusantara"},
		{ID: "org-2", Name: "Kopi Gayo", Slug: "kopi-gayo"},
	}})

	t.Run("kredensial valid menerbitkan token dengan identitas user", func(t *testing.T) {
		u, token, orgs, err := svc.Login(context.Background(), "  ADMIN ", "secret123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.ID != stored.ID {
			t.Fatalf("expected user %q, got %q", stored.ID, u.ID)
		}

		claims, err := manager.Verify(token)
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
	manager := appjwt.NewManager("test-secret", time.Minute)
	repo := registerRepo{taken: map[string]bool{"budi": true}}
	svc := auth.NewService(repo, users.NewService(repo), manager, nil)

	t.Run("registrasi valid menerbitkan token untuk user baru", func(t *testing.T) {
		u, token, orgs, err := svc.Register(context.Background(), "Ani", "Ani", "ani@lattepos.com", "secret123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(orgs) != 0 {
			t.Fatalf("expected organizations kosong untuk user baru, got %+v", orgs)
		}
		if u.Username != "ani" {
			t.Fatalf("expected username ternormalisasi %q, got %q", "ani", u.Username)
		}
		claims, err := manager.Verify(token)
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
