package prices

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
	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	"github.com/ramadhanrzq/backend-go/internal/modules/variants"
)

func withOrgID(r *http.Request, orgID string) *http.Request {
	ctx := middleware.NewContextWithOrgID(r.Context(), orgID)
	return r.WithContext(ctx)
}

// --- stub repo ---

type stubRepo struct {
	createFn        func(ctx context.Context, p *ProductPrice) error
	findByIDFn      func(ctx context.Context, orgID, storeID, productID, priceID string) (*ProductPrice, error)
	findByProductFn func(ctx context.Context, orgID, storeID, productID string, onlyActive bool) ([]ProductPrice, error)
	updateFn        func(ctx context.Context, p *ProductPrice) error
	deleteFn        func(ctx context.Context, orgID, storeID, productID, priceID string) error
	findEffectiveFn func(ctx context.Context, orgID, storeID, productID string, variantID *string, priceType string, qty int, now time.Time) (*ProductPrice, error)
}

func (s *stubRepo) Create(ctx context.Context, p *ProductPrice) error {
	if s.createFn != nil {
		return s.createFn(ctx, p)
	}
	p.ID = "price-1"
	return nil
}

func (s *stubRepo) FindByID(ctx context.Context, orgID, storeID, productID, priceID string) (*ProductPrice, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, orgID, storeID, productID, priceID)
	}
	return &ProductPrice{ID: priceID}, nil
}

func (s *stubRepo) FindByProduct(ctx context.Context, orgID, storeID, productID string, onlyActive bool) ([]ProductPrice, error) {
	if s.findByProductFn != nil {
		return s.findByProductFn(ctx, orgID, storeID, productID, onlyActive)
	}
	return []ProductPrice{}, nil
}

func (s *stubRepo) Update(ctx context.Context, p *ProductPrice) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, p)
	}
	return nil
}

func (s *stubRepo) Delete(ctx context.Context, orgID, storeID, productID, priceID string) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, orgID, storeID, productID, priceID)
	}
	return nil
}

func (s *stubRepo) FindEffective(ctx context.Context, orgID, storeID, productID string, variantID *string, priceType string, qty int, now time.Time) (*ProductPrice, error) {
	if s.findEffectiveFn != nil {
		return s.findEffectiveFn(ctx, orgID, storeID, productID, variantID, priceType, qty, now)
	}
	return &ProductPrice{}, nil
}

// --- stub store checker ---

type stubStoreChecker struct {
	fn func(ctx context.Context, orgID, id string) (*stores.Store, error)
}

func (s *stubStoreChecker) FindByIDInOrg(ctx context.Context, orgID, id string) (*stores.Store, error) {
	if s.fn != nil {
		return s.fn(ctx, orgID, id)
	}
	return &stores.Store{ID: id, OrganizationID: orgID}, nil
}

// --- stub product checker ---

type stubProductChecker struct {
	fn func(ctx context.Context, orgID, storeID, id string) (*products.Product, error)
}

func (s *stubProductChecker) FindByID(ctx context.Context, orgID, storeID, id string) (*products.Product, error) {
	if s.fn != nil {
		return s.fn(ctx, orgID, storeID, id)
	}
	return &products.Product{ID: id, StoreID: storeID, OrganizationID: orgID}, nil
}

// --- stub variant checker ---

type stubVariantChecker struct {
	fn func(ctx context.Context, orgID, storeID, productID, id string) (*variants.ProductVariant, error)
}

func (s *stubVariantChecker) FindByID(ctx context.Context, orgID, storeID, productID, id string) (*variants.ProductVariant, error) {
	if s.fn != nil {
		return s.fn(ctx, orgID, storeID, productID, id)
	}
	return &variants.ProductVariant{ID: id}, nil
}

// --- helpers ---

func setupHandler(repo *stubRepo, sc *stubStoreChecker, pc *stubProductChecker, vc *stubVariantChecker) *Handler {
	if repo == nil {
		repo = &stubRepo{}
	}
	if sc == nil {
		sc = &stubStoreChecker{}
	}
	if pc == nil {
		pc = &stubProductChecker{}
	}
	if vc == nil {
		vc = &stubVariantChecker{}
	}
	svc := NewService(repo, sc, pc, vc)
	return NewHandler(svc)
}

func jsonBody(v any) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

func makeCreateReq(r *http.Request) *http.Request {
	r = withOrgID(r, "org-1")
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("productId", "prod-1")
	return r
}

// --- Create tests ---

func TestCreate_Success(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	body := jsonBody(CreateRequest{PriceType: "retail", Price: 10000, MinQuantity: 1})
	req := httptest.NewRequest(http.MethodPost, "/prices", body)
	req = makeCreateReq(req)

	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	var got ProductPrice
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID == "" {
		t.Fatal("expected price ID in response")
	}
}

