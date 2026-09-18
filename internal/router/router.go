package router

import (
	"fmt"
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/auth"
	"github.com/ramadhanrzq/backend-go/internal/modules/organizations"
	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/rbac"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
)

// Deps adalah dependency yang dibutuhkan untuk merangkai seluruh route.
// Guard diambil sebagai interface supaya module tidak saling mengimpor.
type Deps struct {
	Verifier    middleware.TokenVerifier
	Permissions middleware.PermissionChecker
	Orgs        middleware.OrgMembership

	Auth          *auth.Handler
	Users         *users.Handler
	RBAC          *rbac.Handler
	Organizations *organizations.Handler
	Stores        *stores.Handler
	Products      *products.Handler
}

// New merangkai route semua module menjadi satu http.Handler.
func New(deps Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", health)
	registerDocs(mux)

	auth.RegisterRoutes(mux, deps.Auth, deps.Verifier)
	users.RegisterRoutes(mux, deps.Users, deps.Verifier, deps.Permissions, deps.Orgs)
	rbac.RegisterRoutes(mux, deps.RBAC, deps.Verifier, deps.Permissions, deps.Orgs)
	organizations.RegisterRoutes(mux, deps.Organizations, deps.Verifier, deps.Orgs)
	stores.RegisterRoutes(mux, deps.Stores, deps.Verifier, deps.Orgs)
	products.RegisterRoutes(mux, deps.Products, deps.Verifier, deps.Orgs)
	return middleware.Logger(mux)
}

// health adalah liveness endpoint level aplikasi.
func health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}
