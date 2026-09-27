package tables

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

func withOrgID(r *http.Request, orgID string) *http.Request {
	return r.WithContext(middleware.NewContextWithOrgID(r.Context(), orgID))
}

// --- stubs ---

type stubRepo struct {
	createFn       func(ctx context.Context, t *Table) error
	findByIDFn     func(ctx context.Context, orgID, storeID, id string) (*Table, error)
	findByStoreFn  func(ctx context.Context, orgID, storeID string, filter Filter) ([]Table, int, error)
	updateFn       func(ctx context.Context, t *Table) error
	updateStatusFn func(ctx context.Context, orgID, storeID, id, status string) (*Table, error)
	deleteFn       func(ctx context.Context, orgID, storeID, id string) error
	existsByNameFn func(ctx context.Context, storeID, name string, excludeID *string) (bool, error)
}

func (s *stubRepo) Create(ctx context.Context, t *Table) error {
	if s.createFn != nil {
		return s.createFn(ctx, t)
	}
	t.ID = "tbl-1"
	return nil
}

func (s *stubRepo) FindByID(ctx context.Context, orgID, storeID, id string) (*Table, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, orgID, storeID, id)
	}
	return nil, ErrNotFound
}

func (s *stubRepo) FindByStore(ctx context.Context, orgID, storeID string, filter Filter) ([]Table, int, error) {
	if s.findByStoreFn != nil {
		return s.findByStoreFn(ctx, orgID, storeID, filter)
	}
	return nil, 0, nil
}

func (s *stubRepo) Update(ctx context.Context, t *Table) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, t)
	}
	return nil
}

func (s *stubRepo) UpdateStatus(ctx context.Context, orgID, storeID, id, status string) (*Table, error) {
	if s.updateStatusFn != nil {
		return s.updateStatusFn(ctx, orgID, storeID, id, status)
	}
	return &Table{ID: id, Status: status}, nil
}

func (s *stubRepo) Delete(ctx context.Context, orgID, storeID, id string) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, orgID, storeID, id)
	}
	return nil
}

func (s *stubRepo) ExistsByName(ctx context.Context, storeID, name string, excludeID *string) (bool, error) {
	if s.existsByNameFn != nil {
		return s.existsByNameFn(ctx, storeID, name, excludeID)
	}
	return false, nil
}

type stubStoreChecker struct {
	fn func(ctx context.Context, orgID, id string) (*stores.Store, error)
}

func (s *stubStoreChecker) FindByIDInOrg(ctx context.Context, orgID, id string) (*stores.Store, error) {
	if s.fn != nil {
		return s.fn(ctx, orgID, id)
	}
	return &stores.Store{ID: id, OrganizationID: orgID}, nil
}

func newHandler(repo *stubRepo, sc *stubStoreChecker) *Handler {
	return NewHandler(NewService(repo, sc))
}

// --- Create ---

func TestCreate_Success(t *testing.T) {
	repo := &stubRepo{createFn: func(_ context.Context, tb *Table) error {
		tb.ID = "tbl-1"
		return nil
	}}
	h := newHandler(repo, &stubStoreChecker{})

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Meja 1","area":"Indoor","capacity":4}`))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d", w.Code)
	}
	var tb Table
	if err := json.NewDecoder(w.Body).Decode(&tb); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if tb.ID != "tbl-1" || tb.Status != StatusAvailable || !tb.IsActive {
		t.Fatalf("response tak sesuai: %+v", tb)
	}
}

func TestCreate_DuplicateNameConflict(t *testing.T) {
	repo := &stubRepo{existsByNameFn: func(context.Context, string, string, *string) (bool, error) {
		return true, nil
	}}
	h := newHandler(repo, &stubStoreChecker{})

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Meja 1"}`))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

func TestCreate_MissingOrgID(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Meja"}`))
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{bad`))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

// --- List ---

func TestList_EmptyIsArrayNotNull(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.List(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if got := strings.TrimSpace(w.Body.String()); got != "[]" {
		t.Fatalf("list kosong harus [], got %q", got)
	}
}

func TestList_InvalidStatusFilter(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodGet, "/?status=dirty", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.List(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

// --- Update / status / delete ---

func TestUpdateStatus_InvalidStatus(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"status":"dirty"}`))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "tbl-1")
	w := httptest.NewRecorder()

	h.UpdateStatus(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdateStatus_NotFound(t *testing.T) {
	repo := &stubRepo{updateStatusFn: func(context.Context, string, string, string, string) (*Table, error) {
		return nil, ErrNotFound
	}}
	h := newHandler(repo, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"status":"occupied"}`))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "tbl-9")
	w := httptest.NewRecorder()

	h.UpdateStatus(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestUpdate_EmptyNameBadRequest(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"name":"  "}`))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "tbl-1")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestDelete_Message(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "tbl-1")
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var msg MessageResponse
	if err := json.NewDecoder(w.Body).Decode(&msg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if msg.Message == "" {
		t.Fatal("message kosong")
	}
}

func TestByID_MissingID(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.ByID(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}
