package stock

import (
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
)

// RegisterRoutes mendaftarkan endpoint stock movement yang ter-scope store+organisasi.
// Semua rute lewat RequireOrgMember: tenant boundary dari slug path,
// store boundary dari storeId path; client tidak boleh mengirim organization_id/store_id.
func RegisterRoutes(
	mux *http.ServeMux,
	h *Handler,
	verifier middleware.TokenVerifier,
	orgs middleware.OrgMembership,
) {
	mux.HandleFunc("POST /api/v1/org/{slug}/stores/{storeId}/stock-movements",
		middleware.RequireOrgMember(verifier, orgs, h.Record))
	mux.HandleFunc("GET /api/v1/org/{slug}/stores/{storeId}/stock-movements",
		middleware.RequireOrgMember(verifier, orgs, h.List))
	mux.HandleFunc("GET /api/v1/org/{slug}/stores/{storeId}/products/{productId}/stock",
		middleware.RequireOrgMember(verifier, orgs, h.StockByProduct))
}
