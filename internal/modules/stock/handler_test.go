package stock

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

// --- stubs ---

type stubRepo struct {
	recordFn      func(ctx context.Context, m *StockMovement, allowNeg bool) error
	findByStoreFn func(ctx context.Context, orgID, storeID string, f Filter) ([]StockMovement, int, error)
	findByProdFn  func(ctx context.Context, orgID, storeID, prodID string) ([]StockMovement, error)
	getSummaryFn  func(ctx context.Context, orgID, storeID, prodID string) (*StockSummary, error)
}

func (s *stubRepo) Record(ctx context.Context, m *StockMovement, allowNeg bool) error {
	return s.recordFn(ctx, m, allowNeg)
}
func (s *stubRepo) FindByStore(ctx context.Context, orgID, storeID string, f Filter) ([]StockMovement, int, error) {
	return s.findByStoreFn(ctx, orgID, storeID, f)
}
func (s *stubRepo) FindByProduct(ctx context.Context, orgID, storeID, prodID string) ([]StockMovement, error) {
	return s.findByProdFn(ctx, orgID, storeID, prodID)
}
func (s *stubRepo) GetSummary(ctx context.Context, orgID, storeID, prodID string) (*StockSummary, error) {
	return s.getSummaryFn(ctx, orgID, storeID, prodID)
}

type stubStoreChecker struct {
	findFn     func(ctx context.Context, orgID, id string) (*stores.Store, error)
	assignedFn func(ctx context.Context, userID, storeID string) (bool, error)
}

func (s *stubStoreChecker) FindByIDInOrg(ctx context.Context, orgID, id string) (*stores.Store, error) {
	return s.findFn(ctx, orgID, id)
}
func (s *stubStoreChecker) IsAssigned(ctx context.Context, userID, storeID string) (bool, error) {
	return s.assignedFn(ctx, userID, storeID)
}

// --- helpers ---

func okStoreChecker() *stubStoreChecker {
	return &stubStoreChecker{
		findFn:     func(context.Context, string, string) (*stores.Store, error) { return &stores.Store{}, nil },
		assignedFn: func(context.Context, string, string) (bool, error) { return true, nil },
	}
}

func newTestHandler(repo *stubRepo, sc *stubStoreChecker) *Handler {
	return NewHandler(NewService(repo, sc))
}

