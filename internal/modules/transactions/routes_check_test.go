package transactions_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/sales"
	"github.com/ramadhanrzq/backend-go/internal/modules/transactions"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

// Route alias lama tetap resolve; /daily resolve ke pola daily.
// Nil handler tak dipakai: hanya cek mux.Handler (routing), bukan invoke.
func TestAliasRoutesRegistered(t *testing.T) {
	mux := http.NewServeMux()
	transactions.RegisterRoutes(mux, (*sales.Handler)(nil), (*transactions.Handler)(nil), nil, nil)

	cases := []struct{ method, url, want string }{
		{"GET", "/api/v1/org/kopi/stores/s1/transactions/daily", "GET /api/v1/org/{slug}/stores/{storeId}/transactions/daily"},
		{"POST", "/api/v1/org/kopi/stores/s1/transactions", "POST /api/v1/org/{slug}/stores/{storeId}/transactions"},
		{"GET", "/api/v1/org/kopi/stores/s1/transactions", "GET /api/v1/org/{slug}/stores/{storeId}/transactions"},
		{"GET", "/api/v1/org/kopi/stores/s1/transactions/t1", "GET /api/v1/org/{slug}/stores/{storeId}/transactions/{id}"},
		{"PATCH", "/api/v1/org/kopi/stores/s1/transactions/t1/cancel", "PATCH /api/v1/org/{slug}/stores/{storeId}/transactions/{id}/cancel"},
	}

	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.url, nil)
		_, pattern := mux.Handler(req)
		if pattern != c.want {
			t.Errorf("%s %s: got pattern %q, want %q", c.method, c.url, pattern, c.want)
		}
	}
}

// Date rusak di handler harus 400, bukan 500.
func TestDailyBadDateReturns400(t *testing.T) {
	svc := transactions.NewService(&stubSales{}, stubStores{})
	h := transactions.NewHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/org/kopi/stores/s1/transactions/daily?date=bogus", nil)
	ctx := middleware.NewContextWithOrgID(req.Context(), "org-a")
	ctx = middleware.NewContextWithClaims(ctx, &appjwt.Claims{UserID: "u1"})
	req = req.WithContext(ctx)
	req.SetPathValue("storeId", "s1")

	rec := httptest.NewRecorder()
	h.Daily(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// Date valid tanpa data harus 200 + envelope kosong valid.
func TestDailyEmptyReturns200(t *testing.T) {
	svc := transactions.NewService(&stubSales{}, stubStores{})
	h := transactions.NewHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/org/kopi/stores/s1/transactions/daily?date=2026-09-22", nil)
	ctx := middleware.NewContextWithOrgID(req.Context(), "org-a")
	ctx = middleware.NewContextWithClaims(ctx, &appjwt.Claims{UserID: "u1"})
	req = req.WithContext(ctx)
	req.SetPathValue("storeId", "s1")

	rec := httptest.NewRecorder()
	h.Daily(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("bukan JSON valid: %v", err)
	}
	if body["date"] != "2026-09-22" {
		t.Fatalf("date salah: %v", body["date"])
	}
}
