package categories

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

func withOrgID(r *http.Request, orgID string) *http.Request {
	ctx := middleware.NewContextWithOrgID(r.Context(), orgID)
	return r.WithContext(ctx)
}

// --- stubs ---

type stubRepo struct {
	createFn        func(ctx context.Context, c *Category) error
	findByIDFn      func(ctx context.Context, orgID, storeID, id string) (*Category, error)
	findBySlugFn    func(ctx context.Context, orgID, storeID, slug string) (*Category, error)
	findByStoreFn   func(ctx context.Context, orgID, storeID string) ([]Category, error)
	updateFn        func(ctx context.Context, c *Category) error
	deleteFn        func(ctx context.Context, orgID, storeID, id string) error
	existsBySlugFn  func(ctx context.Context, slug, storeID string, excludeID *string) (bool, error)
	countProductsFn func(ctx context.Context, storeID, categoryID string) (int, error)
}

func (s *stubRepo) Create(ctx context.Context, c *Category) error {
	if s.createFn != nil {
		return s.createFn(ctx, c)
	}
	c.ID = "cat-1"
	return nil
}
func (s *stubRepo) FindByID(ctx context.Context, orgID, storeID, id string) (*Category, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, orgID, storeID, id)
	}
	return nil, ErrNotFound
}
func (s *stubRepo) FindBySlug(ctx context.Context, orgID, storeID, slug string) (*Category, error) {
	if s.findBySlugFn != nil {
		return s.findBySlugFn(ctx, orgID, storeID, slug)
	}
	return nil, ErrNotFound
}
func (s *stubRepo) FindByStore(ctx context.Context, orgID, storeID string) ([]Category, error) {
	if s.findByStoreFn != nil {
		return s.findByStoreFn(ctx, orgID, storeID)
	}
	return nil, nil
}
func (s *stubRepo) Update(ctx context.Context, c *Category) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, c)
	}
	return nil
}
func (s *stubRepo) Delete(ctx context.Context, orgID, storeID, id string) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, orgID, storeID, id)
	}
	return nil
}
func (s *stubRepo) ExistsBySlug(ctx context.Context, slug, storeID string, excludeID *string) (bool, error) {
	if s.existsBySlugFn != nil {
		return s.existsBySlugFn(ctx, slug, storeID, excludeID)
	}
	return false, nil
}
func (s *stubRepo) CountProducts(ctx context.Context, storeID, categoryID string) (int, error) {
	if s.countProductsFn != nil {
		return s.countProductsFn(ctx, storeID, categoryID)
	}
	return 0, nil
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
	svc := NewService(repo, sc)
	return NewHandler(svc)
}

