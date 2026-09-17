package organizations

import (
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
)

// RegisterRoutes mendaftarkan endpoint organization dan keanggotaannya.
func RegisterRoutes(
	mux *http.ServeMux,
	h *Handler,
	verifier middleware.TokenVerifier,
	orgs middleware.OrgMembership,
) {
	mux.HandleFunc("POST /api/v1/organizations", middleware.RequireAuth(verifier, h.CreateOrg))
	mux.HandleFunc("GET /api/v1/organizations", middleware.RequireAuth(verifier, h.ListOrgs))
	mux.HandleFunc("GET /api/v1/org/{slug}", middleware.RequireAuth(verifier, h.GetOrgBySlug))
	mux.HandleFunc("POST /api/v1/org/{slug}/auth/select", middleware.RequireAuth(verifier, h.SelectOrg))

	mux.HandleFunc("GET /api/v1/org/{slug}/members", middleware.RequireOrgMember(verifier, orgs, h.ListOrgMembers))
	mux.HandleFunc("POST /api/v1/org/{slug}/members", middleware.RequireOrgMember(verifier, orgs, h.AddOrgMember))
	mux.HandleFunc("DELETE /api/v1/org/{slug}/members/{uid}", middleware.RequireOrgMember(verifier, orgs, h.RemoveOrgMember))
}
