package auth

import (
	"net/http"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
)

// RegisterRoutes mendaftarkan endpoint publik (register, login, refresh, logout)
// dan endpoint terautentikasi yang dimiliki module auth.
// Login dan refresh mendapat rate limiting: 10 req/menit per IP.
func RegisterRoutes(mux *http.ServeMux, h *Handler, verifier middleware.TokenVerifier) {
	rl := middleware.NewRateLimiter(10, time.Minute)

	mux.HandleFunc("POST /api/v1/register", rl.Wrap(h.Register))
	mux.HandleFunc("POST /api/v1/login", rl.Wrap(h.Login))
	mux.HandleFunc("POST /api/v1/auth/refresh", rl.Wrap(h.Refresh))
	mux.HandleFunc("POST /api/v1/auth/logout", h.Logout)

	mux.HandleFunc("POST /api/v1/auth/logout-all", middleware.RequireAuth(verifier, h.LogoutAll))
	mux.HandleFunc("GET /api/v1/me", middleware.RequireAuth(verifier, h.Me))
}
