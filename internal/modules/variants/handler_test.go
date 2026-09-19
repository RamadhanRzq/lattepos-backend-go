package variants

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

func withOrgID(r *http.Request, orgID string) *http.Request {
	ctx := middleware.NewContextWithOrgID(r.Context(), orgID)
	return r.WithContext(ctx)
}

// --- stubs ---

type stubRepo struct {
	createFn        func(ctx context.Context, v *ProductVariant) error
	findByProductFn func(ctx context.Context, orgID, storeID, productID string) ([]ProductVariant, error)
	findByIDFn      func(ctx context.Context, orgID, storeID, productID, id string) (*ProductVariant, error)
	updateFn        func(ctx context.Context, v *ProductVariant) error
	deleteFn        func(ctx context.Context, orgID, storeID, productID, id string) error
	existsBySKUFn   func(ctx context.Context, storeID, sku, excludeID string) (bool, error)
}

func (s *stubRepo) Create(ctx context.Context, v *ProductVariant) error {
	if s.createFn != nil {
		return s.createFn(ctx, v)
	}
	return nil
}
func (s *stubRepo) FindByID(ctx context.Context, orgID, storeID, productID, id string) (*ProductVariant, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, orgID, storeID, productID, id)
	}
	return &ProductVariant{ID: id}, nil
}
func (s *stubRepo) FindByProduct(ctx context.Context, orgID, storeID, productID string) ([]ProductVariant, error) {
	if s.findByProductFn != nil {
		return s.findByProductFn(ctx, orgID, storeID, productID)
	}
	return nil, nil
}
func (s *stubRepo) Update(ctx context.Context, v *ProductVariant) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, v)
	}
	return nil
}
func (s *stubRepo) Delete(ctx context.Context, orgID, storeID, productID, id string) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, orgID, storeID, productID, id)
	}
	return nil
}
func (s *stubRepo) ExistsBySKU(ctx context.Context, storeID, sku, excludeID string) (bool, error) {
	if s.existsBySKUFn != nil {
		return s.existsBySKUFn(ctx, storeID, sku, excludeID)
	}
	return false, nil
}

type stubStoreChecker struct {
	err error
}

func (s *stubStoreChecker) FindByIDInOrg(ctx context.Context, orgID, id string) (*stores.Store, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &stores.Store{ID: id, OrganizationID: orgID}, nil
}

type stubProductChecker struct {
	err error
}

func (s *stubProductChecker) FindByID(ctx context.Context, orgID, storeID, id string) (*products.Product, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &products.Product{ID: id, StoreID: storeID, OrganizationID: orgID}, nil
}

func newTestHandler(repo *stubRepo, sc *stubStoreChecker, pc *stubProductChecker) *Handler {
	if repo == nil {
		repo = &stubRepo{}
	}
	if sc == nil {
		sc = &stubStoreChecker{}
	}
	if pc == nil {
		pc = &stubProductChecker{}
	}
	svc := NewService(repo, sc, pc)
	return NewHandler(svc)
}

func jsonBody(v any) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

// --- Create tests ---

func TestCreate_Success(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	body := jsonBody(CreateRequest{Name: "Large", SKU: "LRG-001", Stock: 10})
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("productId", "prod-1")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	var got ProductVariant
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "Large" {
		t.Errorf("name = %q, want Large", got.Name)
	}
}

func TestCreate_MissingOrgID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	body := jsonBody(CreateRequest{Name: "L"})
	r := httptest.NewRequest(http.MethodPost, "/", body)
	// no org_id injected
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_EmptyStoreID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	body := jsonBody(CreateRequest{Name: "L"})
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "")
	r.SetPathValue("productId", "prod-1")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad"))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_InvalidInput(t *testing.T) {
	// empty name triggers ErrInvalidInput from service
	h := newTestHandler(nil, nil, nil)
	body := jsonBody(CreateRequest{Name: "", Stock: 0})
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_SKUExists(t *testing.T) {
	repo := &stubRepo{
		existsBySKUFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return true, nil
		},
	}
	h := newTestHandler(repo, nil, nil)
	body := jsonBody(CreateRequest{Name: "L", SKU: "DUP", Stock: 1})
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_StoreNotFound(t *testing.T) {
	sc := &stubStoreChecker{err: stores.ErrNotFound}
	h := newTestHandler(nil, sc, nil)
	body := jsonBody(CreateRequest{Name: "L", Stock: 1})
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	w := httptest.NewRecorder()

	h.Create(w, r)

	// store not found → checkProduct returns store error → service wraps as ErrStoreNotFound → 404
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_InternalError(t *testing.T) {
	repo := &stubRepo{
		createFn: func(_ context.Context, _ *ProductVariant) error {
			return errors.New("db down")
		},
	}
	h := newTestHandler(repo, nil, nil)
	body := jsonBody(CreateRequest{Name: "L", Stock: 1})
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d: %s", w.Code, w.Body.String())
	}
}

