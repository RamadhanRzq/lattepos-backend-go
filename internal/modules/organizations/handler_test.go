package organizations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/rbac"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

func withClaims(r *http.Request, claims *appjwt.Claims) *http.Request {
	ctx := middleware.NewContextWithClaims(r.Context(), claims)
	return r.WithContext(ctx)
}

func withOrgID(r *http.Request, orgID string) *http.Request {
	ctx := middleware.NewContextWithOrgID(r.Context(), orgID)
	return r.WithContext(ctx)
}

// --- stub repositories ---

type stubRepo struct {
	createFn       func(ctx context.Context, org *Organization) error
	findByIDFn     func(ctx context.Context, id string) (*Organization, error)
	findBySlugFn   func(ctx context.Context, slug string) (*Organization, error)
	listFn         func(ctx context.Context) ([]Organization, error)
	addMemberFn    func(ctx context.Context, orgID, userID string) error
	removeMemberFn func(ctx context.Context, orgID, userID string) error
	isMemberFn     func(ctx context.Context, orgID, userID string) (bool, error)
	listMembersFn  func(ctx context.Context, orgID string) ([]users.User, error)
	listByUserIDFn func(ctx context.Context, userID string) ([]Organization, error)
}

func (s *stubRepo) Create(ctx context.Context, org *Organization) error {
	return s.createFn(ctx, org)
}
func (s *stubRepo) FindByID(ctx context.Context, id string) (*Organization, error) {
	return s.findByIDFn(ctx, id)
}
func (s *stubRepo) FindBySlug(ctx context.Context, slug string) (*Organization, error) {
	return s.findBySlugFn(ctx, slug)
}
func (s *stubRepo) List(ctx context.Context) ([]Organization, error) {
	return s.listFn(ctx)
}
func (s *stubRepo) AddMember(ctx context.Context, orgID, userID string) error {
	return s.addMemberFn(ctx, orgID, userID)
}
func (s *stubRepo) RemoveMember(ctx context.Context, orgID, userID string) error {
	return s.removeMemberFn(ctx, orgID, userID)
}
func (s *stubRepo) IsMember(ctx context.Context, orgID, userID string) (bool, error) {
	return s.isMemberFn(ctx, orgID, userID)
}
func (s *stubRepo) ListMembers(ctx context.Context, orgID string) ([]users.User, error) {
	return s.listMembersFn(ctx, orgID)
}
func (s *stubRepo) ListByUserID(ctx context.Context, userID string) ([]Organization, error) {
	return s.listByUserIDFn(ctx, userID)
}

type stubRoleRepo struct{}

func (s *stubRoleRepo) Create(ctx context.Context, r *rbac.Role) error            { return nil }
func (s *stubRoleRepo) FindByID(ctx context.Context, id, orgID string) (*rbac.Role, error) {
	return nil, nil
}
func (s *stubRoleRepo) FindByName(ctx context.Context, name, orgID string) (*rbac.Role, error) {
	return nil, errors.New("not found")
}
func (s *stubRoleRepo) List(ctx context.Context, orgID string) ([]*rbac.Role, error) {
	return nil, nil
}
func (s *stubRoleRepo) AssignPermission(ctx context.Context, roleID, permissionID string) error {
	return nil
}
func (s *stubRoleRepo) GetPermissionsByRoleID(ctx context.Context, roleID string) ([]rbac.Permission, error) {
	return nil, nil
}
func (s *stubRoleRepo) AssignRoleToUser(ctx context.Context, orgID, userID, roleID string) error {
	return nil
}
func (s *stubRoleRepo) GetRolesByUserID(ctx context.Context, userID string) ([]rbac.Role, error) {
	return nil, nil
}
func (s *stubRoleRepo) GetPermissionsByUserID(ctx context.Context, orgID, userID string) ([]rbac.Permission, error) {
	return nil, nil
}

// --- helpers ---

func newTestHandler(repo *stubRepo) *Handler {
	svc := NewService(repo, &stubRoleRepo{})
	mgr := appjwt.NewManager("test-secret-key-for-testing", time.Hour)
	return NewHandler(svc, mgr)
}

