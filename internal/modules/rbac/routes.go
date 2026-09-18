package rbac

import (
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
)

// RegisterRoutes mendaftarkan endpoint RBAC: rute global memakai role global,
// rute /api/v1/org/{slug}/... memakai role milik organisasi tersebut.
func RegisterRoutes(
	mux *http.ServeMux,
	h *Handler,
	verifier middleware.TokenVerifier,
	perms middleware.PermissionChecker,
	orgs middleware.OrgMembership,
) {
	mux.HandleFunc("GET /api/v1/permissions", middleware.RequirePermission(verifier, perms, "permissions:read", h.ListPermissions))
	mux.HandleFunc("POST /api/v1/permissions", middleware.RequirePermission(verifier, perms, "permissions:create", h.CreatePermission))
	mux.HandleFunc("GET /api/v1/roles", middleware.RequirePermission(verifier, perms, "roles:read", h.ListRoles))
	mux.HandleFunc("POST /api/v1/roles", middleware.RequirePermission(verifier, perms, "roles:create", h.CreateRole))
	mux.HandleFunc("GET /api/v1/roles/{id}", middleware.RequirePermission(verifier, perms, "roles:read", h.GetRoleByID))
	mux.HandleFunc("POST /api/v1/roles/{id}/permissions", middleware.RequirePermission(verifier, perms, "roles:update", h.AssignPermission))
	mux.HandleFunc("POST /api/v1/users/{id}/roles", middleware.RequirePermission(verifier, perms, "users:update", h.AssignRoleToUser))

	mux.HandleFunc("GET /api/v1/org/{slug}/roles",
		middleware.RequireOrgMember(verifier, orgs, middleware.RequirePermission(verifier, perms, "roles:read", h.ListOrgRoles)))
	mux.HandleFunc("POST /api/v1/org/{slug}/roles",
		middleware.RequireOrgMember(verifier, orgs, middleware.RequirePermission(verifier, perms, "roles:create", h.CreateOrgRole)))
	mux.HandleFunc("GET /api/v1/org/{slug}/roles/{id}",
		middleware.RequireOrgMember(verifier, orgs, middleware.RequirePermission(verifier, perms, "roles:read", h.GetOrgRoleByID)))
	mux.HandleFunc("POST /api/v1/org/{slug}/roles/{id}/permissions",
		middleware.RequireOrgMember(verifier, orgs, middleware.RequirePermission(verifier, perms, "roles:update", h.AssignOrgPermission)))
	mux.HandleFunc("POST /api/v1/org/{slug}/users/{id}/roles",
		middleware.RequireOrgMember(verifier, orgs, middleware.RequirePermission(verifier, perms, "users:update", h.AssignOrgRoleToUser)))
}
