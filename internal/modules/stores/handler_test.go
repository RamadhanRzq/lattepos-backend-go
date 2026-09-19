package stores

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
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
)

func withOrgID(r *http.Request, orgID string) *http.Request {
	ctx := middleware.NewContextWithOrgID(r.Context(), orgID)
	return r.WithContext(ctx)
}

// --- stubs ---

type mockRepo struct {
	store     *Store
	stores    []Store
	userList  []users.User
	assigned  bool
	findErr   error
	updateErr error
	setActErr error
	assignErr error
	removeErr error
	listErr   error
	createErr error
}

func (m *mockRepo) Create(_ context.Context, s *Store) error {
	if m.createErr != nil {
		return m.createErr
	}
	s.ID = "s-new"
	s.IsActive = true
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	return nil
}

func (m *mockRepo) FindByIDInOrg(_ context.Context, _, _ string) (*Store, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.store, nil
}

func (m *mockRepo) ListByOrg(_ context.Context, _ string) ([]Store, error) {
	return m.stores, m.listErr
}

func (m *mockRepo) Update(_ context.Context, _ *Store) error { return m.updateErr }

func (m *mockRepo) SetActive(_ context.Context, _, _ string, active bool) (*Store, error) {
	if m.setActErr != nil {
		return nil, m.setActErr
	}
	s := *m.store
	s.IsActive = active
	return &s, nil
}

func (m *mockRepo) AssignUser(_ context.Context, _, _ string) error { return m.assignErr }

func (m *mockRepo) IsAssigned(_ context.Context, _, _ string) (bool, error) {
	return m.assigned, nil
}

func (m *mockRepo) RemoveUser(_ context.Context, _, _ string) error { return m.removeErr }

func (m *mockRepo) ListUserStores(_ context.Context, _, _ string) ([]Store, error) {
	return m.stores, m.listErr
}

func (m *mockRepo) ListStoreUsers(_ context.Context, _, _ string) ([]users.User, error) {
	return m.userList, m.listErr
}

type mockOrgChecker struct {
	isMember bool
	err      error
}

func (m *mockOrgChecker) IsMember(_ context.Context, _, _ string) (bool, error) {
	return m.isMember, m.err
}

func findUserOK(_ context.Context, id string) (*users.User, error) {
	return &users.User{ID: id, Name: "Test"}, nil
}

func findUserFail(_ context.Context, _ string) (*users.User, error) {
	return nil, errors.New("not found")
}

func makeHandler(repo *mockRepo, orgs *mockOrgChecker, fu func(context.Context, string) (*users.User, error)) *Handler {
	return NewHandler(NewService(repo, orgs, fu))
}

func sampleStore() *Store {
	return &Store{
		ID:             "s-1",
		OrganizationID: "org-1",
		Name:           "Main",
		Code:           "MAIN",
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func jbody(v any) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

func assertCode(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("status: got %d, want %d", got, want)
	}
}

// ==================== Create ====================

func TestCreate_Success(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodPost, "/", jbody(CreateRequest{Name: "Shop", Code: "SH01"}))
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.Create(w, r)
	assertCode(t, w.Code, http.StatusCreated)

	var s Store
	json.NewDecoder(w.Body).Decode(&s)
	if s.ID == "" {
		t.Fatal("expected store ID")
	}
}