func TestCreate_MinQuantityDefaultsTo1(t *testing.T) {
	var captured int
	repo := &stubRepo{
		createFn: func(_ context.Context, p *ProductPrice) error {
			captured = p.MinQuantity
			p.ID = "price-1"
			return nil
		},
	}
	h := setupHandler(repo, nil, nil, nil)

	body := jsonBody(CreateRequest{PriceType: "retail", Price: 5000, MinQuantity: 0})
	req := httptest.NewRequest(http.MethodPost, "/prices", body)
	req = makeCreateReq(req)

	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	if captured != 1 {
		t.Fatalf("want min_quantity=1, got %d", captured)
	}
}

func TestCreate_NoOrgID(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	body := jsonBody(CreateRequest{PriceType: "retail", Price: 10000})
	req := httptest.NewRequest(http.MethodPost, "/prices", body)
	// no org ID injected
	req.SetPathValue("storeId", "store-1")
	req.SetPathValue("productId", "prod-1")

	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_EmptyStoreID(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	body := jsonBody(CreateRequest{PriceType: "retail", Price: 10000})
	req := httptest.NewRequest(http.MethodPost, "/prices", body)
	req = withOrgID(req, "org-1")
	req.SetPathValue("storeId", "")
	req.SetPathValue("productId", "prod-1")

	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/prices", bytes.NewBufferString("{bad"))
	req = makeCreateReq(req)

	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_InvalidInput(t *testing.T) {
	// Price < 0 triggers ErrInvalidInput from service
	h := setupHandler(nil, nil, nil, nil)

	body := jsonBody(CreateRequest{PriceType: "retail", Price: -1, MinQuantity: 1})
	req := httptest.NewRequest(http.MethodPost, "/prices", body)
	req = makeCreateReq(req)

	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_StoreNotFound(t *testing.T) {
	sc := &stubStoreChecker{fn: func(context.Context, string, string) (*stores.Store, error) {
		return nil, ErrStoreNotFound
	}}
	h := setupHandler(nil, sc, nil, nil)

	body := jsonBody(CreateRequest{PriceType: "retail", Price: 5000, MinQuantity: 1})
	req := httptest.NewRequest(http.MethodPost, "/prices", body)
	req = makeCreateReq(req)

	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_ProductNotFound(t *testing.T) {
	pc := &stubProductChecker{fn: func(context.Context, string, string, string) (*products.Product, error) {
		return nil, ErrProductNotFound
	}}
	h := setupHandler(nil, nil, pc, nil)

	body := jsonBody(CreateRequest{PriceType: "retail", Price: 5000, MinQuantity: 1})
	req := httptest.NewRequest(http.MethodPost, "/prices", body)
	req = makeCreateReq(req)

	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_VariantNotFound(t *testing.T) {
	vid := "var-missing"
	vc := &stubVariantChecker{fn: func(context.Context, string, string, string, string) (*variants.ProductVariant, error) {
		return nil, ErrVariantNotFound
	}}
	h := setupHandler(nil, nil, nil, vc)

	body := jsonBody(CreateRequest{VariantID: &vid, PriceType: "retail", Price: 5000, MinQuantity: 1})
	req := httptest.NewRequest(http.MethodPost, "/prices", body)
	req = makeCreateReq(req)

	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_InternalError(t *testing.T) {
	repo := &stubRepo{createFn: func(context.Context, *ProductPrice) error {
		return errSentinel
	}}
	h := setupHandler(repo, nil, nil, nil)

	body := jsonBody(CreateRequest{PriceType: "retail", Price: 5000, MinQuantity: 1})
	req := httptest.NewRequest(http.MethodPost, "/prices", body)
	req = makeCreateReq(req)

	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

var errSentinel = errors.New("boom")

// --- List tests ---

func TestList_Success(t *testing.T) {
	repo := &stubRepo{
		findByProductFn: func(context.Context, string, string, string, bool) ([]ProductPrice, error) {
			return []ProductPrice{{ID: "p1"}, {ID: "p2"}}, nil
		},
	}
	h := setupHandler(repo, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/prices", nil)
	req = makeCreateReq(req) // reuses orgID + path values

	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp ListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("want 2 prices, got %d", len(resp.Data))
	}
}

func TestList_EmptyReturnsEmptyArray(t *testing.T) {
	repo := &stubRepo{
		findByProductFn: func(context.Context, string, string, string, bool) ([]ProductPrice, error) {
			return nil, nil // nil slice
		},
	}
	h := setupHandler(repo, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/prices", nil)
	req = makeCreateReq(req)

	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp ListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Data == nil {
		t.Fatal("want non-nil empty slice, got nil")
	}
}

func TestList_NoOrgID(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/prices", nil)
	req.SetPathValue("storeId", "s1")
	req.SetPathValue("productId", "p1")

	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestList_ActiveQueryParam(t *testing.T) {
	var capturedActive bool
	repo := &stubRepo{
		findByProductFn: func(_ context.Context, _, _, _ string, onlyActive bool) ([]ProductPrice, error) {
			capturedActive = onlyActive
			return []ProductPrice{}, nil
		},
	}
	h := setupHandler(repo, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/prices?active=true", nil)
	req = makeCreateReq(req)

	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if !capturedActive {
		t.Fatal("expected onlyActive=true")
	}
}

func TestList_ServiceError(t *testing.T) {
	repo := &stubRepo{
		findByProductFn: func(context.Context, string, string, string, bool) ([]ProductPrice, error) {
			return nil, errSentinel
		},
	}
	h := setupHandler(repo, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/prices", nil)
	req = makeCreateReq(req)

	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

// --- Update tests ---

func boolPtr(v bool) *bool { return &v }

func TestUpdate_Success(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	body := jsonBody(UpdateRequest{PriceType: "retail", Price: 15000, MinQuantity: 2, IsActive: boolPtr(true)})
	req := httptest.NewRequest(http.MethodPut, "/prices/price-1", body)
	req = makeCreateReq(req)
	req.SetPathValue("priceId", "price-1")

	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var got ProductPrice
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
}

func TestUpdate_NoOrgID(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	body := jsonBody(UpdateRequest{PriceType: "retail", Price: 10000, IsActive: boolPtr(true)})
	req := httptest.NewRequest(http.MethodPut, "/prices/price-1", body)
	req.SetPathValue("storeId", "s1")
	req.SetPathValue("productId", "p1")
	req.SetPathValue("priceId", "price-1")

	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdate_EmptyPriceID(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	body := jsonBody(UpdateRequest{PriceType: "retail", Price: 10000, IsActive: boolPtr(true)})
	req := httptest.NewRequest(http.MethodPut, "/prices/", body)
	req = withOrgID(req, "org-1")
	req.SetPathValue("storeId", "s1")
	req.SetPathValue("productId", "p1")
	req.SetPathValue("priceId", "")

	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdate_InvalidJSON(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/prices/price-1", bytes.NewBufferString("nope"))
	req = makeCreateReq(req)
	req.SetPathValue("priceId", "price-1")

	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdate_IsActiveRequired(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	body := jsonBody(UpdateRequest{PriceType: "retail", Price: 10000}) // IsActive nil
	req := httptest.NewRequest(http.MethodPut, "/prices/price-1", body)
	req = makeCreateReq(req)
	req.SetPathValue("priceId", "price-1")

	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdate_NotFound(t *testing.T) {
	repo := &stubRepo{
		updateFn: func(context.Context, *ProductPrice) error {
			return ErrNotFound
		},
	}
	h := setupHandler(repo, nil, nil, nil)

	body := jsonBody(UpdateRequest{PriceType: "retail", Price: 10000, MinQuantity: 1, IsActive: boolPtr(true)})
	req := httptest.NewRequest(http.MethodPut, "/prices/price-1", body)
	req = makeCreateReq(req)
	req.SetPathValue("priceId", "price-1")

	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdate_InvalidInput(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	body := jsonBody(UpdateRequest{PriceType: "retail", Price: -1, MinQuantity: 1, IsActive: boolPtr(true)})
	req := httptest.NewRequest(http.MethodPut, "/prices/price-1", body)
	req = makeCreateReq(req)
	req.SetPathValue("priceId", "price-1")

	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdate_InternalError(t *testing.T) {
	repo := &stubRepo{
		updateFn: func(context.Context, *ProductPrice) error {
			return errSentinel
		},
	}
	h := setupHandler(repo, nil, nil, nil)

	body := jsonBody(UpdateRequest{PriceType: "retail", Price: 5000, MinQuantity: 1, IsActive: boolPtr(true)})
	req := httptest.NewRequest(http.MethodPut, "/prices/price-1", body)
	req = makeCreateReq(req)
	req.SetPathValue("priceId", "price-1")

	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

// --- Delete tests ---

func TestDelete_Success(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/prices/price-1", nil)
	req = makeCreateReq(req)
	req.SetPathValue("priceId", "price-1")

	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp MessageResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Message == "" {
		t.Fatal("expected non-empty message")
	}
}

func TestDelete_NoOrgID(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/prices/price-1", nil)
	req.SetPathValue("storeId", "s1")
	req.SetPathValue("productId", "p1")
	req.SetPathValue("priceId", "price-1")

	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestDelete_EmptyPriceID(t *testing.T) {
	h := setupHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/prices/", nil)
	req = withOrgID(req, "org-1")
	req.SetPathValue("storeId", "s1")
	req.SetPathValue("productId", "p1")
	req.SetPathValue("priceId", "")

	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := &stubRepo{
		deleteFn: func(context.Context, string, string, string, string) error {
			return ErrNotFound
		},
	}
	h := setupHandler(repo, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/prices/price-1", nil)
	req = makeCreateReq(req)
	req.SetPathValue("priceId", "price-1")

	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDelete_InternalError(t *testing.T) {
	repo := &stubRepo{
		deleteFn: func(context.Context, string, string, string, string) error {
			return errSentinel
		},
	}
	h := setupHandler(repo, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/prices/price-1", nil)
	req = makeCreateReq(req)
	req.SetPathValue("priceId", "price-1")

	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}
