package users

import (
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
)

// RegisterRoutes mendaftarkan endpoint user, baik yang global maupun yang
// ter-scope organisasi. Guard HTTP diambil dari module middleware.
func RegisterRoutes(
	mux *http.ServeMux,
	h *Handler,
	verifier middleware.TokenVerifier,
	perms middleware.PermissionChecker,
	orgs middleware.OrgMembership,
) {
	mux.HandleFunc("GET /users", middleware.RequirePermission(verifier, perms, "users:read", h.List))
	mux.HandleFunc("POST /users", middleware.RequirePermission(verifier, perms, "users:create", h.Create))
	mux.HandleFunc("GET /users/{id}", middleware.RequirePermission(verifier, perms, "users:read", h.ByID))

	mux.HandleFunc("GET /api/v1/org/{slug}/users",
		middleware.RequireOrgMember(verifier, orgs, middleware.RequirePermission(verifier, perms, "users:read", h.ListOrg)))
	mux.HandleFunc("POST /api/v1/org/{slug}/users",
		middleware.RequireOrgMember(verifier, orgs, middleware.RequirePermission(verifier, perms, "users:create", h.Create)))
	mux.HandleFunc("GET /api/v1/org/{slug}/users/{id}",
		middleware.RequireOrgMember(verifier, orgs, middleware.RequirePermission(verifier, perms, "users:read", h.OrgByID)))
}
