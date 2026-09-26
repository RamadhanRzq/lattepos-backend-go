package recipes

import (
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
)

// RegisterRoutes mendaftarkan endpoint recipe yang ter-scope product+store+organisasi.
// Semua rute lewat RequireOrgMember: tenant boundary dari slug path,
// store/product boundary dari path; client tidak boleh mengirim organization_id/store_id.
func RegisterRoutes(
	mux *http.ServeMux,
	h *Handler,
	verifier middleware.TokenVerifier,
	orgs middleware.OrgMembership,
) {
	mux.HandleFunc("POST /api/v1/org/{slug}/stores/{storeId}/products/{productId}/recipes",
		middleware.RequireOrgMember(verifier, orgs, h.Create))
	mux.HandleFunc("GET /api/v1/org/{slug}/stores/{storeId}/products/{productId}/recipes",
		middleware.RequireOrgMember(verifier, orgs, h.List))
	mux.HandleFunc("GET /api/v1/org/{slug}/stores/{storeId}/products/{productId}/recipes/{recipeId}",
		middleware.RequireOrgMember(verifier, orgs, h.ByID))
	mux.HandleFunc("PUT /api/v1/org/{slug}/stores/{storeId}/products/{productId}/recipes/{recipeId}",
		middleware.RequireOrgMember(verifier, orgs, h.Update))
	mux.HandleFunc("DELETE /api/v1/org/{slug}/stores/{storeId}/products/{productId}/recipes/{recipeId}",
		middleware.RequireOrgMember(verifier, orgs, h.Delete))
	mux.HandleFunc("PATCH /api/v1/org/{slug}/stores/{storeId}/products/{productId}/recipes/{recipeId}/activate",
		middleware.RequireOrgMember(verifier, orgs, h.Activate))
	mux.HandleFunc("GET /api/v1/org/{slug}/stores/{storeId}/recipes",
		middleware.RequireOrgMember(verifier, orgs, h.ListByStore))
}
