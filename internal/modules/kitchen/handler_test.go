package kitchen

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
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"

	"github.com/ramadhanrzq/backend-go/internal/modules/sales"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

func withOrgID(r *http.Request, orgID string) *http.Request {
	return r.WithContext(middleware.NewContextWithOrgID(r.Context(), orgID))
}

func withClaims(r *http.Request, claims *appjwt.Claims) *http.Request {
	return r.WithContext(middleware.NewContextWithClaims(r.Context(), claims))
}

func withAuth(r *http.Request) *http.Request {
	r = withOrgID(r, "org-1")
	return withClaims(r, &appjwt.Claims{UserID: "user-1"})
}

// ---- stubs ----

type stubRepo struct {
	queue         []KitchenSale
	queueErr      error
	sale          *KitchenSale
	findErr       error
	updated       *KitchenSale
	updateErr     error
	updatedItem   *KitchenSaleItem
	updateItemErr error
}

func (s *stubRepo) Create(context.Context, *KitchenSale) error { return nil }
func (s *stubRepo) FindByID(context.Context, string, string, string) (*KitchenSale, error) {
	return s.sale, s.findErr
}
func (s *stubRepo) FindQueue(context.Context, string, string) ([]KitchenSale, error) {
	return s.queue, s.queueErr
}
func (s *stubRepo) FindBySaleID(context.Context, string) (*KitchenSale, error) { return nil, nil }
func (s *stubRepo) CancelBySale(context.Context, string, string, string) (int64, error) {
	return 0, nil
}
func (s *stubRepo) UpdateStatus(context.Context, string, string, string, string, *time.Time, *time.Time) (*KitchenSale, error) {
	return s.updated, s.updateErr
}
func (s *stubRepo) UpdateItemStatus(context.Context, string, string, string, string, string) (*KitchenSaleItem, error) {
	return s.updatedItem, s.updateItemErr
}
func (s *stubRepo) FindItems(context.Context, string) ([]KitchenSaleItem, error) { return nil, nil }

type stubStoreChecker struct {
	store     *stores.Store
	findErr   error
	assigned  bool
	assignErr error
}

func (s *stubStoreChecker) FindByIDInOrg(_ context.Context, _, _ string) (*stores.Store, error) {
	return s.store, s.findErr
}
func (s *stubStoreChecker) IsAssigned(_ context.Context, _, _ string) (bool, error) {
	return s.assigned, s.assignErr
}

type stubSaleChecker struct {
	sale    *sales.Sale
	findErr error
}

func (s *stubSaleChecker) FindByID(_ context.Context, _, _, _ string) (*sales.Sale, error) {
	return s.sale, s.findErr
}

// ---- helpers ----

func setupHandler(repo *stubRepo, sc *stubStoreChecker, salec *stubSaleChecker) *Handler {
	svc := NewService(repo, sc, salec)
	return NewHandler(svc)
}

func defaultStoreChecker() *stubStoreChecker {
	return &stubStoreChecker{
		store:    &stores.Store{ID: "store-1"},
		assigned: true,
	}
}

func defaultSaleChecker() *stubSaleChecker {
	return &stubSaleChecker{
		sale: &sales.Sale{ID: "sale-1", Status: "open"},
	}
}

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(b)
}

// ---- Queue tests ----

