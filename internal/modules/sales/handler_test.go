package sales

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

func withOrgID(r *http.Request, orgID string) *http.Request {
	return r.WithContext(middleware.NewContextWithOrgID(r.Context(), orgID))
}

func withClaims(r *http.Request, claims *appjwt.Claims) *http.Request {
	return r.WithContext(middleware.NewContextWithClaims(r.Context(), claims))
}

func testClaims() *appjwt.Claims {
	return &appjwt.Claims{UserID: "user-1", Username: "kasir1"}
}

// --- stubs ---

type stubRepo struct {
	sale *Sale
	err  error
}

func (s *stubRepo) Create(_ context.Context, sa *Sale) error {
	if s.err != nil {
		return s.err
	}
	sa.ID = "sale-1"
	sa.Status = StatusPending
	s.sale = sa
	return nil
}

func (s *stubRepo) FindByID(_ context.Context, _, _, id string) (*Sale, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.sale == nil {
		return nil, ErrNotFound
	}
	return s.sale, nil
}

func (s *stubRepo) FindItems(_ context.Context, _ string) ([]SaleItem, error) {
	if s.sale == nil {
		return nil, ErrNotFound
	}
	return s.sale.Items, nil
}

func (s *stubRepo) FindByStore(_ context.Context, _, _ string, _ Filter) ([]Sale, int, error) {
	if s.err != nil {
		return nil, 0, s.err
	}
	if s.sale != nil {
		return []Sale{*s.sale}, 1, nil
	}
	return []Sale{}, 0, nil
}

func (s *stubRepo) Cancel(_ context.Context, _, _, id string) (*Sale, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.sale == nil {
		return nil, ErrNotFound
	}
	if s.sale.Status != StatusPending {
		return nil, ErrInvalidStatus
	}
	s.sale.Status = StatusCancelled
	return s.sale, nil
}

type stubStores struct {
	found    bool
	assigned bool
}

func (s stubStores) FindByIDInOrg(_ context.Context, _, _ string) (*stores.Store, error) {
	if !s.found {
		return nil, stores.ErrNotFound
	}
	return &stores.Store{ID: "store-1", OrganizationID: "org-1"}, nil
}

func (s stubStores) IsAssigned(_ context.Context, _, _ string) (bool, error) {
	return s.assigned, nil
}

type stubProducts struct {
	found bool
	stock int
}

func (s stubProducts) FindByID(_ context.Context, _, _, _ string) (*products.Product, error) {
	if !s.found {
		return nil, products.ErrNotFound
	}
	return &products.Product{ID: "prod-1", StoreID: "store-1", Price: 10000, Stock: s.stock}, nil
}

func newHandler(repo *stubRepo, sc stubStores, pc stubProducts) *Handler {
	svc := NewService(repo, sc, pc, nil, nil)
	return NewHandler(svc)
}

func defaultHandler() (*Handler, *stubRepo) {
	repo := &stubRepo{}
	h := newHandler(repo, stubStores{found: true, assigned: true}, stubProducts{found: true, stock: 100})
	return h, repo
}

func createBody(t *testing.T) *bytes.Buffer {
	t.Helper()
	req := CreateRequest{
		PaymentMethod: "cash",
		Items:         []CreateItem{{ProductID: "prod-1", Quantity: 2}},
	}
	b, _ := json.Marshal(req)
	return bytes.NewBuffer(b)
}

// --- Create tests ---

func TestCreate_Success(t *testing.T) {
	h, _ := defaultHandler()
	r := httptest.NewRequest(http.MethodPost, "/sales", createBody(t))
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	var sale Sale
	if err := json.NewDecoder(w.Body).Decode(&sale); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if sale.ID == "" {
		t.Fatal("sale ID empty")
	}
}

