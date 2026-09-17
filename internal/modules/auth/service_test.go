package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ramadhanrzq/backend-go/internal/modules/auth"
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
	svc := auth.NewService(repo, manager)

	t.Run("kredensial valid menerbitkan token dengan identitas user", func(t *testing.T) {
		u, token, err := svc.Login(context.Background(), "  ADMIN ", "secret123")
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
	})

	t.Run("user tidak dikenal dan password salah berbagi error yang sama", func(t *testing.T) {
		_, _, errUnknown := svc.Login(context.Background(), "ghost", "secret123")
		_, _, errWrong := svc.Login(context.Background(), "admin", "salah")

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
