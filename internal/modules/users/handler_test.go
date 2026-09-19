package users

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
)

func withOrgID(r *http.Request, orgID string) *http.Request {
	return r.WithContext(middleware.NewContextWithOrgID(r.Context(), orgID))
}

// --- stub repository ---

type stubRepo struct {
	listFn         func(ctx context.Context) ([]*User, error)
	createFn       func(ctx context.Context, u *User) error
	findByIDFn     func(ctx context.Context, id string) (*User, error)
	findByUserFn   func(ctx context.Context, username string) (*User, error)
	findByIDInOrgFn func(ctx context.Context, orgID, id string) (*OrgUser, error)
	listByOrgFn    func(ctx context.Context, orgID string) ([]*OrgUser, error)
}

func (s *stubRepo) List(ctx context.Context) ([]*User, error) {
	return s.listFn(ctx)
}
func (s *stubRepo) Create(ctx context.Context, u *User) error {
	return s.createFn(ctx, u)
}
func (s *stubRepo) FindByID(ctx context.Context, id string) (*User, error) {
	return s.findByIDFn(ctx, id)
}
func (s *stubRepo) FindByUsername(ctx context.Context, username string) (*User, error) {
	return s.findByUserFn(ctx, username)
}
func (s *stubRepo) FindByIDInOrg(ctx context.Context, orgID, id string) (*OrgUser, error) {
	return s.findByIDInOrgFn(ctx, orgID, id)
}
func (s *stubRepo) ListByOrg(ctx context.Context, orgID string) ([]*OrgUser, error) {
	return s.listByOrgFn(ctx, orgID)
}

// helpers

func newHandler(repo *stubRepo) *Handler {
	return NewHandler(NewService(repo))
}

func decodeJSON(t *testing.T, body *bytes.Buffer, v any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// --- List ---

func TestHandler_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			listFn: func(ctx context.Context) ([]*User, error) {
				return []*User{{ID: "u1", Name: "Alice"}}, nil
			},
		}
		h := newHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		h.List(rr, r)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var got []User
		decodeJSON(t, rr.Body, &got)
		if len(got) != 1 || got[0].ID != "u1" {
			t.Fatalf("body = %+v, want [{ID:u1}]", got)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &stubRepo{
			listFn: func(ctx context.Context) ([]*User, error) {
				return nil, errors.New("db down")
			},
		}
		h := newHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		h.List(rr, r)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})
}

// --- Create ---

func TestHandler_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			findByUserFn: func(ctx context.Context, username string) (*User, error) {
				return nil, ErrNotFound
			},
			createFn: func(ctx context.Context, u *User) error {
				u.ID = "new-id"
				return nil
			},
		}
		h := newHandler(repo)

		body := `{"username":"bob","name":"Bob","email":"bob@x.com","password":"secret123"}`
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(body))
		h.Create(rr, r)

		if rr.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusCreated, rr.Body.String())
		}
		var got User
		decodeJSON(t, rr.Body, &got)
		if got.Username != "bob" {
			t.Fatalf("username = %q, want bob", got.Username)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		h := newHandler(&stubRepo{})

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString("{bad"))
		h.Create(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		// empty fields → ErrInvalidInput from service
		repo := &stubRepo{
			findByUserFn: func(ctx context.Context, username string) (*User, error) {
				return nil, ErrNotFound
			},
			createFn: func(ctx context.Context, u *User) error { return nil },
		}
		h := newHandler(repo)

		body := `{"username":"","name":"","email":"","password":""}`
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(body))
		h.Create(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("username taken", func(t *testing.T) {
		repo := &stubRepo{
			findByUserFn: func(ctx context.Context, username string) (*User, error) {
				return nil, ErrNotFound
			},
			createFn: func(ctx context.Context, u *User) error {
				return ErrUsernameTaken
			},
		}
		h := newHandler(repo)

		body := `{"username":"taken","name":"T","email":"t@x.com","password":"pass1234"}`
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(body))
		h.Create(rr, r)

		if rr.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
		}
	})

	t.Run("email taken", func(t *testing.T) {
		repo := &stubRepo{
			findByUserFn: func(ctx context.Context, username string) (*User, error) {
				return nil, ErrNotFound
			},
			createFn: func(ctx context.Context, u *User) error {
				return ErrEmailTaken
			},
		}
		h := newHandler(repo)

		body := `{"username":"fresh","name":"F","email":"dup@x.com","password":"pass1234"}`
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(body))
		h.Create(rr, r)

		if rr.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		repo := &stubRepo{
			findByUserFn: func(ctx context.Context, username string) (*User, error) {
				return nil, ErrNotFound
			},
			createFn: func(ctx context.Context, u *User) error {
				return errors.New("boom")
			},
		}
		h := newHandler(repo)

		body := `{"username":"x","name":"X","email":"x@x.com","password":"pass1234"}`
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(body))
		h.Create(rr, r)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})
}

