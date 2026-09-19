package products

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

// ---------------------------------------------------------------------------
// context helpers
// ---------------------------------------------------------------------------

func withOrgID(r *http.Request, orgID string) *http.Request {
	return r.WithContext(middleware.NewContextWithOrgID(r.Context(), orgID))
}

func withClaims(r *http.Request, claims *appjwt.Claims) *http.Request {
	return r.WithContext(middleware.NewContextWithClaims(r.Context(), claims))
}

// ---------------------------------------------------------------------------
// stubs
// ---------------------------------------------------------------------------

type mockRepo struct {
	product  *Product
	products []Product
	total    int
	err      error
	skuDup   bool
}

func (m *mockRepo) Create(_ context.Context, _ *Product) error { return m.err }
func (m *mockRepo) FindByID(_ context.Context, _, _, _ string) (*Product, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.product, nil
}
func (m *mockRepo) FindByStore(_ context.Context, _, _ string, _ Filter) ([]Product, int, error) {
	return m.products, m.total, m.err
}
func (m *mockRepo) Update(_ context.Context, _ *Product) error         { return m.err }
func (m *mockRepo) SoftDelete(_ context.Context, _, _, _ string) error { return m.err }
func (m *mockRepo) ExistsBySKU(_ context.Context, _, _ string, _ *string) (bool, error) {
	return m.skuDup, m.err
}

type mockStores struct{ err error }

func (m *mockStores) FindByIDInOrg(_ context.Context, orgID, id string) (*stores.Store, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &stores.Store{ID: id, OrganizationID: orgID}, nil
}

type mockDetail struct{}

func (mockDetail) ListVariants(_ context.Context, _, _, _ string) ([]VariantView, error) {
	return []VariantView{}, nil
}
func (mockDetail) ListPrices(_ context.Context, _, _, _ string) ([]PriceView, error) {
	return []PriceView{}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTestHandler(repo *mockRepo, sc *mockStores) *Handler {
	if sc == nil {
		sc = &mockStores{}
	}
	return NewHandler(NewService(repo, sc, mockDetail{}))
}

func validProduct() *Product {
	return &Product{
		ID:             "p-1",
		StoreID:        "store-1",
		OrganizationID: "org-1",
		Name:           "Latte",
		SKU:            "LAT-001",
		Price:          25000,
		Stock:          10,
		Unit:           "pcs",
		IsActive:       true,
	}
}

func createBody(name, sku string, price int64, stock int) string {
	b, _ := json.Marshal(CreateRequest{
		Name:  name,
		SKU:   sku,
		Price: price,
		Stock: stock,
	})
	return string(b)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCreate_Success(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)

	body := createBody("Latte", "LAT-001", 25000, 10)
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r = withClaims(r, &appjwt.Claims{UserID: "user-1"})
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("want JSON content-type, got %s", ct)
	}
}