func decodeJSON(t *testing.T, body *bytes.Buffer, v any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return bytes.NewBuffer(b)
}

var testClaims = &appjwt.Claims{
	UserID:   "user-1",
	Username: "testuser",
	Email:    "test@example.com",
	Name:     "Test User",
}

// --- CreateOrg ---

func TestHandler_CreateOrg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			createFn: func(ctx context.Context, org *Organization) error {
				org.ID = "org-1"
				return nil
			},
			addMemberFn: func(ctx context.Context, orgID, userID string) error {
				return nil
			},
		}
		h := newTestHandler(repo)

		body := jsonBody(t, CreateRequest{Name: "My Org", Slug: "my-org"})
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", body)
		r = withClaims(r, testClaims)
		h.CreateOrg(rr, r)

		if rr.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusCreated)
		}
		var got Response
		decodeJSON(t, rr.Body, &got)
		if got.ID != "org-1" {
			t.Errorf("id = %q, want %q", got.ID, "org-1")
		}
		if got.Slug != "my-org" {
			t.Errorf("slug = %q, want %q", got.Slug, "my-org")
		}
	})

	t.Run("no claims returns 401", func(t *testing.T) {
		h := newTestHandler(&stubRepo{})
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", nil)
		h.CreateOrg(rr, r)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
		}
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		h := newTestHandler(&stubRepo{})
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBufferString("{bad"))
		r = withClaims(r, testClaims)
		h.CreateOrg(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid slug returns 400", func(t *testing.T) {
		h := newTestHandler(&stubRepo{})

		body := jsonBody(t, CreateRequest{Name: "My Org", Slug: "AB"})
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", body)
		r = withClaims(r, testClaims)
		h.CreateOrg(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("slug taken returns 409", func(t *testing.T) {
		repo := &stubRepo{
			createFn: func(ctx context.Context, org *Organization) error {
				return ErrSlugTaken
			},
		}
		h := newTestHandler(repo)

		body := jsonBody(t, CreateRequest{Name: "My Org", Slug: "taken-slug"})
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", body)
		r = withClaims(r, testClaims)
		h.CreateOrg(rr, r)

		if rr.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
		}
	})

	t.Run("empty name returns 400", func(t *testing.T) {
		h := newTestHandler(&stubRepo{})

		body := jsonBody(t, CreateRequest{Name: "", Slug: "valid-slug"})
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", body)
		r = withClaims(r, testClaims)
		h.CreateOrg(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})
}

// --- ListOrgs ---

func TestHandler_ListOrgs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			listFn: func(ctx context.Context) ([]Organization, error) {
				return []Organization{
					{ID: "o1", Name: "Org1", Slug: "org-1"},
					{ID: "o2", Name: "Org2", Slug: "org-2"},
				}, nil
			},
		}
		h := newTestHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/organizations", nil)
		h.ListOrgs(rr, r)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var got []Response
		decodeJSON(t, rr.Body, &got)
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		repo := &stubRepo{
			listFn: func(ctx context.Context) ([]Organization, error) {
				return nil, errors.New("db down")
			},
		}
		h := newTestHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/organizations", nil)
		h.ListOrgs(rr, r)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})
}

// --- GetOrgBySlug ---

func TestHandler_GetOrgBySlug(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			findBySlugFn: func(ctx context.Context, slug string) (*Organization, error) {
				return &Organization{ID: "o1", Name: "Org1", Slug: slug}, nil
			},
		}
		h := newTestHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/my-org", nil)
		r.SetPathValue("slug", "my-org")
		h.GetOrgBySlug(rr, r)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var got Response
		decodeJSON(t, rr.Body, &got)
		if got.Slug != "my-org" {
			t.Errorf("slug = %q, want %q", got.Slug, "my-org")
		}
	})

	t.Run("empty slug returns 400", func(t *testing.T) {
		h := newTestHandler(&stubRepo{})

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/", nil)
		h.GetOrgBySlug(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := &stubRepo{
			findBySlugFn: func(ctx context.Context, slug string) (*Organization, error) {
				return nil, ErrNotFound
			},
		}
		h := newTestHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/nope", nil)
		r.SetPathValue("slug", "nope")
		h.GetOrgBySlug(rr, r)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		repo := &stubRepo{
			findBySlugFn: func(ctx context.Context, slug string) (*Organization, error) {
				return nil, errors.New("db error")
			},
		}
		h := newTestHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/fail", nil)
		r.SetPathValue("slug", "fail")
		h.GetOrgBySlug(rr, r)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})
}

