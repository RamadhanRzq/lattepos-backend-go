package stores

import (
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
)

// RegisterRoutes mendaftarkan endpoint store yang ter-scope organisasi.
// Semua rute lewat RequireOrgMember: tenant boundary dari slug path,
// bukan dari input client.
func RegisterRoutes(
	mux *http.ServeMux,
	h *Handler,
	verifier middleware.TokenVerifier,
	orgs middleware.OrgMembership,
) {
	mux.HandleFunc("POST /api/v1/org/{slug}/stores",
		middleware.RequireOrgMember(verifier, orgs, h.Create))
	mux.HandleFunc("GET /api/v1/org/{slug}/stores",
		middleware.RequireOrgMember(verifier, orgs, h.List))
	mux.HandleFunc("GET /api/v1/org/{slug}/stores/{id}",
		middleware.RequireOrgMember(verifier, orgs, h.ByID))
	mux.HandleFunc("PUT /api/v1/org/{slug}/stores/{id}",
		middleware.RequireOrgMember(verifier, orgs, h.Update))
	mux.HandleFunc("PATCH /api/v1/org/{slug}/stores/{id}/status",
		middleware.RequireOrgMember(verifier, orgs, h.SetStatus))

	mux.HandleFunc("POST /api/v1/org/{slug}/stores/{id}/users",
		middleware.RequireOrgMember(verifier, orgs, h.AssignUser))
	mux.HandleFunc("DELETE /api/v1/org/{slug}/stores/{id}/users/{userId}",
		middleware.RequireOrgMember(verifier, orgs, h.RemoveUser))
	mux.HandleFunc("GET /api/v1/org/{slug}/stores/{id}/users",
		middleware.RequireOrgMember(verifier, orgs, h.ListStoreUsers))

	mux.HandleFunc("GET /api/v1/org/{slug}/users/{id}/stores",
		middleware.RequireOrgMember(verifier, orgs, h.ListUserStores))
}
