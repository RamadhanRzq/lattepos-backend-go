package kitchen

import (
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
)

// RegisterRoutes mendaftarkan endpoint kitchen yang ter-scope store+organisasi.
// Semua rute lewat RequireOrgMember: tenant boundary dari slug path,
// store boundary dari storeId path.
func RegisterRoutes(
	mux *http.ServeMux,
	h *Handler,
	verifier middleware.TokenVerifier,
	orgs middleware.OrgMembership,
) {
	mux.HandleFunc("GET /api/v1/org/{slug}/stores/{storeId}/kitchen/queue",
		middleware.RequireOrgMember(verifier, orgs, h.Queue))
	mux.HandleFunc("GET /api/v1/org/{slug}/stores/{storeId}/kitchen/{kitchenSaleId}",
		middleware.RequireOrgMember(verifier, orgs, h.ByID))
	mux.HandleFunc("PATCH /api/v1/org/{slug}/stores/{storeId}/kitchen/{kitchenSaleId}/status",
		middleware.RequireOrgMember(verifier, orgs, h.UpdateStatus))
	mux.HandleFunc("PATCH /api/v1/org/{slug}/stores/{storeId}/kitchen/{kitchenSaleId}/items/{itemId}/status",
		middleware.RequireOrgMember(verifier, orgs, h.UpdateItemStatus))
}