func TestQueue_Success(t *testing.T) {
	repo := &stubRepo{queue: []KitchenSale{{ID: "ks-1", Status: StatusPending}}}
	h := setupHandler(repo, defaultStoreChecker(), defaultSaleChecker())

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Queue(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp QueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 || resp.Data[0].ID != "ks-1" {
		t.Fatalf("unexpected data: %+v", resp.Data)
	}
}

func TestQueue_NilBecomesEmptySlice(t *testing.T) {
	repo := &stubRepo{queue: nil}
	h := setupHandler(repo, defaultStoreChecker(), defaultSaleChecker())

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Queue(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp QueueResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Data == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(resp.Data) != 0 {
		t.Fatalf("expected empty, got %d items", len(resp.Data))
	}
}

func TestQueue_MissingOrgID(t *testing.T) {
	h := setupHandler(&stubRepo{}, defaultStoreChecker(), defaultSaleChecker())

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	// no org_id in context
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Queue(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestQueue_EmptyStoreID(t *testing.T) {
	h := setupHandler(&stubRepo{}, defaultStoreChecker(), defaultSaleChecker())

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withAuth(r)
	// storeId not set → empty
	w := httptest.NewRecorder()

	h.Queue(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestQueue_StoreNotFound(t *testing.T) {
	sc := &stubStoreChecker{findErr: stores.ErrNotFound}
	h := setupHandler(&stubRepo{}, sc, defaultSaleChecker())

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Queue(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestQueue_NoAccess(t *testing.T) {
	sc := &stubStoreChecker{store: &stores.Store{ID: "store-1"}, assigned: false}
	h := setupHandler(&stubRepo{}, sc, defaultSaleChecker())

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Queue(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w.Code)
	}
}

func TestQueue_InternalError(t *testing.T) {
	repo := &stubRepo{queueErr: errors.New("db down")}
	h := setupHandler(repo, defaultStoreChecker(), defaultSaleChecker())

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.Queue(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

// ---- ByID tests ----

func TestByID_Success(t *testing.T) {
	ks := &KitchenSale{ID: "ks-1", Status: StatusPending}
	repo := &stubRepo{sale: ks}
	h := setupHandler(repo, defaultStoreChecker(), defaultSaleChecker())

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("kitchenSaleId", "ks-1")
	w := httptest.NewRecorder()

	h.ByID(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var got KitchenSale
	json.NewDecoder(w.Body).Decode(&got)
	if got.ID != "ks-1" {
		t.Fatalf("want ks-1, got %s", got.ID)
	}
}

func TestByID_MissingOrgID(t *testing.T) {
	h := setupHandler(&stubRepo{}, defaultStoreChecker(), defaultSaleChecker())

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("kitchenSaleId", "ks-1")
	w := httptest.NewRecorder()

	h.ByID(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestByID_EmptyIDs(t *testing.T) {
	h := setupHandler(&stubRepo{}, defaultStoreChecker(), defaultSaleChecker())

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withAuth(r)
	// storeId set, kitchenSaleId empty
	r.SetPathValue("storeId", "store-1")
	w := httptest.NewRecorder()

	h.ByID(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestByID_NotFound(t *testing.T) {
	repo := &stubRepo{findErr: ErrNotFound}
	h := setupHandler(repo, defaultStoreChecker(), defaultSaleChecker())

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("kitchenSaleId", "ks-1")
	w := httptest.NewRecorder()

	h.ByID(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestByID_NoAccess(t *testing.T) {
	sc := &stubStoreChecker{store: &stores.Store{ID: "store-1"}, assigned: false}
	h := setupHandler(&stubRepo{sale: &KitchenSale{ID: "ks-1"}}, sc, defaultSaleChecker())

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("kitchenSaleId", "ks-1")
	w := httptest.NewRecorder()

	h.ByID(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w.Code)
	}
}

// ---- UpdateStatus tests ----

func TestUpdateStatus_Success(t *testing.T) {
	ks := &KitchenSale{ID: "ks-1", SaleID: "sale-1", Status: StatusPreparing}
	repo := &stubRepo{
		sale:    &KitchenSale{ID: "ks-1", SaleID: "sale-1", Status: StatusPending},
		updated: ks,
	}
	h := setupHandler(repo, defaultStoreChecker(), defaultSaleChecker())

	body := jsonBody(t, UpdateStatusRequest{Status: StatusPreparing})
	r := httptest.NewRequest(http.MethodPatch, "/", body)
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("kitchenSaleId", "ks-1")
	w := httptest.NewRecorder()

	h.UpdateStatus(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var got KitchenSale
	json.NewDecoder(w.Body).Decode(&got)
	if got.Status != StatusPreparing {
		t.Fatalf("want preparing, got %s", got.Status)
	}
}

func TestUpdateStatus_MissingOrgID(t *testing.T) {
	h := setupHandler(&stubRepo{}, defaultStoreChecker(), defaultSaleChecker())

	body := jsonBody(t, UpdateStatusRequest{Status: StatusPreparing})
	r := httptest.NewRequest(http.MethodPatch, "/", body)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("kitchenSaleId", "ks-1")
	w := httptest.NewRecorder()

	h.UpdateStatus(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdateStatus_EmptyIDs(t *testing.T) {
	h := setupHandler(&stubRepo{}, defaultStoreChecker(), defaultSaleChecker())

	body := jsonBody(t, UpdateStatusRequest{Status: StatusPreparing})
	r := httptest.NewRequest(http.MethodPatch, "/", body)
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	// kitchenSaleId not set
	w := httptest.NewRecorder()

	h.UpdateStatus(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdateStatus_InvalidJSON(t *testing.T) {
	h := setupHandler(&stubRepo{}, defaultStoreChecker(), defaultSaleChecker())

	r := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader([]byte("{bad")))
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("kitchenSaleId", "ks-1")
	w := httptest.NewRecorder()

	h.UpdateStatus(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdateStatus_ErrorMappings(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect int
	}{
		{"InvalidStatus", ErrInvalidStatus, http.StatusBadRequest},
		{"InvalidInput", ErrInvalidInput, http.StatusBadRequest},
		{"InvalidTransition", ErrInvalidTransition, http.StatusConflict},
		{"SaleCancelled", ErrSaleCancelled, http.StatusConflict},
		{"SaleNotFound", ErrSaleNotFound, http.StatusNotFound},
		{"StoreNotFound", ErrStoreNotFound, http.StatusNotFound},
		{"NotFound", ErrNotFound, http.StatusNotFound},
		{"NoAccess", ErrNoAccess, http.StatusForbidden},
		{"Unknown", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Need the service to reach svc.UpdateStatus, which calls authorize then repo.
			// Easiest: stub the store checker to pass, stub repo FindByID to return a sale,
			// then stub repo UpdateStatus to return the error.
			repo := &stubRepo{
				sale:      &KitchenSale{ID: "ks-1", SaleID: "sale-1", Status: StatusPending},
				updateErr: tt.err,
			}
			sc := defaultStoreChecker()
			salec := defaultSaleChecker()

			// For errors that happen before repo.UpdateStatus (authorize phase),
			// we need to set them at the right stub level.
			switch tt.err {
			case ErrStoreNotFound:
				sc = &stubStoreChecker{findErr: stores.ErrNotFound}
				repo.sale = nil
				repo.updateErr = nil
			case ErrNoAccess:
				sc = &stubStoreChecker{store: &stores.Store{ID: "store-1"}, assigned: false}
				repo.sale = nil
				repo.updateErr = nil
			case ErrNotFound:
				repo.sale = nil
				repo.findErr = ErrNotFound
				repo.updateErr = nil
			case ErrSaleNotFound:
				repo.sale = &KitchenSale{ID: "ks-1", SaleID: "sale-1", Status: StatusPending}
				salec = &stubSaleChecker{findErr: sales.ErrNotFound}
				repo.updateErr = nil
			case ErrSaleCancelled:
				repo.sale = &KitchenSale{ID: "ks-1", SaleID: "sale-1", Status: StatusPending}
				salec = &stubSaleChecker{sale: &sales.Sale{ID: "sale-1", Status: "cancelled"}}
				repo.updateErr = nil
			}

			h := setupHandler(repo, sc, salec)

			body := jsonBody(t, UpdateStatusRequest{Status: StatusPreparing})
			r := httptest.NewRequest(http.MethodPatch, "/", body)
			r = withAuth(r)
			r.SetPathValue("storeId", "store-1")
			r.SetPathValue("kitchenSaleId", "ks-1")
			w := httptest.NewRecorder()

			h.UpdateStatus(w, r)

			if w.Code != tt.expect {
				t.Fatalf("want %d, got %d: %s", tt.expect, w.Code, w.Body.String())
			}
		})
	}
}

// ---- UpdateItemStatus tests ----

func TestUpdateItemStatus_Success(t *testing.T) {
	item := &KitchenSaleItem{ID: "item-1", KitchenSaleID: "ks-1", Status: StatusPreparing}
	repo := &stubRepo{
		sale: &KitchenSale{
			ID: "ks-1", SaleID: "sale-1", Status: StatusPreparing,
			Items: []KitchenSaleItem{{ID: "item-1", KitchenSaleID: "ks-1", Status: StatusPending}},
		},
		updatedItem: item,
	}
	h := setupHandler(repo, defaultStoreChecker(), defaultSaleChecker())

	body := jsonBody(t, UpdateStatusRequest{Status: StatusPreparing})
	r := httptest.NewRequest(http.MethodPatch, "/", body)
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("kitchenSaleId", "ks-1")
	r.SetPathValue("itemId", "item-1")
	w := httptest.NewRecorder()

	h.UpdateItemStatus(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var got KitchenSaleItem
	json.NewDecoder(w.Body).Decode(&got)
	if got.ID != "item-1" {
		t.Fatalf("want item-1, got %s", got.ID)
	}
}

func TestUpdateItemStatus_MissingOrgID(t *testing.T) {
	h := setupHandler(&stubRepo{}, defaultStoreChecker(), defaultSaleChecker())

	body := jsonBody(t, UpdateStatusRequest{Status: StatusPreparing})
	r := httptest.NewRequest(http.MethodPatch, "/", body)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("kitchenSaleId", "ks-1")
	r.SetPathValue("itemId", "item-1")
	w := httptest.NewRecorder()

	h.UpdateItemStatus(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdateItemStatus_EmptyIDs(t *testing.T) {
	h := setupHandler(&stubRepo{}, defaultStoreChecker(), defaultSaleChecker())

	body := jsonBody(t, UpdateStatusRequest{Status: StatusPreparing})
	r := httptest.NewRequest(http.MethodPatch, "/", body)
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("kitchenSaleId", "ks-1")
	// itemId not set → empty
	w := httptest.NewRecorder()

	h.UpdateItemStatus(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdateItemStatus_InvalidJSON(t *testing.T) {
	h := setupHandler(&stubRepo{}, defaultStoreChecker(), defaultSaleChecker())

	r := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader([]byte("nope")))
	r = withAuth(r)
	r.SetPathValue("storeId", "store-1")
	r.SetPathValue("kitchenSaleId", "ks-1")
	r.SetPathValue("itemId", "item-1")
	w := httptest.NewRecorder()

	h.UpdateItemStatus(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestUpdateItemStatus_ErrorMappings(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect int
	}{
		{"InvalidStatus", ErrInvalidStatus, http.StatusBadRequest},
		{"InvalidInput", ErrInvalidInput, http.StatusBadRequest},
		{"InvalidTransition", ErrInvalidTransition, http.StatusConflict},
		{"SaleCancelled", ErrSaleCancelled, http.StatusConflict},
		{"SaleNotFound", ErrSaleNotFound, http.StatusNotFound},
		{"StoreNotFound", ErrStoreNotFound, http.StatusNotFound},
		{"NotFound", ErrNotFound, http.StatusNotFound},
		{"NoAccess", ErrNoAccess, http.StatusForbidden},
		{"Unknown", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks := &KitchenSale{
				ID: "ks-1", SaleID: "sale-1", Status: StatusPreparing,
				Items: []KitchenSaleItem{{ID: "item-1", KitchenSaleID: "ks-1", Status: StatusPending}},
			}
			repo := &stubRepo{
				sale:          ks,
				updateItemErr: tt.err,
			}
			sc := defaultStoreChecker()
			salec := defaultSaleChecker()

			switch tt.err {
			case ErrStoreNotFound:
				sc = &stubStoreChecker{findErr: stores.ErrNotFound}
				repo.sale = nil
				repo.updateItemErr = nil
			case ErrNoAccess:
				sc = &stubStoreChecker{store: &stores.Store{ID: "store-1"}, assigned: false}
				repo.sale = nil
				repo.updateItemErr = nil
			case ErrNotFound:
				repo.sale = nil
				repo.findErr = ErrNotFound
				repo.updateItemErr = nil
			case ErrSaleNotFound:
				repo.sale = &KitchenSale{
					ID: "ks-1", SaleID: "sale-1", Status: StatusPreparing,
					Items: []KitchenSaleItem{{ID: "item-1", KitchenSaleID: "ks-1", Status: StatusPending}},
				}
				salec = &stubSaleChecker{findErr: sales.ErrNotFound}
				repo.updateItemErr = nil
			case ErrSaleCancelled:
				repo.sale = &KitchenSale{
					ID: "ks-1", SaleID: "sale-1", Status: StatusPreparing,
					Items: []KitchenSaleItem{{ID: "item-1", KitchenSaleID: "ks-1", Status: StatusPending}},
				}
				salec = &stubSaleChecker{sale: &sales.Sale{ID: "sale-1", Status: "cancelled"}}
				repo.updateItemErr = nil
			}

			h := setupHandler(repo, sc, salec)

			body := jsonBody(t, UpdateStatusRequest{Status: StatusPreparing})
			r := httptest.NewRequest(http.MethodPatch, "/", body)
			r = withAuth(r)
			r.SetPathValue("storeId", "store-1")
			r.SetPathValue("kitchenSaleId", "ks-1")
			r.SetPathValue("itemId", "item-1")
			w := httptest.NewRecorder()

			h.UpdateItemStatus(w, r)

			if w.Code != tt.expect {
				t.Fatalf("want %d, got %d: %s", tt.expect, w.Code, w.Body.String())
			}
		})
	}
}
