package router

import (
	"fmt"
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/auth"
	"github.com/ramadhanrzq/backend-go/internal/modules/categories"
	"github.com/ramadhanrzq/backend-go/internal/modules/kitchen"
	"github.com/ramadhanrzq/backend-go/internal/modules/organizations"
	"github.com/ramadhanrzq/backend-go/internal/modules/prices"
	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/rbac"
	"github.com/ramadhanrzq/backend-go/internal/modules/sales"
	"github.com/ramadhanrzq/backend-go/internal/modules/stock"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	"github.com/ramadhanrzq/backend-go/internal/modules/transactions"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
	"github.com/ramadhanrzq/backend-go/internal/modules/variants"
)

// Deps adalah dependency yang dibutuhkan untuk merangkai seluruh route.
// Guard diambil sebagai interface supaya module tidak saling mengimpor.
type Deps struct {
	Verifier    middleware.TokenVerifier
	Permissions middleware.PermissionChecker
	Orgs        middleware.OrgMembership
	CORSConfig  middleware.CORSConfig

	Auth          *auth.Handler
	Users         *users.Handler
	RBAC          *rbac.Handler
	Organizations *organizations.Handler
	Stores        *stores.Handler
	Products      *products.Handler
	Sales         *sales.Handler
	Transactions  *transactions.Handler
	Categories    *categories.Handler
	Variants      *variants.Handler
	Prices        *prices.Handler
	Stock         *stock.Handler
	Kitchen       *kitchen.Handler
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
	sales.RegisterRoutes(mux, deps.Sales, deps.Verifier, deps.Orgs)
	transactions.RegisterRoutes(mux, deps.Sales, deps.Transactions, deps.Verifier, deps.Orgs)
	categories.RegisterRoutes(mux, deps.Categories, deps.Verifier, deps.Orgs)
	variants.RegisterRoutes(mux, deps.Variants, deps.Verifier, deps.Orgs)
	prices.RegisterRoutes(mux, deps.Prices, deps.Verifier, deps.Orgs)
	stock.RegisterRoutes(mux, deps.Stock, deps.Verifier, deps.Orgs)
	kitchen.RegisterRoutes(mux, deps.Kitchen, deps.Verifier, deps.Orgs)
	return middleware.CORS(deps.CORSConfig, middleware.Logger(mux))
}

// health adalah liveness endpoint level aplikasi.
func health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}
