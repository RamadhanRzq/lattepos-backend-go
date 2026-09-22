package transactions

import (
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/sales"
)

// RegisterRoutes mendaftarkan endpoint transactions yang ter-scope store+organisasi.
// /daily = laporan harian agregat dari sales.Service.List (tanpa tabel baru).
// CRUD lain tetap alias langsung ke sales.Handler supaya payload identik.
// skipped: service/query agregasi SQL sendiri, add when volume butuh COUNT/SUM di DB.
func RegisterRoutes(
	mux *http.ServeMux,
	h *sales.Handler,
	report *Handler,
	verifier middleware.TokenVerifier,
	orgs middleware.OrgMembership,
) {
	mux.HandleFunc("GET /api/v1/org/{slug}/stores/{storeId}/transactions/daily",
		middleware.RequireOrgMember(verifier, orgs, report.Daily))
	mux.HandleFunc("POST /api/v1/org/{slug}/stores/{storeId}/transactions",
		middleware.RequireOrgMember(verifier, orgs, h.Create))
	mux.HandleFunc("GET /api/v1/org/{slug}/stores/{storeId}/transactions",
		middleware.RequireOrgMember(verifier, orgs, h.List))
	mux.HandleFunc("GET /api/v1/org/{slug}/stores/{storeId}/transactions/{id}",
		middleware.RequireOrgMember(verifier, orgs, h.ByID))
	mux.HandleFunc("PATCH /api/v1/org/{slug}/stores/{storeId}/transactions/{id}/cancel",
		middleware.RequireOrgMember(verifier, orgs, h.Cancel))
}
