package rbac

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- stubs ---

type stubPermRepo struct {
	createFn     func(ctx context.Context, p *Permission) error
	findByIDFn   func(ctx context.Context, id string) (*Permission, error)
	findByNameFn func(ctx context.Context, name string) (*Permission, error)
	listFn       func(ctx context.Context) ([]*Permission, error)
}

func (s *stubPermRepo) Create(ctx context.Context, p *Permission) error {
	if s.createFn != nil {
		return s.createFn(ctx, p)
	}
	return nil
}
func (s *stubPermRepo) FindByID(ctx context.Context, id string) (*Permission, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, id)
	}
	return nil, ErrPermissionNotFound
}
func (s *stubPermRepo) FindByName(ctx context.Context, name string) (*Permission, error) {
	if s.findByNameFn != nil {
		return s.findByNameFn(ctx, name)
	}
	return nil, ErrPermissionNotFound
}
func (s *stubPermRepo) List(ctx context.Context) ([]*Permission, error) {
	if s.listFn != nil {
		return s.listFn(ctx)
	}
	return nil, nil
}

type stubRoleRepo struct {
	createFn           func(ctx context.Context, r *Role) error
	findByIDFn         func(ctx context.Context, id, orgID string) (*Role, error)
	findByNameFn       func(ctx context.Context, name, orgID string) (*Role, error)
	listFn             func(ctx context.Context, orgID string) ([]*Role, error)
	assignPermissionFn func(ctx context.Context, roleID, permissionID string) error
	getPermsByRoleIDFn func(ctx context.Context, roleID string) ([]Permission, error)
	assignRoleToUserFn func(ctx context.Context, orgID, userID, roleID string) error
	getRolesByUserIDFn func(ctx context.Context, userID string) ([]Role, error)
	getPermsByUserIDFn func(ctx context.Context, orgID, userID string) ([]Permission, error)
}