func decodeJSON(t *testing.T, body *bytes.Buffer, v any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func defaultClaims() *appjwt.Claims {
	return &appjwt.Claims{UserID: "user-1"}
}

func withCtx(r *http.Request, orgID string, claims *appjwt.Claims) *http.Request {
	ctx := middleware.NewContextWithOrgID(r.Context(), orgID)
	ctx = middleware.NewContextWithClaims(ctx, claims)
	return r.WithContext(ctx)
}

func withOrgOnly(r *http.Request, orgID string) *http.Request {
	return r.WithContext(middleware.NewContextWithOrgID(r.Context(), orgID))
}

// ==================== Record ====================

func TestHandler_Record(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			recordFn: func(_ context.Context, m *StockMovement, _ bool) error {
				m.ID = "mov-1"
				m.StockBefore = 10
				m.StockAfter = 15
				return nil
			},
		}
		h := newTestHandler(repo, okStoreChecker())

		body, _ := json.Marshal(RecordRequest{
			ProductID: "prod-1",
			Type:      TypeIn,
			Quantity:  5,
		})
		r := httptest.NewRequest(http.MethodPost, "/org/x/stores/store-1/stock-movements", bytes.NewReader(body))
		r.SetPathValue("storeId", "store-1")
		r = withCtx(r, "org-1", defaultClaims())

		rr := httptest.NewRecorder()
		h.Record(rr, r)

		if rr.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusCreated)
		}
		var got StockMovement
		decodeJSON(t, rr.Body, &got)
		if got.ID != "mov-1" {
			t.Fatalf("ID = %q, want mov-1", got.ID)
		}
	})

	t.Run("missing org context", func(t *testing.T) {
		h := newTestHandler(&stubRepo{}, okStoreChecker())
		body, _ := json.Marshal(RecordRequest{ProductID: "p", Type: TypeIn, Quantity: 1})
		r := httptest.NewRequest(http.MethodPost, "/org/x/stores/s1/stock-movements", bytes.NewReader(body))
		r.SetPathValue("storeId", "s1")
		// no org context injected
		rr := httptest.NewRecorder()
		h.Record(rr, r)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("empty storeId", func(t *testing.T) {
		h := newTestHandler(&stubRepo{}, okStoreChecker())
		body, _ := json.Marshal(RecordRequest{ProductID: "p", Type: TypeIn, Quantity: 1})
		r := httptest.NewRequest(http.MethodPost, "/org/x/stores//stock-movements", bytes.NewReader(body))
		r.SetPathValue("storeId", "")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.Record(rr, r)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		h := newTestHandler(&stubRepo{}, okStoreChecker())
		r := httptest.NewRequest(http.MethodPost, "/org/x/stores/s1/stock-movements", bytes.NewReader([]byte("{bad")))
		r.SetPathValue("storeId", "s1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.Record(rr, r)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid input from service", func(t *testing.T) {
		repo := &stubRepo{
			recordFn: func(context.Context, *StockMovement, bool) error { return nil },
		}
		h := newTestHandler(repo, okStoreChecker())
		body, _ := json.Marshal(RecordRequest{ProductID: "", Type: TypeIn, Quantity: 1})
		r := httptest.NewRequest(http.MethodPost, "/org/x/stores/s1/stock-movements", bytes.NewReader(body))
		r.SetPathValue("storeId", "s1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.Record(rr, r)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		repo := &stubRepo{
			recordFn: func(context.Context, *StockMovement, bool) error { return nil },
		}
		h := newTestHandler(repo, okStoreChecker())
		body, _ := json.Marshal(RecordRequest{ProductID: "p1", Type: "badtype", Quantity: 1})
		r := httptest.NewRequest(http.MethodPost, "/org/x/stores/s1/stock-movements", bytes.NewReader(body))
		r.SetPathValue("storeId", "s1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.Record(rr, r)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("insufficient stock", func(t *testing.T) {
		repo := &stubRepo{
			recordFn: func(_ context.Context, _ *StockMovement, _ bool) error { return ErrInsufficient },
		}
		h := newTestHandler(repo, okStoreChecker())
		body, _ := json.Marshal(RecordRequest{ProductID: "p1", Type: TypeOut, Quantity: 5})
		r := httptest.NewRequest(http.MethodPost, "/org/x/stores/s1/stock-movements", bytes.NewReader(body))
		r.SetPathValue("storeId", "s1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.Record(rr, r)
		if rr.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
		}
	})

	t.Run("store not found", func(t *testing.T) {
		sc := &stubStoreChecker{
			findFn:     func(context.Context, string, string) (*stores.Store, error) { return nil, stores.ErrNotFound },
			assignedFn: func(context.Context, string, string) (bool, error) { return false, nil },
		}
		repo := &stubRepo{
			recordFn: func(context.Context, *StockMovement, bool) error { return nil },
		}
		h := newTestHandler(repo, sc)
		body, _ := json.Marshal(RecordRequest{ProductID: "p1", Type: TypeIn, Quantity: 1})
		r := httptest.NewRequest(http.MethodPost, "/org/x/stores/s1/stock-movements", bytes.NewReader(body))
		r.SetPathValue("storeId", "s1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.Record(rr, r)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("no access", func(t *testing.T) {
		sc := &stubStoreChecker{
			findFn:     func(context.Context, string, string) (*stores.Store, error) { return &stores.Store{}, nil },
			assignedFn: func(context.Context, string, string) (bool, error) { return false, nil },
		}
		repo := &stubRepo{
			recordFn: func(context.Context, *StockMovement, bool) error { return nil },
		}
		h := newTestHandler(repo, sc)
		body, _ := json.Marshal(RecordRequest{ProductID: "p1", Type: TypeIn, Quantity: 1})
		r := httptest.NewRequest(http.MethodPost, "/org/x/stores/s1/stock-movements", bytes.NewReader(body))
		r.SetPathValue("storeId", "s1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.Record(rr, r)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
		}
	})

	t.Run("repo error 500", func(t *testing.T) {
		repo := &stubRepo{
			recordFn: func(context.Context, *StockMovement, bool) error { return errors.New("db down") },
		}
		h := newTestHandler(repo, okStoreChecker())
		body, _ := json.Marshal(RecordRequest{ProductID: "p1", Type: TypeIn, Quantity: 1})
		r := httptest.NewRequest(http.MethodPost, "/org/x/stores/s1/stock-movements", bytes.NewReader(body))
		r.SetPathValue("storeId", "s1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.Record(rr, r)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})
}

// ==================== List ====================

func TestHandler_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			findByStoreFn: func(_ context.Context, _, _ string, _ Filter) ([]StockMovement, int, error) {
				return []StockMovement{{ID: "m1"}}, 1, nil
			},
		}
		h := newTestHandler(repo, okStoreChecker())

		r := httptest.NewRequest(http.MethodGet, "/org/x/stores/s1/stock-movements?page=1&limit=20", nil)
		r.SetPathValue("storeId", "s1")
		r = withCtx(r, "org-1", defaultClaims())

		rr := httptest.NewRecorder()
		h.List(rr, r)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var got ListResponse
		decodeJSON(t, rr.Body, &got)
		if got.Total != 1 || len(got.Data) != 1 {
			t.Fatalf("response = %+v, want 1 item", got)
		}
	})

	t.Run("missing org", func(t *testing.T) {
		h := newTestHandler(&stubRepo{}, okStoreChecker())
		r := httptest.NewRequest(http.MethodGet, "/org/x/stores/s1/stock-movements", nil)
		r.SetPathValue("storeId", "s1")
		rr := httptest.NewRecorder()
		h.List(rr, r)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("empty storeId", func(t *testing.T) {
		h := newTestHandler(&stubRepo{}, okStoreChecker())
		r := httptest.NewRequest(http.MethodGet, "/org/x/stores//stock-movements", nil)
		r.SetPathValue("storeId", "")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.List(rr, r)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("bad date filter", func(t *testing.T) {
		h := newTestHandler(&stubRepo{}, okStoreChecker())
		r := httptest.NewRequest(http.MethodGet, "/org/x/stores/s1/stock-movements?from=not-a-date", nil)
		r.SetPathValue("storeId", "s1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.List(rr, r)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("store not found", func(t *testing.T) {
		sc := &stubStoreChecker{
			findFn:     func(context.Context, string, string) (*stores.Store, error) { return nil, stores.ErrNotFound },
			assignedFn: func(context.Context, string, string) (bool, error) { return false, nil },
		}
		repo := &stubRepo{
			findByStoreFn: func(context.Context, string, string, Filter) ([]StockMovement, int, error) {
				return nil, 0, nil
			},
		}
		h := newTestHandler(repo, sc)
		r := httptest.NewRequest(http.MethodGet, "/org/x/stores/s1/stock-movements", nil)
		r.SetPathValue("storeId", "s1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.List(rr, r)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("no access", func(t *testing.T) {
		sc := &stubStoreChecker{
			findFn:     func(context.Context, string, string) (*stores.Store, error) { return &stores.Store{}, nil },
			assignedFn: func(context.Context, string, string) (bool, error) { return false, nil },
		}
		repo := &stubRepo{
			findByStoreFn: func(context.Context, string, string, Filter) ([]StockMovement, int, error) {
				return nil, 0, nil
			},
		}
		h := newTestHandler(repo, sc)
		r := httptest.NewRequest(http.MethodGet, "/org/x/stores/s1/stock-movements", nil)
		r.SetPathValue("storeId", "s1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.List(rr, r)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
		}
	})

	t.Run("repo error 500", func(t *testing.T) {
		repo := &stubRepo{
			findByStoreFn: func(context.Context, string, string, Filter) ([]StockMovement, int, error) {
				return nil, 0, errors.New("db down")
			},
		}
		h := newTestHandler(repo, okStoreChecker())
		r := httptest.NewRequest(http.MethodGet, "/org/x/stores/s1/stock-movements", nil)
		r.SetPathValue("storeId", "s1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.List(rr, r)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})

	t.Run("nil list returns empty array", func(t *testing.T) {
		repo := &stubRepo{
			findByStoreFn: func(context.Context, string, string, Filter) ([]StockMovement, int, error) {
				return nil, 0, nil
			},
		}
		h := newTestHandler(repo, okStoreChecker())
		r := httptest.NewRequest(http.MethodGet, "/org/x/stores/s1/stock-movements?page=1&limit=10", nil)
		r.SetPathValue("storeId", "s1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.List(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var got ListResponse
		decodeJSON(t, rr.Body, &got)
		if got.Data == nil {
			t.Fatal("Data should be empty slice, not nil")
		}
		if len(got.Data) != 0 {
			t.Fatalf("len(Data) = %d, want 0", len(got.Data))
		}
	})
}

// ==================== StockByProduct ====================

func TestHandler_StockByProduct(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubRepo{
			getSummaryFn: func(context.Context, string, string, string) (*StockSummary, error) {
				return &StockSummary{ProductID: "p1", ProductStock: 42}, nil
			},
			findByProdFn: func(context.Context, string, string, string) ([]StockMovement, error) {
				return []StockMovement{{ID: "m1"}}, nil
			},
		}
		h := newTestHandler(repo, okStoreChecker())

		r := httptest.NewRequest(http.MethodGet, "/org/x/stores/s1/products/p1/stock", nil)
		r.SetPathValue("storeId", "s1")
		r.SetPathValue("productId", "p1")
		r = withCtx(r, "org-1", defaultClaims())

		rr := httptest.NewRecorder()
		h.StockByProduct(rr, r)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var got StockResponse
		decodeJSON(t, rr.Body, &got)
		if got.Summary.ProductStock != 42 {
			t.Fatalf("stock = %d, want 42", got.Summary.ProductStock)
		}
		if len(got.History) != 1 {
			t.Fatalf("history len = %d, want 1", len(got.History))
		}
	})

	t.Run("missing org", func(t *testing.T) {
		h := newTestHandler(&stubRepo{}, okStoreChecker())
		r := httptest.NewRequest(http.MethodGet, "/org/x/stores/s1/products/p1/stock", nil)
		r.SetPathValue("storeId", "s1")
		r.SetPathValue("productId", "p1")
		rr := httptest.NewRecorder()
		h.StockByProduct(rr, r)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("empty storeId or productId", func(t *testing.T) {
		h := newTestHandler(&stubRepo{}, okStoreChecker())
		r := httptest.NewRequest(http.MethodGet, "/org/x/stores//products//stock", nil)
		r.SetPathValue("storeId", "")
		r.SetPathValue("productId", "")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.StockByProduct(rr, r)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("product not found", func(t *testing.T) {
		repo := &stubRepo{
			getSummaryFn: func(context.Context, string, string, string) (*StockSummary, error) {
				return nil, ErrProductNotFound
			},
			findByProdFn: func(context.Context, string, string, string) ([]StockMovement, error) {
				return nil, nil
			},
		}
		h := newTestHandler(repo, okStoreChecker())
		r := httptest.NewRequest(http.MethodGet, "/org/x/stores/s1/products/p1/stock", nil)
		r.SetPathValue("storeId", "s1")
		r.SetPathValue("productId", "p1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.StockByProduct(rr, r)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("store not found", func(t *testing.T) {
		sc := &stubStoreChecker{
			findFn:     func(context.Context, string, string) (*stores.Store, error) { return nil, stores.ErrNotFound },
			assignedFn: func(context.Context, string, string) (bool, error) { return false, nil },
		}
		repo := &stubRepo{
			getSummaryFn: func(context.Context, string, string, string) (*StockSummary, error) { return nil, nil },
			findByProdFn: func(context.Context, string, string, string) ([]StockMovement, error) { return nil, nil },
		}
		h := newTestHandler(repo, sc)
		r := httptest.NewRequest(http.MethodGet, "/org/x/stores/s1/products/p1/stock", nil)
		r.SetPathValue("storeId", "s1")
		r.SetPathValue("productId", "p1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.StockByProduct(rr, r)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("no access", func(t *testing.T) {
		sc := &stubStoreChecker{
			findFn:     func(context.Context, string, string) (*stores.Store, error) { return &stores.Store{}, nil },
			assignedFn: func(context.Context, string, string) (bool, error) { return false, nil },
		}
		repo := &stubRepo{
			getSummaryFn: func(context.Context, string, string, string) (*StockSummary, error) { return nil, nil },
			findByProdFn: func(context.Context, string, string, string) ([]StockMovement, error) { return nil, nil },
		}
		h := newTestHandler(repo, sc)
		r := httptest.NewRequest(http.MethodGet, "/org/x/stores/s1/products/p1/stock", nil)
		r.SetPathValue("storeId", "s1")
		r.SetPathValue("productId", "p1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.StockByProduct(rr, r)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
		}
	})

	t.Run("repo error 500", func(t *testing.T) {
		repo := &stubRepo{
			getSummaryFn: func(context.Context, string, string, string) (*StockSummary, error) {
				return nil, errors.New("db down")
			},
			findByProdFn: func(context.Context, string, string, string) ([]StockMovement, error) {
				return nil, nil
			},
		}
		h := newTestHandler(repo, okStoreChecker())
		r := httptest.NewRequest(http.MethodGet, "/org/x/stores/s1/products/p1/stock", nil)
		r.SetPathValue("storeId", "s1")
		r.SetPathValue("productId", "p1")
		r = withCtx(r, "org-1", defaultClaims())
		rr := httptest.NewRecorder()
		h.StockByProduct(rr, r)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})
}
