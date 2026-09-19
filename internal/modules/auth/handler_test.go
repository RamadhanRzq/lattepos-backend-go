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
	findIDFn func(ctx context.Context, id string) (*users.User, error)
	createFn func(ctx context.Context, u *users.User) error
}

func (s stubUserRepo) FindByUsername(ctx context.Context, username string) (*users.User, error) {
	if s.findFn != nil {
		return s.findFn(ctx, username)
	}
	return nil, users.ErrNotFound
}

func (s stubUserRepo) FindByID(ctx context.Context, id string) (*users.User, error) {
	if s.findIDFn != nil {
		return s.findIDFn(ctx, id)
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

type stubRTRepo struct {
	tokens map[string]*RefreshToken
}

func newStubRTRepo() *stubRTRepo {
	return &stubRTRepo{tokens: make(map[string]*RefreshToken)}
}

func (s *stubRTRepo) Create(_ context.Context, rt *RefreshToken) error {
	rt.ID = "rt-" + rt.TokenHash[:8]
	rt.CreatedAt = time.Now()
	rt.UpdatedAt = time.Now()
	s.tokens[rt.TokenHash] = rt
	return nil
}

func (s *stubRTRepo) FindByHash(_ context.Context, hash string) (*RefreshToken, error) {
	rt, ok := s.tokens[hash]
	if !ok {
		return nil, ErrTokenInvalid
	}
	return rt, nil
}

func (s *stubRTRepo) Revoke(_ context.Context, id string) error {
	for _, rt := range s.tokens {
		if rt.ID == id {
			now := time.Now()
			rt.RevokedAt = &now
			return nil
		}
	}
	return nil
}

func (s *stubRTRepo) RevokeFamily(_ context.Context, familyID string) error {
	now := time.Now()
	for _, rt := range s.tokens {
		if rt.FamilyID == familyID {
			rt.RevokedAt = &now
		}
	}
	return nil
}

func (s *stubRTRepo) RevokeAllByUser(_ context.Context, userID string) error {
	now := time.Now()
	for _, rt := range s.tokens {
		if rt.UserID == userID {
			rt.RevokedAt = &now
		}
	}
	return nil
}

func (s *stubRTRepo) DeleteExpired(_ context.Context) error {
	return nil
}

// --- helpers ---

func newTestJWTManager() *appjwt.Manager {
	return appjwt.NewManager("test-access-secret", 15*time.Minute, "test-refresh-secret", 24*time.Hour)
}

func newHandler(repo stubUserRepo, orgLister OrgLister) *Handler {
	rtRepo := newStubRTRepo()
	userSvc := users.NewService(repo)
	svc := NewService(repo, userSvc, newTestJWTManager(), orgLister, rtRepo)
	return NewHandler(svc)
}

func loginHandler(t *testing.T) (*Handler, *stubRTRepo) {
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
	rtRepo := newStubRTRepo()
	repo := stubUserRepo{
		findFn: func(_ context.Context, username string) (*users.User, error) {
			if username == "alice" {
				return stored, nil
			}
			return nil, users.ErrNotFound
		},
		findIDFn: func(_ context.Context, id string) (*users.User, error) {
			if id == "user-1" {
				return stored, nil
			}
			return nil, users.ErrNotFound
		},
	}
	mgr := newTestJWTManager()
	userSvc := users.NewService(repo)
	svc := NewService(repo, userSvc, mgr, stubOrgLister{}, rtRepo)
	return NewHandler(svc), rtRepo
}

func postJSON(path, body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
}

// doLogin logs in as alice and returns the response body.
func doLogin(t *testing.T, h *Handler) LoginResponse {
	t.Helper()
	body := `{"username":"alice","password":"correct-password"}`
	w := httptest.NewRecorder()
	h.Login(w, postJSON("/api/v1/login", body))
	if w.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}
	var resp LoginResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	return resp
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
		if resp.AccessToken == "" {
			t.Error("access_token empty")
		}
		if resp.RefreshToken == "" {
			t.Error("refresh_token empty")
		}
		if resp.TokenType != "Bearer" {
			t.Errorf("token_type = %q, want Bearer", resp.TokenType)
		}
		if resp.ExpiresIn != 900 {
			t.Errorf("expires_in = %d, want 900", resp.ExpiresIn)
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

	t.Run("nonexistent user", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.Login(w, postJSON("/api/v1/login", `{"username":"ghost","password":"anything"}`))

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
		if resp.AccessToken == "" {
			t.Error("access_token empty")
		}
		if resp.RefreshToken == "" {
			t.Error("refresh_token empty")
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

// --- Refresh tests ---

func TestHandler_Refresh(t *testing.T) {
	h, _ := loginHandler(t)

	t.Run("valid refresh token", func(t *testing.T) {
		loginResp := doLogin(t, h)

		body := `{"refresh_token":"` + loginResp.RefreshToken + `"}`
		w := httptest.NewRecorder()
		h.Refresh(w, postJSON("/api/v1/auth/refresh", body))

		if w.Code != http.StatusOK {
			t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp RefreshResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		if resp.AccessToken == "" {
			t.Error("access_token empty")
		}
		if resp.RefreshToken == "" {
			t.Error("new refresh_token empty")
		}
		if resp.RefreshToken == loginResp.RefreshToken {
			t.Error("refresh token should rotate to new value")
		}
	})

	t.Run("revoked refresh token reuse", func(t *testing.T) {
		loginResp := doLogin(t, h)
		oldRT := loginResp.RefreshToken

		// Use once (rotate)
		body := `{"refresh_token":"` + oldRT + `"}`
		w := httptest.NewRecorder()
		h.Refresh(w, postJSON("/api/v1/auth/refresh", body))
		if w.Code != http.StatusOK {
			t.Fatalf("first refresh want 200, got %d", w.Code)
		}

		// Reuse old token (should fail with reuse detection)
		w2 := httptest.NewRecorder()
		h.Refresh(w2, postJSON("/api/v1/auth/refresh", body))
		if w2.Code != http.StatusUnauthorized {
			t.Fatalf("reuse want 401, got %d: %s", w2.Code, w2.Body.String())
		}
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		body := `{"refresh_token":"totally-invalid-token"}`
		w := httptest.NewRecorder()
		h.Refresh(w, postJSON("/api/v1/auth/refresh", body))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", w.Code)
		}
	})

	t.Run("empty refresh token", func(t *testing.T) {
		body := `{"refresh_token":""}`
		w := httptest.NewRecorder()
		h.Refresh(w, postJSON("/api/v1/auth/refresh", body))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", w.Code)
		}
	})
}

// --- Logout tests ---

func TestHandler_Logout(t *testing.T) {
	h, _ := loginHandler(t)

	t.Run("logout revokes refresh token", func(t *testing.T) {
		loginResp := doLogin(t, h)

		body := `{"refresh_token":"` + loginResp.RefreshToken + `"}`
		w := httptest.NewRecorder()
		h.Logout(w, postJSON("/api/v1/auth/logout", body))

		if w.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", w.Code)
		}

		// Try refresh with revoked token
		w2 := httptest.NewRecorder()
		h.Refresh(w2, postJSON("/api/v1/auth/refresh", body))
		if w2.Code != http.StatusUnauthorized {
			t.Fatalf("want 401 after logout, got %d", w2.Code)
		}
	})
}

// --- LogoutAll tests ---

func TestHandler_LogoutAll(t *testing.T) {
	h, _ := loginHandler(t)

	t.Run("revokes all user tokens", func(t *testing.T) {
		login1 := doLogin(t, h)
		login2 := doLogin(t, h)

		r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout-all", nil)
		ctx := middleware.NewContextWithClaims(r.Context(), &appjwt.Claims{
			UserID: "user-1",
		})
		r = r.WithContext(ctx)
		w := httptest.NewRecorder()
		h.LogoutAll(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", w.Code)
		}

		// Both refresh tokens should be invalid
		for i, rt := range []string{login1.RefreshToken, login2.RefreshToken} {
			body := `{"refresh_token":"` + rt + `"}`
			w := httptest.NewRecorder()
			h.Refresh(w, postJSON("/api/v1/auth/refresh", body))
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("token %d: want 401 after logout-all, got %d", i, w.Code)
			}
		}
	})
}

// --- Me tests ---

func TestHandler_Me(t *testing.T) {
	h, _ := loginHandler(t)

	t.Run("success", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		ctx := middleware.NewContextWithClaims(r.Context(), &appjwt.Claims{
			UserID: "user-1",
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-1",
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
		if resp.UserID != "user-1" {
			t.Errorf("user_id = %q, want user-1", resp.UserID)
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
