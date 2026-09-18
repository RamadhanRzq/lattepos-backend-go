package auth

import (
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
)

// RegisterRoutes mendaftarkan endpoint publik (register, login) dan endpoint
// terautentikasi yang dimiliki module auth.
func RegisterRoutes(mux *http.ServeMux, h *Handler, verifier middleware.TokenVerifier) {
	mux.HandleFunc("POST /api/v1/register", h.Register)
	mux.HandleFunc("POST /api/v1/login", h.Login)

	mux.HandleFunc("GET /api/v1/me", middleware.RequireAuth(verifier, h.Me))
}