// --- SelectOrg ---

func TestHandler_SelectOrg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			findBySlugFn: func(ctx context.Context, slug string) (*Organization, error) {
				return &Organization{ID: "o1", Name: "Org1", Slug: slug}, nil
			},
			isMemberFn: func(ctx context.Context, orgID, userID string) (bool, error) {
				return true, nil
			},
		}
		h := newTestHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/org/my-org/auth/select", nil)
		r.SetPathValue("slug", "my-org")
		r = withClaims(r, testClaims)
		h.SelectOrg(rr, r)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var got SelectResponse
		decodeJSON(t, rr.Body, &got)
		if got.Token == "" {
			t.Fatal("token should not be empty")
		}
		if got.TokenType != "Bearer" {
			t.Errorf("token_type = %q, want %q", got.TokenType, "Bearer")
		}
		if got.Org.ID != "o1" {
			t.Errorf("org.id = %q, want %q", got.Org.ID, "o1")
		}
	})

	t.Run("no claims returns 401", func(t *testing.T) {
		h := newTestHandler(&stubRepo{})

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/org/my-org/auth/select", nil)
		r.SetPathValue("slug", "my-org")
		h.SelectOrg(rr, r)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
		}
	})

	t.Run("empty slug returns 400", func(t *testing.T) {
		h := newTestHandler(&stubRepo{})

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/org//auth/select", nil)
		r = withClaims(r, testClaims)
		h.SelectOrg(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("org not found returns 404", func(t *testing.T) {
		repo := &stubRepo{
			findBySlugFn: func(ctx context.Context, slug string) (*Organization, error) {
				return nil, ErrNotFound
			},
		}
		h := newTestHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/org/nope/auth/select", nil)
		r.SetPathValue("slug", "nope")
		r = withClaims(r, testClaims)
		h.SelectOrg(rr, r)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("not member returns 403", func(t *testing.T) {
		repo := &stubRepo{
			findBySlugFn: func(ctx context.Context, slug string) (*Organization, error) {
				return &Organization{ID: "o1", Name: "Org1", Slug: slug}, nil
			},
			isMemberFn: func(ctx context.Context, orgID, userID string) (bool, error) {
				return false, nil
			},
		}
		h := newTestHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/org/my-org/auth/select", nil)
		r.SetPathValue("slug", "my-org")
		r = withClaims(r, testClaims)
		h.SelectOrg(rr, r)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
		}
	})
}

// --- ListOrgMembers ---

func TestHandler_ListOrgMembers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			listMembersFn: func(ctx context.Context, orgID string) ([]users.User, error) {
				return []users.User{{ID: "u1", Name: "Alice"}}, nil
			},
		}
		h := newTestHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/my-org/members", nil)
		r = withOrgID(r, "org-1")
		h.ListOrgMembers(rr, r)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var got []users.User
		decodeJSON(t, rr.Body, &got)
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].ID != "u1" {
			t.Errorf("user.id = %q, want %q", got[0].ID, "u1")
		}
	})

	t.Run("missing org context returns 400", func(t *testing.T) {
		h := newTestHandler(&stubRepo{})

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/my-org/members", nil)
		h.ListOrgMembers(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		repo := &stubRepo{
			listMembersFn: func(ctx context.Context, orgID string) ([]users.User, error) {
				return nil, errors.New("db error")
			},
		}
		h := newTestHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/org/my-org/members", nil)
		r = withOrgID(r, "org-1")
		h.ListOrgMembers(rr, r)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})
}

// --- AddOrgMember ---