func TestCreate_MissingOrgID(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_EmptyStoreID(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
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
	h := newTestHandler(&mockRepo{}, nil)
	body := createBody("", "", 0, 0)
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_InvalidPrice(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	body := createBody("Latte", "LAT-001", -100, 10)
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_InvalidStock(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	body := createBody("Latte", "LAT-001", 100, -5)
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_SKUDuplicate(t *testing.T) {
	h := newTestHandler(&mockRepo{skuDup: true}, nil)
	body := createBody("Latte", "LAT-001", 25000, 10)
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
	h := newTestHandler(&mockRepo{}, &mockStores{err: stores.ErrNotFound})
	body := createBody("Latte", "LAT-001", 25000, 10)
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestCreate_InternalError(t *testing.T) {
	h := newTestHandler(&mockRepo{err: errors.New("db down")}, nil)
	body := createBody("Latte", "LAT-001", 25000, 10)
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestList_Success(t *testing.T) {
	p := validProduct()
	repo := &mockRepo{products: []Product{*p}, total: 1}
	h := newTestHandler(repo, nil)

	r := httptest.NewRequest(http.MethodGet, "/?page=1&limit=10&search=latte", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.List(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp ListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("want total 1, got %d", resp.Total)
	}
}

func TestList_MissingOrgID(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()
	h.List(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestList_EmptyStoreID(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()
	h.List(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestList_StoreNotFound(t *testing.T) {
	h := newTestHandler(&mockRepo{}, &mockStores{err: stores.ErrNotFound})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()
	h.List(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestList_EmptyResult(t *testing.T) {
	h := newTestHandler(&mockRepo{products: nil, total: 0}, nil)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()
	h.List(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp ListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data == nil {
		t.Fatal("want non-nil data slice for empty list")
	}
	if len(resp.Data) != 0 {
		t.Fatalf("want 0 items, got %d", len(resp.Data))
	}
}

// ---------------------------------------------------------------------------
// ByID
// ---------------------------------------------------------------------------

func TestByID_Success(t *testing.T) {
	p := validProduct()
	h := newTestHandler(&mockRepo{product: p}, nil)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")

	w := httptest.NewRecorder()
	h.ByID(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp ProductDetail
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "p-1" {
		t.Fatalf("want id p-1, got %s", resp.ID)
	}
}

func TestByID_MissingOrgID(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")
	w := httptest.NewRecorder()
	h.ByID(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestByID_EmptyIDs(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()
	h.ByID(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestByID_NotFound(t *testing.T) {
	h := newTestHandler(&mockRepo{err: ErrNotFound}, nil)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "no-such")
	w := httptest.NewRecorder()
	h.ByID(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestByID_InternalError(t *testing.T) {
	h := newTestHandler(&mockRepo{err: errors.New("db")}, nil)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")
	w := httptest.NewRecorder()
	h.ByID(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func updateBody(name, sku string, price int64, stock int, isActive bool) []byte {
	b := &isActive
	data, _ := json.Marshal(UpdateRequest{
		Name: name, SKU: sku, Price: price, Stock: stock, IsActive: b,
	})
	return data
}

func TestUpdate_Success(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	body := updateBody("Cappuccino", "CAP-001", 30000, 5, true)
	r := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")
	w := httptest.NewRecorder()
	h.Update(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdate_MissingOrgID(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	body := updateBody("X", "X", 0, 0, true)
	r := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body))
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")
	w := httptest.NewRecorder()
	h.Update(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdate_EmptyIDs(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	body := updateBody("X", "X", 0, 0, true)
	r := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body))
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()
	h.Update(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdate_InvalidJSON(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	r := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`not json`))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")
	w := httptest.NewRecorder()
	h.Update(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdate_MissingIsActive(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	body, _ := json.Marshal(map[string]any{
		"name": "X", "sku": "X", "price": 0, "stock": 0,
	})
	r := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")
	w := httptest.NewRecorder()
	h.Update(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "is_active") {
		t.Fatalf("want is_active error, got: %s", w.Body.String())
	}
}

func TestUpdate_InvalidInput(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	body := updateBody("", "", 0, 0, true)
	r := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")
	w := httptest.NewRecorder()
	h.Update(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdate_SKUDuplicate(t *testing.T) {
	h := newTestHandler(&mockRepo{skuDup: true}, nil)
	body := updateBody("Latte", "DUPE", 25000, 10, true)
	r := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")
	w := httptest.NewRecorder()
	h.Update(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	h := newTestHandler(&mockRepo{err: ErrNotFound}, nil)
	body := updateBody("Latte", "LAT-001", 25000, 10, true)
	r := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body))
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")
	w := httptest.NewRecorder()
	h.Update(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestDelete_Success(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")
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
		t.Fatal("want non-empty message")
	}
}

func TestDelete_MissingOrgID(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")
	w := httptest.NewRecorder()
	h.Delete(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestDelete_EmptyIDs(t *testing.T) {
	h := newTestHandler(&mockRepo{}, nil)
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	w := httptest.NewRecorder()
	h.Delete(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestDelete_NotFound(t *testing.T) {
	h := newTestHandler(&mockRepo{err: ErrNotFound}, nil)
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")
	w := httptest.NewRecorder()
	h.Delete(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestDelete_InternalError(t *testing.T) {
	h := newTestHandler(&mockRepo{err: errors.New("db")}, nil)
	r := httptest.NewRequest(http.MethodDelete, "/", nil)
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "p-1")
	w := httptest.NewRecorder()
	h.Delete(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}
