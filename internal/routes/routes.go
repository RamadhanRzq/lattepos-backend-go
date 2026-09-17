package routes

import (
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/domain/organization"
	"github.com/ramadhanrzq/backend-go/internal/domain/rbac"
	"github.com/ramadhanrzq/backend-go/internal/domain/user"
	"github.com/ramadhanrzq/backend-go/internal/handler"
	"github.com/ramadhanrzq/backend-go/internal/middleware"
)

type Config struct {
	AuthSvc     user.AuthService
	RBACSvc     rbac.RBACService
	OrgSvc      organization.OrgService
	AuthHandler *handler.AuthHandler
	UserHandler *handler.UserHandler
	RBACHandler *handler.RBACHandler
	OrgHandler  *handler.OrgHandler
}

func Register(cfg Config) http.Handler {
	mux := http.NewServeMux()

	registerV1(mux, cfg)

	return middleware.Logger(mux)
}

func registerV1(mux *http.ServeMux, cfg Config) {
	// Public routes
	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("POST /login", cfg.AuthHandler.Login)
	mux.HandleFunc("POST /api/v1/login", cfg.AuthHandler.Login)

	// Authenticated routes (no org context needed)
	mux.HandleFunc("GET /me", middleware.RequireAuth(cfg.AuthSvc, cfg.AuthHandler.Me))
	mux.HandleFunc("GET /api/v1/me", middleware.RequireAuth(cfg.AuthSvc, cfg.AuthHandler.Me))

	// Global organization management
	mux.HandleFunc("POST /api/v1/organizations", middleware.RequireAuth(cfg.AuthSvc, cfg.OrgHandler.CreateOrg))
	mux.HandleFunc("GET /api/v1/organizations", middleware.RequireAuth(cfg.AuthSvc, cfg.OrgHandler.ListOrgs))
	mux.HandleFunc("GET /api/v1/org/{slug}", middleware.RequireAuth(cfg.AuthSvc, cfg.OrgHandler.GetOrgBySlug))
	mux.HandleFunc("POST /api/v1/org/{slug}/auth/select", middleware.RequireAuth(cfg.AuthSvc, cfg.OrgHandler.SelectOrg))

	// Org-scoped member management
	mux.HandleFunc("GET /api/v1/org/{slug}/members", middleware.RequireOrgMember(cfg.AuthSvc, cfg.OrgSvc, cfg.OrgHandler.ListOrgMembers))
	mux.HandleFunc("POST /api/v1/org/{slug}/members", middleware.RequireOrgMember(cfg.AuthSvc, cfg.OrgSvc, cfg.OrgHandler.AddOrgMember))
	mux.HandleFunc("DELETE /api/v1/org/{slug}/members/{uid}", middleware.RequireOrgMember(cfg.AuthSvc, cfg.OrgSvc, cfg.OrgHandler.RemoveOrgMember))

	// Org-scoped user routes
	mux.HandleFunc("GET /api/v1/org/{slug}/users", middleware.RequireOrgMember(cfg.AuthSvc, cfg.OrgSvc, middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "users:read", cfg.UserHandler.List)))
	mux.HandleFunc("POST /api/v1/org/{slug}/users", middleware.RequireOrgMember(cfg.AuthSvc, cfg.OrgSvc, middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "users:create", cfg.UserHandler.Create)))
	mux.HandleFunc("GET /api/v1/org/{slug}/users/{id}", middleware.RequireOrgMember(cfg.AuthSvc, cfg.OrgSvc, middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "users:read", cfg.UserHandler.UserByID)))

	// Legacy user routes (unscoped fallback)
	mux.HandleFunc("GET /users", middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "users:read", cfg.UserHandler.List))
	mux.HandleFunc("POST /users", middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "users:create", cfg.UserHandler.Create))
	mux.HandleFunc("GET /users/{id}", middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "users:read", cfg.UserHandler.UserByID))

	// Org-scoped RBAC management routes
	mux.HandleFunc("GET /api/v1/org/{slug}/roles", middleware.RequireOrgMember(cfg.AuthSvc, cfg.OrgSvc, middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "roles:read", cfg.RBACHandler.ListRoles)))
	mux.HandleFunc("POST /api/v1/org/{slug}/roles", middleware.RequireOrgMember(cfg.AuthSvc, cfg.OrgSvc, middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "roles:create", cfg.RBACHandler.CreateRole)))
	mux.HandleFunc("GET /api/v1/org/{slug}/roles/{id}", middleware.RequireOrgMember(cfg.AuthSvc, cfg.OrgSvc, middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "roles:read", cfg.RBACHandler.GetRoleByID)))
	mux.HandleFunc("POST /api/v1/org/{slug}/roles/{id}/permissions", middleware.RequireOrgMember(cfg.AuthSvc, cfg.OrgSvc, middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "roles:update", cfg.RBACHandler.AssignPermission)))

	// Global RBAC management routes
	mux.HandleFunc("GET /permissions", middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "permissions:read", cfg.RBACHandler.ListPermissions))
	mux.HandleFunc("POST /permissions", middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "permissions:create", cfg.RBACHandler.CreatePermission))
	mux.HandleFunc("GET /roles", middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "roles:read", cfg.RBACHandler.ListRoles))
	mux.HandleFunc("POST /roles", middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "roles:create", cfg.RBACHandler.CreateRole))
	mux.HandleFunc("GET /roles/{id}", middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "roles:read", cfg.RBACHandler.GetRoleByID))
	mux.HandleFunc("POST /roles/{id}/permissions", middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "roles:update", cfg.RBACHandler.AssignPermission))
	mux.HandleFunc("POST /users/{id}/roles", middleware.RequirePermission(cfg.AuthSvc, cfg.RBACSvc, "users:update", cfg.RBACHandler.AssignRoleToUser))
}
