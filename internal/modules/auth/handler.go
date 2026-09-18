package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint autentikasi: /api/v1/register, /api/v1/login dan /api/v1/me.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Login memvalidasi kredensial dan menerbitkan JWT.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Username == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "Username dan password wajib diisi")
		return
	}

	u, token, orgs, err := h.svc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, users.ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, "Username atau password salah")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal membuat token")
		return
	}

	response.JSON(w, http.StatusOK, LoginResponse{
		Token:         token,
		TokenType:     "Bearer",
		ExpiresAt:     time.Now().Add(h.svc.TokenExpiration()),
		User:          u,
		Organizations: orgs,
	})
}

// Register mendaftarkan user baru dan langsung menerbitkan JWT.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	u, token, orgs, err := h.svc.Register(r.Context(), req.Username, req.Name, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Username, name, email, dan password wajib diisi")
		case errors.Is(err, users.ErrUsernameTaken):
			response.Error(w, http.StatusConflict, "Username sudah dipakai")
		case errors.Is(err, users.ErrEmailTaken):
			response.Error(w, http.StatusConflict, "Email sudah dipakai")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal registrasi")
		}
		return
	}

	response.JSON(w, http.StatusCreated, LoginResponse{
		Token:         token,
		TokenType:     "Bearer",
		ExpiresAt:     time.Now().Add(h.svc.TokenExpiration()),
		User:          u,
		Organizations: orgs,
	})
}

// Me mengembalikan identitas user dari token yang sedang dipakai.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	response.JSON(w, http.StatusOK, MeResponse{
		UserID:    claims.UserID,
		Username:  claims.Username,
		Email:     claims.Email,
		Name:      claims.Name,
		ExpiresAt: claims.ExpiresAt.Time,
	})
}