func TestCreate_MissingOrgID(t *testing.T) {
	h, _ := defaultHandler()
	r := httptest.NewRequest(http.MethodPost, "/sales", createBody(t))
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_EmptyStoreID(t *testing.T) {
	h, _ := defaultHandler()
	r := httptest.NewRequest(http.MethodPost, "/sales", createBody(t))
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())

	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	h, _ := defaultHandler()
	r := httptest.NewRequest(http.MethodPost, "/sales", strings.NewReader("{bad"))
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreate_StoreNotFound(t *testing.T) {
	repo := &stubRepo{}
	h := newHandler(repo, stubStores{found: false, assigned: false}, stubProducts{found: true, stock: 100})
	r := httptest.NewRequest(http.MethodPost, "/sales", createBody(t))
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_NoAccess(t *testing.T) {
	repo := &stubRepo{}
	h := newHandler(repo, stubStores{found: true, assigned: false}, stubProducts{found: true, stock: 100})
	r := httptest.NewRequest(http.MethodPost, "/sales", createBody(t))
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_ProductNotFound(t *testing.T) {
	repo := &stubRepo{}
	h := newHandler(repo, stubStores{found: true, assigned: true}, stubProducts{found: false, stock: 0})
	r := httptest.NewRequest(http.MethodPost, "/sales", createBody(t))
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_InsufficientStock(t *testing.T) {
	repo := &stubRepo{}
	h := newHandler(repo, stubStores{found: true, assigned: true}, stubProducts{found: true, stock: 1})

	req := CreateRequest{
		PaymentMethod: "cash",
		Items:         []CreateItem{{ProductID: "prod-1", Quantity: 999}},
	}
	b, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/sales", bytes.NewBuffer(b))
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_InvalidInput_EmptyPaymentMethod(t *testing.T) {
	h, _ := defaultHandler()
	req := CreateRequest{
		PaymentMethod: "",
		Items:         []CreateItem{{ProductID: "prod-1", Quantity: 2}},
	}
	b, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/sales", bytes.NewBuffer(b))
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- List tests ---

func TestList_Success(t *testing.T) {
	h, repo := defaultHandler()
	repo.sale = &Sale{ID: "sale-1", Status: StatusPending}

	r := httptest.NewRequest(http.MethodGet, "/sales", nil)
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
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
		t.Fatalf("want total=1, got %d", resp.Total)
	}
}

func TestList_MissingOrgID(t *testing.T) {
	h, _ := defaultHandler()
	r := httptest.NewRequest(http.MethodGet, "/sales", nil)
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.List(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestList_EmptyStoreID(t *testing.T) {
	h, _ := defaultHandler()
	r := httptest.NewRequest(http.MethodGet, "/sales", nil)
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())

	w := httptest.NewRecorder()
	h.List(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestList_InvalidDateFilter(t *testing.T) {
	h, _ := defaultHandler()
	r := httptest.NewRequest(http.MethodGet, "/sales?from=not-a-date", nil)
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.List(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestList_EmptyResult(t *testing.T) {
	h, _ := defaultHandler()
	r := httptest.NewRequest(http.MethodGet, "/sales", nil)
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
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
	if len(resp.Data) != 0 {
		t.Fatalf("want empty data, got %d", len(resp.Data))
	}
}

// --- ByID tests ---

func TestByID_Success(t *testing.T) {
	h, repo := defaultHandler()
	repo.sale = &Sale{ID: "sale-1", Status: StatusPending}

	r := httptest.NewRequest(http.MethodGet, "/sales/sale-1", nil)
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "sale-1")

	w := httptest.NewRecorder()
	h.ByID(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var sale Sale
	if err := json.NewDecoder(w.Body).Decode(&sale); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if sale.ID != "sale-1" {
		t.Fatalf("want sale-1, got %s", sale.ID)
	}
}

func TestByID_MissingOrgID(t *testing.T) {
	h, _ := defaultHandler()
	r := httptest.NewRequest(http.MethodGet, "/sales/sale-1", nil)
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "sale-1")

	w := httptest.NewRecorder()
	h.ByID(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestByID_MissingIDs(t *testing.T) {
	h, _ := defaultHandler()
	r := httptest.NewRequest(http.MethodGet, "/sales/", nil)
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.ByID(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestByID_NotFound(t *testing.T) {
	h, _ := defaultHandler()

	r := httptest.NewRequest(http.MethodGet, "/sales/nope", nil)
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "nope")

	w := httptest.NewRecorder()
	h.ByID(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Cancel tests ---

func TestCancel_Success(t *testing.T) {
	h, repo := defaultHandler()
	repo.sale = &Sale{ID: "sale-1", Status: StatusPending}

	r := httptest.NewRequest(http.MethodPatch, "/sales/sale-1/cancel", nil)
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "sale-1")

	w := httptest.NewRecorder()
	h.Cancel(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var sale Sale
	if err := json.NewDecoder(w.Body).Decode(&sale); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if sale.Status != StatusCancelled {
		t.Fatalf("want cancelled, got %s", sale.Status)
	}
}

func TestCancel_MissingOrgID(t *testing.T) {
	h, _ := defaultHandler()
	r := httptest.NewRequest(http.MethodPatch, "/sales/sale-1/cancel", nil)
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "sale-1")

	w := httptest.NewRecorder()
	h.Cancel(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCancel_MissingIDs(t *testing.T) {
	h, _ := defaultHandler()
	r := httptest.NewRequest(http.MethodPatch, "/sales//cancel", nil)
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")

	w := httptest.NewRecorder()
	h.Cancel(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCancel_NotFound(t *testing.T) {
	h, _ := defaultHandler()

	r := httptest.NewRequest(http.MethodPatch, "/sales/nope/cancel", nil)
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "nope")

	w := httptest.NewRecorder()
	h.Cancel(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCancel_AlreadyCancelled(t *testing.T) {
	h, repo := defaultHandler()
	repo.sale = &Sale{ID: "sale-1", Status: StatusCancelled}

	r := httptest.NewRequest(http.MethodPatch, "/sales/sale-1/cancel", nil)
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "sale-1")

	w := httptest.NewRecorder()
	h.Cancel(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCancel_AlreadyCompleted(t *testing.T) {
	h, repo := defaultHandler()
	repo.sale = &Sale{ID: "sale-1", Status: StatusCompleted}

	r := httptest.NewRequest(http.MethodPatch, "/sales/sale-1/cancel", nil)
	r = withOrgID(r, "org-1")
	r = withClaims(r, testClaims())
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("id", "sale-1")

	w := httptest.NewRecorder()
	h.Cancel(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d: %s", w.Code, w.Body.String())
	}
}
