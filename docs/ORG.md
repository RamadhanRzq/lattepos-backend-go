You are an expert Go backend developer. I have an existing modular monolith Go backend with this structure:

## Existing Codebase

### Module path
github.com/ramadhanrzq/backend-go

### Project structure
internal/
  config/
  domain/
    user/       → User entity, AuthService, UserService interfaces
    rbac/       → Permission, Role, RBACService interfaces
  handler/
    auth_handler.go
    user_handler.go
    rbac_handler.go
  middleware/
    auth.go         → RequireAuth(authSvc, next)
    permission.go   → RequirePermission(authSvc, rbacSvc, permission, next)
    logger.go
  repository/
    postgres/
      user_repository.go
      permission_repository.go
      role_repository.go
  routes/
    routes.go       → Register(cfg Config) http.Handler, registerV1(mux, cfg)
  service/
    auth_service.go
    user_service.go
    rbac_service.go
pkg/
  jwt/

### Stack
- Go standard library net/http (no router framework)
- PostgreSQL via database/sql + lib/pq
- JWT authentication (HS256)
- RBAC: permissions, roles, role_permissions, user_roles tables

### Existing DB tables
- users: id UUID PK, name, email, password_hash, created_at
- permissions: id UUID PK, name, description, created_at
- roles: id UUID PK, name, description, created_at
- role_permissions: role_id FK, permission_id FK
- user_roles: user_id FK, role_id FK

### Existing auth flow
POST /login → AuthService.Login → validates credentials → returns JWT
JWT payload contains: user_id, email

---

## Task: Add Multi-Organization (Multi-Tenant) Feature

### Business rules
- Every user belongs to exactly one organization
- Organization has: id, name, slug (unique, URL-safe), created_at
- Users are scoped per organization — user from org A cannot access data from org B
- Roles are scoped per organization — org A can have its own "admin" role separate from org B's "admin"
- The first user to create an organization automatically becomes its owner (has all permissions)
- Organization slug is used in API path: /api/v1/org/{slug}/users, etc.
- JWT must carry organization context after user selects/switches org

### What I need

#### 1. DB Schema (migration files)
New tables:
- organizations: id UUID PK, name VARCHAR, slug VARCHAR UNIQUE, created_at TIMESTAMP
- organization_members: id UUID PK, org_id FK → organizations, user_id FK → users, joined_at TIMESTAMP, UNIQUE(org_id, user_id)

Modify existing:
- roles: add org_id UUID FK → organizations (roles are now org-scoped)
- user_roles: add org_id UUID FK → organizations (so same user can have different roles in different orgs)

Write both UP and DOWN migrations. Use UUID for all PKs.

#### 2. Domain Layer
Following existing domain pattern in internal/domain/:

File internal/domain/organization/organization.go:
- Struct: Organization { ID, Name, Slug, CreatedAt }
- Struct: Member { ID, OrgID, UserID, JoinedAt }
- Interface OrgRepository:
  - Create(ctx, org *Organization) error
  - FindByID(ctx, id UUID) (*Organization, error)
  - FindBySlug(ctx, slug string) (*Organization, error)
  - List(ctx) ([]Organization, error)
  - AddMember(ctx, orgID, userID UUID) error
  - RemoveMember(ctx, orgID, userID UUID) error
  - IsMember(ctx, orgID, userID UUID) (bool, error)
  - ListMembers(ctx, orgID UUID) ([]user.User, error)
- Interface OrgService:
  - Create(ctx, name, slug string, ownerID UUID) (*Organization, error)
  - GetBySlug(ctx, slug string) (*Organization, error)
  - List(ctx) ([]Organization, error)
  - AddMember(ctx, orgID, userID UUID) error
  - RemoveMember(ctx, orgID, userID UUID) error
  - ListMembers(ctx, orgID UUID) ([]user.User, error)
  - IsUserMember(ctx, orgID, userID UUID) (bool, error)

#### 3. Repository Layer
File internal/repository/postgres/organization_repository.go:
- Implement OrgRepository interface
- Use database/sql, no ORM
- Use prepared statements
- IsMember must be efficient (SELECT EXISTS query)

#### 4. Service Layer
File internal/service/organization_service.go:
- Implement OrgService interface
- When Create is called: insert org, then AddMember for ownerID, then assign owner role
- Slug validation: lowercase, alphanumeric + hyphen only, min 3 chars
- Return typed domain errors (not raw DB errors)
- IsMember check must be used inside every service method that touches org data

#### 5. Middleware
File internal/middleware/org.go:

RequireOrgMember middleware:
- Signature: RequireOrgMember(authSvc user.AuthService, orgSvc organization.OrgService, next http.HandlerFunc) http.HandlerFunc
- Extract org slug from URL path using r.PathValue("slug") (Go 1.22+)
- Validate JWT (reuse existing auth logic)
- Check IsMember(orgID, userID) — return 403 if not a member
- Store org in request context: ctx = context.WithValue(ctx, OrgKey, org)
- Helper: OrgFromContext(ctx) (*Organization, bool)