// --- List tests ---

func TestList_Success(t *testing.T) {
	repo := &stubRepo{
		findByProductFn: func(_ context.Context, _, _, _ string) ([]ProductVariant, error) {
			return []ProductVariant{{ID: "v1", Name: "Small"}}, nil
		},
	}
	h := newTestHandler(repo, nil, nil)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	w := httptest.NewRecorder()

	h.List(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp ListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("want 1 variant, got %d", len(resp.Data))
	}
}

func TestList_EmptyReturnsEmptyArray(t *testing.T) {
	repo := &stubRepo{
		findByProductFn: func(_ context.Context, _, _, _ string) ([]ProductVariant, error) {
			return nil, nil
		},
	}
	h := newTestHandler(repo, nil, nil)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	w := httptest.NewRecorder()

	h.List(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp ListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Data == nil || len(resp.Data) != 0 {
		t.Fatalf("want empty non-nil data slice")
	}
}

func TestList_MissingOrgID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	w := httptest.NewRecorder()

	h.List(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestList_ProductNotFound(t *testing.T) {
	pc := &stubProductChecker{err: products.ErrNotFound}
	h := newTestHandler(nil, nil, pc)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	w := httptest.NewRecorder()

	h.List(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Update tests ---

func boolPtr(b bool) *bool { return &b }

func TestUpdate_Success(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	body := jsonBody(UpdateRequest{Name: "New", SKU: "S1", Stock: 5, IsActive: boolPtr(true)})
	r := httptest.NewRequest(http.MethodPut, "/", body)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	r.SetPathValue("variantId", "v1")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var got ProductVariant
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "New" {
		t.Errorf("name = %q, want New", got.Name)
	}
}

func TestUpdate_MissingIsActive(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	// IsActive omitted (nil)
	body := jsonBody(map[string]any{"name": "X", "sku": "", "stock": 1})
	r := httptest.NewRequest(http.MethodPut, "/", body)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	r.SetPathValue("variantId", "v1")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdate_InvalidJSON(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	r := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString("nope"))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	r.SetPathValue("variantId", "v1")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdate_EmptyVariantID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	body := jsonBody(UpdateRequest{Name: "X", IsActive: boolPtr(true)})
	r := httptest.NewRequest(http.MethodPut, "/", body)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	r.SetPathValue("variantId", "")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	repo := &stubRepo{
		updateFn: func(_ context.Context, _ *ProductVariant) error {
			return ErrNotFound
		},
	}
	h := newTestHandler(repo, nil, nil)
	body := jsonBody(UpdateRequest{Name: "X", SKU: "", Stock: 1, IsActive: boolPtr(true)})
	r := httptest.NewRequest(http.MethodPut, "/", body)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	r.SetPathValue("variantId", "v1")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdate_SKUExists(t *testing.T) {
	repo := &stubRepo{
		existsBySKUFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return true, nil
		},
	}
	h := newTestHandler(repo, nil, nil)
	body := jsonBody(UpdateRequest{Name: "X", SKU: "DUP", Stock: 1, IsActive: boolPtr(true)})
	r := httptest.NewRequest(http.MethodPut, "/", body)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	r.SetPathValue("variantId", "v1")
	w := httptest.NewRecorder()

	h.Update(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Delete tests ---

func TestDelete_Success(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	r.SetPathValue("variantId", "v1")
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp MessageResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestDelete_MissingOrgID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	r.SetPathValue("variantId", "v1")
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestDelete_EmptyVariantID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	r.SetPathValue("variantId", "")
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := &stubRepo{
		deleteFn: func(_ context.Context, _, _, _, _ string) error {
			return ErrNotFound
		},
	}
	h := newTestHandler(repo, nil, nil)
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	r.SetPathValue("variantId", "v1")
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDelete_InternalError(t *testing.T) {
	repo := &stubRepo{
		deleteFn: func(_ context.Context, _, _, _, _ string) error {
			return errors.New("db down")
		},
	}
	h := newTestHandler(repo, nil, nil)
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "s")
	r.SetPathValue("productId", "p")
	r.SetPathValue("variantId", "v1")
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d: %s", w.Code, w.Body.String())
	}
}