func TestCreate_MissingOrg(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodPost, "/", jbody(CreateRequest{Name: "X", Code: "X"}))
	w := httptest.NewRecorder()

	h.Create(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestCreate_InvalidJSON(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad"))
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.Create(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestCreate_EmptyNameCode(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodPost, "/", jbody(CreateRequest{}))
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.Create(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestCreate_InvalidCode(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodPost, "/", jbody(CreateRequest{Name: "Shop", Code: "bad code!"}))
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.Create(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestCreate_CodeTaken(t *testing.T) {
	repo := &mockRepo{createErr: ErrCodeTaken}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodPost, "/", jbody(CreateRequest{Name: "X", Code: "TAKEN"}))
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.Create(w, r)
	assertCode(t, w.Code, http.StatusConflict)
}

func TestCreate_InternalError(t *testing.T) {
	repo := &mockRepo{createErr: errors.New("boom")}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodPost, "/", jbody(CreateRequest{Name: "X", Code: "XX"}))
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.Create(w, r)
	assertCode(t, w.Code, http.StatusInternalServerError)
}

// ==================== List ====================

func TestList_Success(t *testing.T) {
	repo := &mockRepo{stores: []Store{{ID: "s-1"}, {ID: "s-2"}}}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.List(w, r)
	assertCode(t, w.Code, http.StatusOK)

	var list []Store
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 2 {
		t.Fatalf("want 2, got %d", len(list))
	}
}

func TestList_MissingOrg(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.List(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestList_InternalError(t *testing.T) {
	repo := &mockRepo{listErr: errors.New("db")}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.List(w, r)
	assertCode(t, w.Code, http.StatusInternalServerError)
}

// ==================== ByID ====================

func TestByID_Success(t *testing.T) {
	repo := &mockRepo{store: sampleStore()}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.ByID(w, r)
	assertCode(t, w.Code, http.StatusOK)
}

func TestByID_EmptyID(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.ByID(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestByID_NotFound(t *testing.T) {
	repo := &mockRepo{findErr: ErrNotFound}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-999")
	w := httptest.NewRecorder()

	h.ByID(w, r)
	assertCode(t, w.Code, http.StatusNotFound)
}

func TestByID_InternalError(t *testing.T) {
	repo := &mockRepo{findErr: errors.New("db")}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.ByID(w, r)
	assertCode(t, w.Code, http.StatusInternalServerError)
}

// ==================== Update ====================

func TestUpdate_Success(t *testing.T) {
	repo := &mockRepo{store: sampleStore()}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodPut, "/", jbody(UpdateRequest{Name: "New", Code: "NEW1"}))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.Update(w, r)
	assertCode(t, w.Code, http.StatusOK)
}

func TestUpdate_EmptyID(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodPut, "/", jbody(UpdateRequest{Name: "X", Code: "X"}))
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.Update(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestUpdate_InvalidJSON(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString("nope"))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.Update(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestUpdate_NotFound(t *testing.T) {
	repo := &mockRepo{updateErr: ErrNotFound}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodPut, "/", jbody(UpdateRequest{Name: "X", Code: "XX"}))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-999")
	w := httptest.NewRecorder()

	h.Update(w, r)
	assertCode(t, w.Code, http.StatusNotFound)
}

func TestUpdate_CodeTaken(t *testing.T) {
	repo := &mockRepo{updateErr: ErrCodeTaken}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodPut, "/", jbody(UpdateRequest{Name: "X", Code: "XX"}))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.Update(w, r)
	assertCode(t, w.Code, http.StatusConflict)
}

func TestUpdate_InvalidCode(t *testing.T) {
	h := makeHandler(&mockRepo{store: sampleStore()}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodPut, "/", jbody(UpdateRequest{Name: "X", Code: "bad code!"}))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.Update(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

// ==================== SetStatus ====================

func TestSetStatus_Success(t *testing.T) {
	repo := &mockRepo{store: sampleStore()}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	active := false
	r := httptest.NewRequest(http.MethodPatch, "/", jbody(StatusRequest{IsActive: &active}))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.SetStatus(w, r)
	assertCode(t, w.Code, http.StatusOK)
}

func TestSetStatus_NilIsActive(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodPatch, "/", jbody(map[string]any{}))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.SetStatus(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestSetStatus_NotFound(t *testing.T) {
	repo := &mockRepo{setActErr: ErrNotFound}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	active := true
	r := httptest.NewRequest(http.MethodPatch, "/", jbody(StatusRequest{IsActive: &active}))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.SetStatus(w, r)
	assertCode(t, w.Code, http.StatusNotFound)
}

func TestSetStatus_EmptyID(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	active := true
	r := httptest.NewRequest(http.MethodPatch, "/", jbody(StatusRequest{IsActive: &active}))
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.SetStatus(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

// ==================== AssignUser ====================

func TestAssignUser_Success(t *testing.T) {
	repo := &mockRepo{store: sampleStore()}
	orgs := &mockOrgChecker{isMember: true}
	h := makeHandler(repo, orgs, findUserOK)

	r := httptest.NewRequest(http.MethodPost, "/", jbody(AssignRequest{UserID: "u-1"}))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.AssignUser(w, r)
	assertCode(t, w.Code, http.StatusOK)

	var msg MessageResponse
	json.NewDecoder(w.Body).Decode(&msg)
	if msg.Message == "" {
		t.Fatal("expected message")
	}
}

func TestAssignUser_EmptyUserID(t *testing.T) {
	h := makeHandler(&mockRepo{store: sampleStore()}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodPost, "/", jbody(AssignRequest{UserID: ""}))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.AssignUser(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestAssignUser_EmptyStoreID(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodPost, "/", jbody(AssignRequest{UserID: "u-1"}))
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.AssignUser(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestAssignUser_StoreNotFound(t *testing.T) {
	repo := &mockRepo{findErr: ErrNotFound}
	h := makeHandler(repo, &mockOrgChecker{isMember: true}, findUserOK)

	r := httptest.NewRequest(http.MethodPost, "/", jbody(AssignRequest{UserID: "u-1"}))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.AssignUser(w, r)
	assertCode(t, w.Code, http.StatusNotFound)
}

func TestAssignUser_UserNotFound(t *testing.T) {
	repo := &mockRepo{store: sampleStore()}
	h := makeHandler(repo, &mockOrgChecker{isMember: true}, findUserFail)

	r := httptest.NewRequest(http.MethodPost, "/", jbody(AssignRequest{UserID: "u-999"}))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.AssignUser(w, r)
	assertCode(t, w.Code, http.StatusNotFound)
}

func TestAssignUser_CrossOrg(t *testing.T) {
	repo := &mockRepo{store: sampleStore()}
	h := makeHandler(repo, &mockOrgChecker{isMember: false}, findUserOK)

	r := httptest.NewRequest(http.MethodPost, "/", jbody(AssignRequest{UserID: "u-1"}))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.AssignUser(w, r)
	assertCode(t, w.Code, http.StatusForbidden)
}

func TestAssignUser_AlreadyAssigned(t *testing.T) {
	repo := &mockRepo{store: sampleStore(), assigned: true}
	h := makeHandler(repo, &mockOrgChecker{isMember: true}, findUserOK)

	r := httptest.NewRequest(http.MethodPost, "/", jbody(AssignRequest{UserID: "u-1"}))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.AssignUser(w, r)
	assertCode(t, w.Code, http.StatusConflict)
}

func TestAssignUser_InvalidJSON(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad"))
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.AssignUser(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

// ==================== RemoveUser ====================

func TestRemoveUser_Success(t *testing.T) {
	repo := &mockRepo{store: sampleStore()}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	r.SetPathValue("userId", "u-1")
	w := httptest.NewRecorder()

	h.RemoveUser(w, r)
	assertCode(t, w.Code, http.StatusOK)

	var msg MessageResponse
	json.NewDecoder(w.Body).Decode(&msg)
	if msg.Message == "" {
		t.Fatal("expected message")
	}
}

func TestRemoveUser_EmptyIDs(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.RemoveUser(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestRemoveUser_StoreNotFound(t *testing.T) {
	repo := &mockRepo{findErr: ErrNotFound}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	r.SetPathValue("userId", "u-1")
	w := httptest.NewRecorder()

	h.RemoveUser(w, r)
	assertCode(t, w.Code, http.StatusNotFound)
}

func TestRemoveUser_NotAssigned(t *testing.T) {
	repo := &mockRepo{store: sampleStore(), removeErr: ErrNotAssigned}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	r.SetPathValue("userId", "u-1")
	w := httptest.NewRecorder()

	h.RemoveUser(w, r)
	assertCode(t, w.Code, http.StatusNotFound)
}

// ==================== ListStoreUsers ====================

func TestListStoreUsers_Success(t *testing.T) {
	repo := &mockRepo{
		store:    sampleStore(),
		userList: []users.User{{ID: "u-1", Name: "Alice"}, {ID: "u-2", Name: "Bob"}},
	}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.ListStoreUsers(w, r)
	assertCode(t, w.Code, http.StatusOK)

	var list []users.User
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 2 {
		t.Fatalf("want 2, got %d", len(list))
	}
}

func TestListStoreUsers_EmptyID(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.ListStoreUsers(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestListStoreUsers_StoreNotFound(t *testing.T) {
	repo := &mockRepo{findErr: ErrNotFound}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "s-1")
	w := httptest.NewRecorder()

	h.ListStoreUsers(w, r)
	assertCode(t, w.Code, http.StatusNotFound)
}

// ==================== ListUserStores ====================

func TestListUserStores_Success(t *testing.T) {
	repo := &mockRepo{stores: []Store{{ID: "s-1"}}}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "u-1")
	w := httptest.NewRecorder()

	h.ListUserStores(w, r)
	assertCode(t, w.Code, http.StatusOK)

	var list []Store
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 1 {
		t.Fatalf("want 1, got %d", len(list))
	}
}

func TestListUserStores_EmptyUserID(t *testing.T) {
	h := makeHandler(&mockRepo{}, &mockOrgChecker{}, findUserOK)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.ListUserStores(w, r)
	assertCode(t, w.Code, http.StatusBadRequest)
}

func TestListUserStores_InternalError(t *testing.T) {
	repo := &mockRepo{listErr: errors.New("db")}
	h := makeHandler(repo, &mockOrgChecker{}, findUserOK)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("id", "u-1")
	w := httptest.NewRecorder()

	h.ListUserStores(w, r)
	assertCode(t, w.Code, http.StatusInternalServerError)
}