#### 6. HTTP Handlers
File internal/handler/organization_handler.go:
- OrgHandler struct { Service organization.OrgService }
- Handlers:
  - CreateOrg    → POST /api/v1/organizations
  - ListOrgs     → GET  /api/v1/organizations
  - GetOrgBySlug → GET  /api/v1/org/{slug}

File: extend internal/handler/user_handler.go or create org_user_handler.go:
- ListOrgMembers   → GET    /api/v1/org/{slug}/members
- AddOrgMember     → POST   /api/v1/org/{slug}/members        { "user_id": "uuid" }
- RemoveOrgMember  → DELETE /api/v1/org/{slug}/members/{uid}

#### 7. JWT changes
Extend JWT payload to carry org context after login:
- Current payload: { user_id, email }
- New payload after org selection: { user_id, email, org_id, org_slug }

Add new endpoint:
- POST /api/v1/org/{slug}/auth/select → user picks which org to "enter", returns new JWT with org context
- Existing POST /login stays the same (returns JWT without org context)
- Middleware RequireOrgMember should work with both: slug from URL takes precedence, falls back to org_id from JWT claims

#### 8. Routes
Update internal/routes/routes.go, add to registerV1():

// Org management (no org context needed yet)
mux.HandleFunc("POST /organizations",    RequireAuth → CreateOrg)
mux.HandleFunc("GET  /organizations",    RequireAuth → ListOrgs)
mux.HandleFunc("GET  /org/{slug}",       RequireAuth → GetOrgBySlug)
mux.HandleFunc("POST /org/{slug}/auth/select", RequireAuth → SelectOrg)

// Org-scoped routes (RequireOrgMember wraps everything under /org/{slug}/*)
mux.HandleFunc("GET    /org/{slug}/members",         RequireOrgMember → ListOrgMembers)
mux.HandleFunc("POST   /org/{slug}/members",         RequireOrgMember → AddOrgMember)
mux.HandleFunc("DELETE /org/{slug}/members/{uid}",   RequireOrgMember → RemoveOrgMember)

// Org-scoped user routes (existing user routes now org-aware)
mux.HandleFunc("GET    /org/{slug}/users",      RequireOrgMember + RequirePermission("users:read")   → UserHandler.List)
mux.HandleFunc("POST   /org/{slug}/users",      RequireOrgMember + RequirePermission("users:create") → UserHandler.Create)
mux.HandleFunc("GET    /org/{slug}/users/{id}", RequireOrgMember + RequirePermission("users:read")   → UserHandler.UserByID)

// Org-scoped RBAC routes
mux.HandleFunc("GET  /org/{slug}/roles",                   RequireOrgMember + RequirePermission("roles:read")   → RBACHandler.ListRoles)
mux.HandleFunc("POST /org/{slug}/roles",                   RequireOrgMember + RequirePermission("roles:create") → RBACHandler.CreateRole)
mux.HandleFunc("GET  /org/{slug}/roles/{id}",              RequireOrgMember + RequirePermission("roles:read")   → RBACHandler.GetRoleByID)
mux.HandleFunc("POST /org/{slug}/roles/{id}/permissions",  RequireOrgMember + RequirePermission("roles:update") → RBACHandler.AssignPermission)

#### 9. Context helpers
File internal/middleware/context.go (or extend existing):
- OrgKey type and constants
- OrgFromContext(ctx) (*organization.Organization, bool)
- UserFromContext(ctx) (*user.User, bool)  ← if not already exists

#### 10. Config struct update
Update internal/routes/routes.go Config struct:
type Config struct {
  AuthSvc     user.AuthService
  RBACSvc     rbac.RBACService
  OrgSvc      organization.OrgService   ← new
  AuthHandler *handler.AuthHandler
  UserHandler *handler.UserHandler
  RBACHandler *handler.RBACHandler
  OrgHandler  *handler.OrgHandler       ← new
}

---

## Code style requirements
- Match existing patterns exactly: same error wrapping, same JSON response shape via Response Helper
- Use r.PathValue("slug") and r.PathValue("id") — Go 1.22+ path params (no third-party router)
- Context keys as unexported types to avoid collision
- Domain errors as typed sentinel errors, not string errors
- No new external dependencies — only standard library + lib/pq + existing jwt package
- UUID: use github.com/google/uuid if already in go.mod
- All handlers follow pattern: decode request → validate → call service → encode response
- Middleware chains: RequireOrgMember always runs before RequirePermission

## Output format
Output each file completely — no placeholders, no "// rest of code here".
Order: migration → domain → repository → service → middleware → handler → routes.
For files that need modification (routes.go, jwt, user_handler), show the complete updated file.