func TestHandler_AddOrgMember(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			isMemberFn: func(ctx context.Context, orgID, userID string) (bool, error) {
				return false, nil
			},
			addMemberFn: func(ctx context.Context, orgID, userID string) error {
				return nil
			},
		}
		h := newTestHandler(repo)

		body := jsonBody(t, AddMemberRequest{UserID: "u2"})
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/org/my-org/members", body)
		r = withOrgID(r, "org-1")
		h.AddOrgMember(rr, r)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var got map[string]string
		decodeJSON(t, rr.Body, &got)
		if got["message"] == "" {
			t.Error("expected message in response")
		}
	})

	t.Run("missing org context returns 400", func(t *testing.T) {
		h := newTestHandler(&stubRepo{})

		body := jsonBody(t, AddMemberRequest{UserID: "u2"})
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/org/my-org/members", body)
		h.AddOrgMember(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		h := newTestHandler(&stubRepo{})

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/org/my-org/members", bytes.NewBufferString("{bad"))
		r = withOrgID(r, "org-1")
		h.AddOrgMember(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("empty user_id returns 400", func(t *testing.T) {
		h := newTestHandler(&stubRepo{})

		body := jsonBody(t, AddMemberRequest{UserID: ""})
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/org/my-org/members", body)
		r = withOrgID(r, "org-1")
		h.AddOrgMember(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("already member returns 409", func(t *testing.T) {
		repo := &stubRepo{
			isMemberFn: func(ctx context.Context, orgID, userID string) (bool, error) {
				return true, nil
			},
		}
		h := newTestHandler(repo)

		body := jsonBody(t, AddMemberRequest{UserID: "u2"})
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/org/my-org/members", body)
		r = withOrgID(r, "org-1")
		h.AddOrgMember(rr, r)

		if rr.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
		}
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		repo := &stubRepo{
			isMemberFn: func(ctx context.Context, orgID, userID string) (bool, error) {
				return false, nil
			},
			addMemberFn: func(ctx context.Context, orgID, userID string) error {
				return errors.New("db error")
			},
		}
		h := newTestHandler(repo)

		body := jsonBody(t, AddMemberRequest{UserID: "u2"})
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/org/my-org/members", body)
		r = withOrgID(r, "org-1")
		h.AddOrgMember(rr, r)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})
}

// --- RemoveOrgMember ---

func TestHandler_RemoveOrgMember(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			isMemberFn: func(ctx context.Context, orgID, userID string) (bool, error) {
				return true, nil
			},
			removeMemberFn: func(ctx context.Context, orgID, userID string) error {
				return nil
			},
		}
		h := newTestHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/api/v1/org/my-org/members/u2", nil)
		r = withOrgID(r, "org-1")
		r.SetPathValue("uid", "u2")
		h.RemoveOrgMember(rr, r)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var got map[string]string
		decodeJSON(t, rr.Body, &got)
		if got["message"] == "" {
			t.Error("expected message in response")
		}
	})

	t.Run("missing org context returns 400", func(t *testing.T) {
		h := newTestHandler(&stubRepo{})

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/api/v1/org/my-org/members/u2", nil)
		r.SetPathValue("uid", "u2")
		h.RemoveOrgMember(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("empty uid returns 400", func(t *testing.T) {
		h := newTestHandler(&stubRepo{})

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/api/v1/org/my-org/members/", nil)
		r = withOrgID(r, "org-1")
		h.RemoveOrgMember(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("not member returns 404", func(t *testing.T) {
		repo := &stubRepo{
			isMemberFn: func(ctx context.Context, orgID, userID string) (bool, error) {
				return false, nil
			},
		}
		h := newTestHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/api/v1/org/my-org/members/u2", nil)
		r = withOrgID(r, "org-1")
		r.SetPathValue("uid", "u2")
		h.RemoveOrgMember(rr, r)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		repo := &stubRepo{
			isMemberFn: func(ctx context.Context, orgID, userID string) (bool, error) {
				return true, nil
			},
			removeMemberFn: func(ctx context.Context, orgID, userID string) error {
				return errors.New("db error")
			},
		}
		h := newTestHandler(repo)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/api/v1/org/my-org/members/u2", nil)
		r = withOrgID(r, "org-1")
		r.SetPathValue("uid", "u2")
		h.RemoveOrgMember(rr, r)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})
}
