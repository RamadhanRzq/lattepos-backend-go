package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/organizations"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

// --- stubs ---

type stubUserRepo struct {
	users.Repository
	findFn   func(ctx context.Context, username string) (*users.User, error)
	createFn func(ctx context.Context, u *users.User) error
}

func (s stubUserRepo) FindByUsername(ctx context.Context, username string) (*users.User, error) {
	if s.findFn != nil {
		return s.findFn(ctx, username)
	}
	return nil, users.ErrNotFound
}

func (s stubUserRepo) Create(ctx context.Context, u *users.User) error {
	if s.createFn != nil {
		return s.createFn(ctx, u)
	}
	u.ID = "user-new"
	return nil
}

type stubOrgLister struct {
	orgs []organizations.Organization
	err  error
}

func (s stubOrgLister) ListUserOrgs(_ context.Context, _ string) ([]organizations.Organization, error) {
	return s.orgs, s.err
}

// --- helpers ---

func newHandler(repo stubUserRepo, orgLister OrgLister) *Handler {
	jwtMgr := appjwt.NewManager("test-secret-key", time.Hour)
	userSvc := users.NewService(repo)
	svc := NewService(repo, userSvc, jwtMgr, orgLister)
	return NewHandler(svc)
}

func loginHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	stored := &users.User{
		ID:           "user-1",
		Username:     "alice",
		Name:         "Alice",
		Email:        "alice@test.com",
		PasswordHash: string(hash),
	}
	repo := stubUserRepo{
		findFn: func(_ context.Context, username string) (*users.User, error) {
			if username == "alice" {
				return stored, nil
			}
			return nil, users.ErrNotFound
		},
	}
	return newHandler(repo, stubOrgLister{}), string(hash)
}

func postJSON(path, body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
}

// --- Login tests ---

func TestHandler_Login(t *testing.T) {
	h, _ := loginHandler(t)

	t.Run("success", func(t *testing.T) {
		body := `{"username":"alice","password":"correct-password"}`
		w := httptest.NewRecorder()
		h.Login(w, postJSON("/api/v1/login", body))

		if w.Code != http.StatusOK {
			t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp LoginResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		if resp.Token == "" {
			t.Error("token empty")
		}
		if resp.TokenType != "Bearer" {
			t.Errorf("token_type = %q, want Bearer", resp.TokenType)
		}
		if resp.User.Username != "alice" {
			t.Errorf("user.username = %q, want alice", resp.User.Username)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.Login(w, postJSON("/api/v1/login", "{bad"))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", w.Code)
		}
	})

	t.Run("empty fields", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.Login(w, postJSON("/api/v1/login", `{"username":"","password":""}`))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", w.Code)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.Login(w, postJSON("/api/v1/login", `{"username":"alice","password":"wrong"}`))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", w.Code)
		}
	})
}

// --- Register tests ---

func TestHandler_Register(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := stubUserRepo{
			createFn: func(_ context.Context, u *users.User) error {
				u.ID = "user-new"
				return nil
			},
		}
		h := newHandler(repo, stubOrgLister{})

		body := `{"username":"bob","name":"Bob","email":"bob@test.com","password":"s3cret"}`
		w := httptest.NewRecorder()
		h.Register(w, postJSON("/api/v1/register", body))

		if w.Code != http.StatusCreated {
			t.Fatalf("want 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp LoginResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		if resp.Token == "" {
			t.Error("token empty")
		}
		if resp.User.Username != "bob" {
			t.Errorf("user.username = %q, want bob", resp.User.Username)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		h := newHandler(stubUserRepo{}, stubOrgLister{})
		w := httptest.NewRecorder()
		h.Register(w, postJSON("/api/v1/register", "not-json"))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", w.Code)
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		h := newHandler(stubUserRepo{}, stubOrgLister{})
		// username empty triggers ErrInvalidInput from users.Service.Create
		body := `{"username":"","name":"","email":"","password":""}`
		w := httptest.NewRecorder()
		h.Register(w, postJSON("/api/v1/register", body))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", w.Code)
		}
	})

	t.Run("duplicate username", func(t *testing.T) {
		repo := stubUserRepo{
			createFn: func(_ context.Context, u *users.User) error {
				return users.ErrUsernameTaken
			},
		}
		h := newHandler(repo, stubOrgLister{})

		body := `{"username":"alice","name":"Alice","email":"a@b.com","password":"pass"}`
		w := httptest.NewRecorder()
		h.Register(w, postJSON("/api/v1/register", body))

		if w.Code != http.StatusConflict {
			t.Fatalf("want 409, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// --- Me tests ---

func TestHandler_Me(t *testing.T) {
	h := newHandler(stubUserRepo{}, stubOrgLister{})

	t.Run("success", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		ctx := middleware.NewContextWithClaims(r.Context(), &appjwt.Claims{
			UserID:   "u-1",
			Username: "alice",
			Email:    "alice@test.com",
			Name:     "Alice",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		})
		r = r.WithContext(ctx)
		w := httptest.NewRecorder()
		h.Me(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp MeResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		if resp.UserID != "u-1" {
			t.Errorf("user_id = %q, want u-1", resp.UserID)
		}
		if resp.Username != "alice" {
			t.Errorf("username = %q, want alice", resp.Username)
		}
	})

	t.Run("no claims in context", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		w := httptest.NewRecorder()
		h.Me(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", w.Code)
		}
	})
}