// --- ByID ---

func TestHandler_ByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			findByIDFn: func(ctx context.Context, id string) (*User, error) {
				return &User{ID: id, Name: "Alice"}, nil
			},
		}
		h := newHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/u1", nil)
		r.SetPathValue("id", "u1")
		h.ByID(rr, r)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var got User
		decodeJSON(t, rr.Body, &got)
		if got.ID != "u1" {
			t.Fatalf("id = %q, want u1", got.ID)
		}
	})

	t.Run("empty id", func(t *testing.T) {
		h := newHandler(&stubRepo{})

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/", nil)
		// no SetPathValue → empty
		h.ByID(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &stubRepo{
			findByIDFn: func(ctx context.Context, id string) (*User, error) {
				return nil, ErrNotFound
			},
		}
		h := newHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/nope", nil)
		r.SetPathValue("id", "nope")
		h.ByID(rr, r)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		repo := &stubRepo{
			findByIDFn: func(ctx context.Context, id string) (*User, error) {
				return nil, errors.New("db down")
			},
		}
		h := newHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/u1", nil)
		r.SetPathValue("id", "u1")
		h.ByID(rr, r)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})
}

// --- ListOrg ---

func TestHandler_ListOrg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			listByOrgFn: func(ctx context.Context, orgID string) ([]*OrgUser, error) {
				return []*OrgUser{{User: User{ID: "u1"}, Roles: []Role{{ID: "r1", Name: "admin"}}}}, nil
			},
		}
		h := newHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/acme/users", nil)
		r = withOrgID(r, "org-1")
		h.ListOrg(rr, r)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var got []OrgUser
		decodeJSON(t, rr.Body, &got)
		if len(got) != 1 || got[0].ID != "u1" {
			t.Fatalf("body = %+v", got)
		}
	})

	t.Run("missing org context", func(t *testing.T) {
		h := newHandler(&stubRepo{})

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/acme/users", nil)
		// no withOrgID
		h.ListOrg(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &stubRepo{
			listByOrgFn: func(ctx context.Context, orgID string) ([]*OrgUser, error) {
				return nil, errors.New("db down")
			},
		}
		h := newHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/acme/users", nil)
		r = withOrgID(r, "org-1")
		h.ListOrg(rr, r)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})
}

// --- OrgByID ---

func TestHandler_OrgByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			findByIDInOrgFn: func(ctx context.Context, orgID, id string) (*OrgUser, error) {
				return &OrgUser{User: User{ID: id}, Roles: []Role{{ID: "r1", Name: "cashier"}}}, nil
			},
		}
		h := newHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/acme/users/u1", nil)
		r = withOrgID(r, "org-1")
		r.SetPathValue("id", "u1")
		h.OrgByID(rr, r)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var got OrgUser
		decodeJSON(t, rr.Body, &got)
		if got.ID != "u1" {
			t.Fatalf("id = %q, want u1", got.ID)
		}
	})

	t.Run("missing org context", func(t *testing.T) {
		h := newHandler(&stubRepo{})

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/acme/users/u1", nil)
		r.SetPathValue("id", "u1")
		h.OrgByID(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("empty id", func(t *testing.T) {
		h := newHandler(&stubRepo{})

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/acme/users/", nil)
		r = withOrgID(r, "org-1")
		// no SetPathValue
		h.OrgByID(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &stubRepo{
			findByIDInOrgFn: func(ctx context.Context, orgID, id string) (*OrgUser, error) {
				return nil, ErrNotFound
			},
		}
		h := newHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/acme/users/nope", nil)
		r = withOrgID(r, "org-1")
		r.SetPathValue("id", "nope")
		h.OrgByID(rr, r)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		repo := &stubRepo{
			findByIDInOrgFn: func(ctx context.Context, orgID, id string) (*OrgUser, error) {
				return nil, errors.New("db down")
			},
		}
		h := newHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/acme/users/u1", nil)
		r = withOrgID(r, "org-1")
		r.SetPathValue("id", "u1")
		h.OrgByID(rr, r)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})
}