func sampleCategory() *Category {
	return &Category{
		ID:             "cat-1",
		OrganizationID: "org-1",
		StoreID:        "store-1",
		Name:           "Beverages",
		Slug:           "beverages",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// --- Create ---

func TestCreate_Success(t *testing.T) {
	repo := &stubRepo{
		createFn: func(_ context.Context, c *Category) error {
			c.ID = "cat-1"
			return nil
		},
	}
	h := newHandler(repo, &stubStoreChecker{})

	body := `{"name":"Beverages","slug":"beverages"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d", w.Code)
	}
	var cat Category
	if err := json.NewDecoder(w.Body).Decode(&cat); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cat.ID == "" {
		t.Fatal("expected non-empty ID")
	}
}

func TestCreate_MissingOrgID(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"x"}`))
	// no org_id in context
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_EmptyStoreID(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"x"}`))
	r = withOrgID(r, "org-1")
	// storeId not set
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

func TestCreate_InvalidInput(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	// empty name triggers ErrInvalidInput from service
	body := `{"name":"","slug":""}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_SlugExists(t *testing.T) {
	repo := &stubRepo{
		existsBySlugFn: func(_ context.Context, slug, storeID string, _ *string) (bool, error) {
			return true, nil
		},
	}
	h := newHandler(repo, &stubStoreChecker{})
	body := `{"name":"Beverages","slug":"beverages"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

func TestCreate_StoreNotFound(t *testing.T) {
	sc := &stubStoreChecker{
		fn: func(_ context.Context, _, _ string) (*stores.Store, error) {
			return nil, stores.ErrNotFound
		},
	}
	h := newHandler(&stubRepo{}, sc)
	body := `{"name":"Beverages","slug":"beverages"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Create(w, r)

	// checkStore error → service returns ErrNotFound → handler returns 404
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestCreate_InternalError(t *testing.T) {
	repo := &stubRepo{
		createFn: func(_ context.Context, _ *Category) error {
			return errors.New("db down")
		},
	}
	h := newHandler(repo, &stubStoreChecker{})
	body := `{"name":"Beverages","slug":"beverages"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

// --- List ---

func TestList_Success(t *testing.T) {
	repo := &stubRepo{
		findByStoreFn: func(_ context.Context, _, _ string) ([]Category, error) {
			return []Category{*sampleCategory()}, nil
		},
	}
	h := newHandler(repo, &stubStoreChecker{})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.List(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var cats []Category
	if err := json.NewDecoder(w.Body).Decode(&cats); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(cats) != 1 {
		t.Fatalf("want 1 category, got %d", len(cats))
	}
}

func TestList_MissingOrgID(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.List(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestList_EmptyStoreID(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.List(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestList_StoreNotFound(t *testing.T) {
	sc := &stubStoreChecker{
		fn: func(_ context.Context, _, _ string) (*stores.Store, error) {
			return nil, stores.ErrNotFound
		},
	}
	h := newHandler(&stubRepo{}, sc)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.List(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

// --- ByID ---

func TestByID_Success(t *testing.T) {
	cat := sampleCategory()
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, _, _, _ string) (*Category, error) {
			return cat, nil
		},
	}
	h := newHandler(repo, &stubStoreChecker{})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-1")
	w := httptest.NewRecorder()

	h.ByID(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var got Category
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != "cat-1" {
		t.Fatalf("want cat-1, got %s", got.ID)
	}
}

func TestByID_MissingOrgID(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-1")
	w := httptest.NewRecorder()

	h.ByID(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestByID_EmptyIDs(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	// storeId and id both empty
	w := httptest.NewRecorder()

	h.ByID(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestByID_NotFound(t *testing.T) {
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, _, _, _ string) (*Category, error) {
			return nil, ErrNotFound
		},
	}
	h := newHandler(repo, &stubStoreChecker{})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-999")
	w := httptest.NewRecorder()

	h.ByID(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestByID_InternalError(t *testing.T) {
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, _, _, _ string) (*Category, error) {
			return nil, errors.New("db down")
		},
	}
	h := newHandler(repo, &stubStoreChecker{})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-1")
	w := httptest.NewRecorder()

	h.ByID(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

// --- Update ---

func TestUpdate_Success(t *testing.T) {
	cat := sampleCategory()
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, _, _, _ string) (*Category, error) {
			return cat, nil
		},
	}
	h := newHandler(repo, &stubStoreChecker{})

	body := `{"name":"Coffee","slug":"coffee"}`
	r := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-1")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var got Category
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "Coffee" {
		t.Fatalf("want name Coffee, got %s", got.Name)
	}
}

func TestUpdate_MissingOrgID(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"name":"x"}`))
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-1")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdate_EmptyIDs(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"name":"x"}`))
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdate_InvalidJSON(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{bad`))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-1")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, _, _, _ string) (*Category, error) {
			return nil, ErrNotFound
		},
	}
	h := newHandler(repo, &stubStoreChecker{})

	body := `{"name":"Coffee","slug":"coffee"}`
	r := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-999")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestUpdate_SlugExists(t *testing.T) {
	cat := sampleCategory()
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, _, _, _ string) (*Category, error) {
			return cat, nil
		},
		existsBySlugFn: func(_ context.Context, slug, storeID string, excludeID *string) (bool, error) {
			return true, nil
		},
	}
	h := newHandler(repo, &stubStoreChecker{})

	body := `{"name":"Coffee","slug":"taken-slug"}`
	r := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-1")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

func TestUpdate_InternalError(t *testing.T) {
	cat := sampleCategory()
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, _, _, _ string) (*Category, error) {
			return cat, nil
		},
		updateFn: func(_ context.Context, _ *Category) error {
			return errors.New("db down")
		},
	}
	h := newHandler(repo, &stubStoreChecker{})

	body := `{"name":"Coffee","slug":"coffee"}`
	r := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-1")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

// --- Delete ---

func TestDelete_Success(t *testing.T) {
	cat := sampleCategory()
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, _, _, _ string) (*Category, error) {
			return cat, nil
		},
	}
	h := newHandler(repo, &stubStoreChecker{})

	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-1")
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
		t.Fatal("expected non-empty message")
	}
}

func TestDelete_MissingOrgID(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-1")
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestDelete_EmptyIDs(t *testing.T) {
	h := newHandler(&stubRepo{}, &stubStoreChecker{})
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, _, _, _ string) (*Category, error) {
			return nil, ErrNotFound
		},
	}
	h := newHandler(repo, &stubStoreChecker{})

	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-999")
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestDelete_InUse(t *testing.T) {
	cat := sampleCategory()
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, _, _, _ string) (*Category, error) {
			return cat, nil
		},
		countProductsFn: func(_ context.Context, _, _ string) (int, error) {
			return 5, nil
		},
	}
	h := newHandler(repo, &stubStoreChecker{})

	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-1")
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

func TestDelete_InternalError(t *testing.T) {
	cat := sampleCategory()
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, _, _, _ string) (*Category, error) {
			return cat, nil
		},
		deleteFn: func(_ context.Context, _, _, _ string) error {
			return errors.New("db down")
		},
	}
	h := newHandler(repo, &stubStoreChecker{})

	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "cat-1")
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