func (s *stubRoleRepo) Create(ctx context.Context, r *Role) error {
	if s.createFn != nil {
		return s.createFn(ctx, r)
	}
	return nil
}
func (s *stubRoleRepo) FindByID(ctx context.Context, id, orgID string) (*Role, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, id, orgID)
	}
	return nil, ErrRoleNotFound
}
func (s *stubRoleRepo) FindByName(ctx context.Context, name, orgID string) (*Role, error) {
	if s.findByNameFn != nil {
		return s.findByNameFn(ctx, name, orgID)
	}
	return nil, ErrRoleNotFound
}
func (s *stubRoleRepo) List(ctx context.Context, orgID string) ([]*Role, error) {
	if s.listFn != nil {
		return s.listFn(ctx, orgID)
	}
	return nil, nil
}
func (s *stubRoleRepo) AssignPermission(ctx context.Context, roleID, permissionID string) error {
	if s.assignPermissionFn != nil {
		return s.assignPermissionFn(ctx, roleID, permissionID)
	}
	return nil
}
func (s *stubRoleRepo) GetPermissionsByRoleID(ctx context.Context, roleID string) ([]Permission, error) {
	if s.getPermsByRoleIDFn != nil {
		return s.getPermsByRoleIDFn(ctx, roleID)
	}
	return nil, nil
}
func (s *stubRoleRepo) AssignRoleToUser(ctx context.Context, orgID, userID, roleID string) error {
	if s.assignRoleToUserFn != nil {
		return s.assignRoleToUserFn(ctx, orgID, userID, roleID)
	}
	return nil
}
func (s *stubRoleRepo) GetRolesByUserID(ctx context.Context, userID string) ([]Role, error) {
	if s.getRolesByUserIDFn != nil {
		return s.getRolesByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (s *stubRoleRepo) GetPermissionsByUserID(ctx context.Context, orgID, userID string) ([]Permission, error) {
	if s.getPermsByUserIDFn != nil {
		return s.getPermsByUserIDFn(ctx, orgID, userID)
	}
	return nil, nil
}

// --- helpers ---

func newTestHandler(pr *stubPermRepo, rr *stubRoleRepo) *Handler {
	svc := NewService(pr, rr)
	return NewHandler(svc)
}

func jsonBody(v any) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

var fixedTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// --- CreatePermission ---

func TestCreatePermission_Success(t *testing.T) {
	pr := &stubPermRepo{
		createFn: func(_ context.Context, p *Permission) error {
			p.ID = "perm-1"
			p.CreatedAt = fixedTime
			return nil
		},
	}
	h := newTestHandler(pr, &stubRoleRepo{})

	body := jsonBody(CreatePermissionRequest{Name: "read", Description: "read access"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/permissions", body)
	w := httptest.NewRecorder()
	h.CreatePermission(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d", w.Code)
	}
	var resp PermissionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != "perm-1" {
		t.Errorf("want id perm-1, got %s", resp.ID)
	}
	if resp.Name != "read" {
		t.Errorf("want name read, got %s", resp.Name)
	}
}

func TestCreatePermission_InvalidJSON(t *testing.T) {
	h := newTestHandler(&stubPermRepo{}, &stubRoleRepo{})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/permissions", bytes.NewBufferString("{bad"))
	w := httptest.NewRecorder()
	h.CreatePermission(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreatePermission_EmptyName(t *testing.T) {
	h := newTestHandler(&stubPermRepo{}, &stubRoleRepo{})
	body := jsonBody(CreatePermissionRequest{Name: "", Description: "x"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/permissions", body)
	w := httptest.NewRecorder()
	h.CreatePermission(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreatePermission_NameTaken(t *testing.T) {
	pr := &stubPermRepo{
		createFn: func(_ context.Context, _ *Permission) error {
			return ErrPermissionNameTaken
		},
	}
	h := newTestHandler(pr, &stubRoleRepo{})

	body := jsonBody(CreatePermissionRequest{Name: "read", Description: "x"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/permissions", body)
	w := httptest.NewRecorder()
	h.CreatePermission(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

// --- ListPermissions ---

func TestListPermissions_Success(t *testing.T) {
	pr := &stubPermRepo{
		listFn: func(_ context.Context) ([]*Permission, error) {
			return []*Permission{
				{ID: "p1", Name: "read", CreatedAt: fixedTime},
				{ID: "p2", Name: "write", CreatedAt: fixedTime},
			}, nil
		},
	}
	h := newTestHandler(pr, &stubRoleRepo{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/permissions", nil)
	w := httptest.NewRecorder()
	h.ListPermissions(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp []PermissionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Fatalf("want 2, got %d", len(resp))
	}
}

func TestListPermissions_Error(t *testing.T) {
	pr := &stubPermRepo{
		listFn: func(_ context.Context) ([]*Permission, error) {
			return nil, errors.New("db down")
		},
	}
	h := newTestHandler(pr, &stubRoleRepo{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/permissions", nil)
	w := httptest.NewRecorder()
	h.ListPermissions(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

// --- CreateRole (global) ---

func TestCreateRole_Success(t *testing.T) {
	rr := &stubRoleRepo{
		createFn: func(_ context.Context, r *Role) error {
			r.ID = "role-1"
			r.CreatedAt = fixedTime
			return nil
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	body := jsonBody(CreateRoleRequest{Name: "admin", Description: "admin role"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/roles", body)
	w := httptest.NewRecorder()
	h.CreateRole(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d", w.Code)
	}
	var resp RoleResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != "role-1" {
		t.Errorf("want role-1, got %s", resp.ID)
	}
	if resp.Permissions == nil {
		t.Error("permissions should be non-nil empty slice")
	}
}

func TestCreateRole_InvalidJSON(t *testing.T) {
	h := newTestHandler(&stubPermRepo{}, &stubRoleRepo{})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	h.CreateRole(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreateRole_EmptyName(t *testing.T) {
	h := newTestHandler(&stubPermRepo{}, &stubRoleRepo{})
	body := jsonBody(CreateRoleRequest{Name: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/roles", body)
	w := httptest.NewRecorder()
	h.CreateRole(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreateRole_NameTaken(t *testing.T) {
	rr := &stubRoleRepo{
		createFn: func(_ context.Context, _ *Role) error {
			return ErrRoleNameTaken
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	body := jsonBody(CreateRoleRequest{Name: "admin"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/roles", body)
	w := httptest.NewRecorder()
	h.CreateRole(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

// --- CreateOrgRole ---
// Org-scoped handlers call middleware.OrgIDFromContext which uses unexported contextKey.
// Since we're in package rbac, we call the private method directly with orgID.

func TestCreateOrgRole_Success(t *testing.T) {
	rr := &stubRoleRepo{
		createFn: func(_ context.Context, r *Role) error {
			r.ID = "org-role-1"
			r.CreatedAt = fixedTime
			return nil
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	body := jsonBody(CreateRoleRequest{Name: "cashier", Description: "org cashier"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/org-1/roles", body)
	w := httptest.NewRecorder()
	h.createRole(w, r, "org-1")

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d", w.Code)
	}
	var resp RoleResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != "org-role-1" {
		t.Errorf("want org-role-1, got %s", resp.ID)
	}
}

func TestCreateOrgRole_NameTaken(t *testing.T) {
	rr := &stubRoleRepo{
		createFn: func(_ context.Context, _ *Role) error {
			return ErrRoleNameTaken
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	body := jsonBody(CreateRoleRequest{Name: "cashier"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/org-1/roles", body)
	w := httptest.NewRecorder()
	h.createRole(w, r, "org-1")

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

// --- ListRoles (global) ---

func TestListRoles_Success(t *testing.T) {
	rr := &stubRoleRepo{
		listFn: func(_ context.Context, orgID string) ([]*Role, error) {
			if orgID != "" {
				t.Errorf("global list should have empty orgID, got %q", orgID)
			}
			return []*Role{
				{ID: "r1", Name: "admin", Permissions: []Permission{}, CreatedAt: fixedTime},
			}, nil
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/roles", nil)
	w := httptest.NewRecorder()
	h.ListRoles(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp []RoleResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Fatalf("want 1 role, got %d", len(resp))
	}
}

func TestListRoles_Error(t *testing.T) {
	rr := &stubRoleRepo{
		listFn: func(_ context.Context, _ string) ([]*Role, error) {
			return nil, errors.New("db")
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/roles", nil)
	w := httptest.NewRecorder()
	h.ListRoles(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

// --- ListOrgRoles ---

func TestListOrgRoles_Success(t *testing.T) {
	rr := &stubRoleRepo{
		listFn: func(_ context.Context, orgID string) ([]*Role, error) {
			if orgID != "org-1" {
				t.Errorf("want orgID org-1, got %q", orgID)
			}
			return []*Role{
				{ID: "r1", Name: "cashier", OrgID: "org-1", Permissions: []Permission{}, CreatedAt: fixedTime},
			}, nil
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/roles", nil)
	w := httptest.NewRecorder()
	h.listRoles(w, r, "org-1")

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

// --- GetRoleByID (global) ---

func TestGetRoleByID_Success(t *testing.T) {
	rr := &stubRoleRepo{
		findByIDFn: func(_ context.Context, id, orgID string) (*Role, error) {
			return &Role{ID: id, Name: "admin", Permissions: []Permission{}, CreatedAt: fixedTime}, nil
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/roles/role-1", nil)
	r.SetPathValue("id", "role-1")
	w := httptest.NewRecorder()
	h.GetRoleByID(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp RoleResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != "role-1" {
		t.Errorf("want role-1, got %s", resp.ID)
	}
}

func TestGetRoleByID_NotFound(t *testing.T) {
	rr := &stubRoleRepo{
		findByIDFn: func(_ context.Context, _, _ string) (*Role, error) {
			return nil, ErrRoleNotFound
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/roles/nope", nil)
	r.SetPathValue("id", "nope")
	w := httptest.NewRecorder()
	h.GetRoleByID(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestGetRoleByID_EmptyID(t *testing.T) {
	h := newTestHandler(&stubPermRepo{}, &stubRoleRepo{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/roles/", nil)
	w := httptest.NewRecorder()
	h.GetRoleByID(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

// --- GetOrgRoleByID ---

func TestGetOrgRoleByID_Success(t *testing.T) {
	rr := &stubRoleRepo{
		findByIDFn: func(_ context.Context, id, orgID string) (*Role, error) {
			if orgID != "org-1" {
				t.Errorf("want orgID org-1, got %q", orgID)
			}
			return &Role{ID: id, Name: "cashier", OrgID: orgID, Permissions: []Permission{}, CreatedAt: fixedTime}, nil
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/roles/role-1", nil)
	r.SetPathValue("id", "role-1")
	w := httptest.NewRecorder()
	h.getRoleByID(w, r, "org-1")

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestGetOrgRoleByID_NotFound(t *testing.T) {
	rr := &stubRoleRepo{
		findByIDFn: func(_ context.Context, _, _ string) (*Role, error) {
			return nil, ErrRoleNotFound
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/roles/role-1", nil)
	r.SetPathValue("id", "role-1")
	w := httptest.NewRecorder()
	h.getRoleByID(w, r, "org-1")

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

// --- AssignPermission (global) ---

func TestAssignPermission_Success(t *testing.T) {
	rr := &stubRoleRepo{
		findByIDFn: func(_ context.Context, _, _ string) (*Role, error) {
			return &Role{ID: "role-1"}, nil
		},
		assignPermissionFn: func(_ context.Context, _, _ string) error {
			return nil
		},
	}
	pr := &stubPermRepo{
		findByIDFn: func(_ context.Context, id string) (*Permission, error) {
			return &Permission{ID: id}, nil
		},
	}
	h := newTestHandler(pr, rr)

	body := jsonBody(AssignPermissionRequest{PermissionID: "perm-1"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/roles/role-1/permissions", body)
	r.SetPathValue("id", "role-1")
	w := httptest.NewRecorder()
	h.AssignPermission(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestAssignPermission_InvalidJSON(t *testing.T) {
	h := newTestHandler(&stubPermRepo{}, &stubRoleRepo{})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/roles/role-1/permissions", bytes.NewBufferString("{bad"))
	r.SetPathValue("id", "role-1")
	w := httptest.NewRecorder()
	h.AssignPermission(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestAssignPermission_EmptyPermissionID(t *testing.T) {
	h := newTestHandler(&stubPermRepo{}, &stubRoleRepo{})
	body := jsonBody(AssignPermissionRequest{PermissionID: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/roles/role-1/permissions", body)
	r.SetPathValue("id", "role-1")
	w := httptest.NewRecorder()
	h.AssignPermission(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestAssignPermission_RoleNotFound(t *testing.T) {
	rr := &stubRoleRepo{
		findByIDFn: func(_ context.Context, _, _ string) (*Role, error) {
			return nil, ErrRoleNotFound
		},
	}
	pr := &stubPermRepo{
		findByIDFn: func(_ context.Context, id string) (*Permission, error) {
			return &Permission{ID: id}, nil
		},
	}
	h := newTestHandler(pr, rr)

	body := jsonBody(AssignPermissionRequest{PermissionID: "perm-1"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/roles/nope/permissions", body)
	r.SetPathValue("id", "nope")
	w := httptest.NewRecorder()
	h.AssignPermission(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestAssignPermission_PermissionNotFound(t *testing.T) {
	rr := &stubRoleRepo{
		findByIDFn: func(_ context.Context, _, _ string) (*Role, error) {
			return &Role{ID: "role-1"}, nil
		},
	}
	pr := &stubPermRepo{
		findByIDFn: func(_ context.Context, _ string) (*Permission, error) {
			return nil, ErrPermissionNotFound
		},
	}
	h := newTestHandler(pr, rr)

	body := jsonBody(AssignPermissionRequest{PermissionID: "perm-x"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/roles/role-1/permissions", body)
	r.SetPathValue("id", "role-1")
	w := httptest.NewRecorder()
	h.AssignPermission(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestAssignPermission_EmptyRoleID(t *testing.T) {
	h := newTestHandler(&stubPermRepo{}, &stubRoleRepo{})
	body := jsonBody(AssignPermissionRequest{PermissionID: "perm-1"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/roles//permissions", body)
	w := httptest.NewRecorder()
	h.AssignPermission(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

// --- AssignOrgPermission ---

func TestAssignOrgPermission_Success(t *testing.T) {
	rr := &stubRoleRepo{
		findByIDFn: func(_ context.Context, _, orgID string) (*Role, error) {
			return &Role{ID: "role-1", OrgID: orgID}, nil
		},
		assignPermissionFn: func(_ context.Context, _, _ string) error {
			return nil
		},
	}
	pr := &stubPermRepo{
		findByIDFn: func(_ context.Context, id string) (*Permission, error) {
			return &Permission{ID: id}, nil
		},
	}
	h := newTestHandler(pr, rr)

	body := jsonBody(AssignPermissionRequest{PermissionID: "perm-1"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/org-1/roles/role-1/permissions", body)
	r.SetPathValue("id", "role-1")
	w := httptest.NewRecorder()
	h.assignPermission(w, r, "org-1")

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestAssignOrgPermission_RoleNotFound(t *testing.T) {
	rr := &stubRoleRepo{
		findByIDFn: func(_ context.Context, _, _ string) (*Role, error) {
			return nil, ErrRoleNotFound
		},
	}
	pr := &stubPermRepo{
		findByIDFn: func(_ context.Context, id string) (*Permission, error) {
			return &Permission{ID: id}, nil
		},
	}
	h := newTestHandler(pr, rr)

	body := jsonBody(AssignPermissionRequest{PermissionID: "perm-1"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/org-1/roles/nope/permissions", body)
	r.SetPathValue("id", "nope")
	w := httptest.NewRecorder()
	h.assignPermission(w, r, "org-1")

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

// --- AssignRoleToUser (global) ---

func TestAssignRoleToUser_Success(t *testing.T) {
	rr := &stubRoleRepo{
		findByIDFn: func(_ context.Context, _, _ string) (*Role, error) {
			return &Role{ID: "role-1"}, nil
		},
		assignRoleToUserFn: func(_ context.Context, _, _, _ string) error {
			return nil
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	body := jsonBody(AssignRoleRequest{RoleID: "role-1"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/users/user-1/roles", body)
	r.SetPathValue("id", "user-1")
	w := httptest.NewRecorder()
	h.AssignRoleToUser(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestAssignRoleToUser_InvalidJSON(t *testing.T) {
	h := newTestHandler(&stubPermRepo{}, &stubRoleRepo{})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/users/user-1/roles", bytes.NewBufferString("xxx"))
	r.SetPathValue("id", "user-1")
	w := httptest.NewRecorder()
	h.AssignRoleToUser(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestAssignRoleToUser_EmptyRoleID(t *testing.T) {
	h := newTestHandler(&stubPermRepo{}, &stubRoleRepo{})
	body := jsonBody(AssignRoleRequest{RoleID: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/users/user-1/roles", body)
	r.SetPathValue("id", "user-1")
	w := httptest.NewRecorder()
	h.AssignRoleToUser(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestAssignRoleToUser_EmptyUserID(t *testing.T) {
	h := newTestHandler(&stubPermRepo{}, &stubRoleRepo{})
	body := jsonBody(AssignRoleRequest{RoleID: "role-1"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/users//roles", body)
	w := httptest.NewRecorder()
	h.AssignRoleToUser(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestAssignRoleToUser_RoleNotFound(t *testing.T) {
	rr := &stubRoleRepo{
		findByIDFn: func(_ context.Context, _, _ string) (*Role, error) {
			return nil, ErrRoleNotFound
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	body := jsonBody(AssignRoleRequest{RoleID: "nope"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/users/user-1/roles", body)
	r.SetPathValue("id", "user-1")
	w := httptest.NewRecorder()
	h.AssignRoleToUser(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

// --- AssignOrgRoleToUser ---

func TestAssignOrgRoleToUser_Success(t *testing.T) {
	rr := &stubRoleRepo{
		findByIDFn: func(_ context.Context, _, orgID string) (*Role, error) {
			return &Role{ID: "role-1", OrgID: orgID}, nil
		},
		assignRoleToUserFn: func(_ context.Context, _, _, _ string) error {
			return nil
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	body := jsonBody(AssignRoleRequest{RoleID: "role-1"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/org-1/users/user-1/roles", body)
	r.SetPathValue("id", "user-1")
	w := httptest.NewRecorder()
	h.assignRoleToUser(w, r, "org-1")

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestAssignOrgRoleToUser_RoleNotFound(t *testing.T) {
	rr := &stubRoleRepo{
		findByIDFn: func(_ context.Context, _, _ string) (*Role, error) {
			return nil, ErrRoleNotFound
		},
	}
	h := newTestHandler(&stubPermRepo{}, rr)

	body := jsonBody(AssignRoleRequest{RoleID: "nope"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/org-1/users/user-1/roles", body)
	r.SetPathValue("id", "user-1")
	w := httptest.NewRecorder()
	h.assignRoleToUser(w, r, "org-1")

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}
