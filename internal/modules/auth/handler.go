package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint autentikasi.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Login memvalidasi kredensial dan menerbitkan token pair.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Username == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "Username dan password wajib diisi")
		return
	}

	u, pair, orgs, err := h.svc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, users.ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, "Username atau password salah")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal memproses login")
		return
	}

	response.JSON(w, http.StatusOK, LoginResponse{
		AccessToken:   pair.AccessToken,
		TokenType:     "Bearer",
		ExpiresIn:     pair.ExpiresIn,
		RefreshToken:  pair.RefreshToken,
		User:          u,
		Organizations: orgs,
	})
}

// Register mendaftarkan user baru dan langsung menerbitkan token pair.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	u, pair, orgs, err := h.svc.Register(r.Context(), req.Username, req.Name, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, users.ErrUsernameTaken):
			response.Error(w, http.StatusConflict, err.Error())
		case errors.Is(err, users.ErrEmailTaken):
			response.Error(w, http.StatusConflict, err.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat akun")
		}
		return
	}

	response.JSON(w, http.StatusCreated, LoginResponse{
		AccessToken:   pair.AccessToken,
		TokenType:     "Bearer",
		ExpiresIn:     pair.ExpiresIn,
		RefreshToken:  pair.RefreshToken,
		User:          u,
		Organizations: orgs,
	})
}

// Refresh melakukan token rotation dan menerbitkan token pair baru.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "Refresh token wajib diisi")
		return
	}

	pair, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, ErrTokenReuse):
			response.Error(w, http.StatusUnauthorized, "Token telah digunakan, semua sesi direvoke")
		case errors.Is(err, ErrTokenRevoked), errors.Is(err, ErrTokenExpired), errors.Is(err, ErrTokenInvalid):
			response.Error(w, http.StatusUnauthorized, "Refresh token tidak valid")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal memperbarui token")
		}
		return
	}

	response.JSON(w, http.StatusOK, RefreshResponse{
		AccessToken:  pair.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    pair.ExpiresIn,
		RefreshToken: pair.RefreshToken,
	})
}

// Logout merevoke refresh token yang diberikan.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "Refresh token wajib diisi")
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal melakukan logout")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Berhasil logout"})
}

// LogoutAll merevoke seluruh refresh token milik user.
func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if err := h.svc.LogoutAll(r.Context(), claims.UserID); err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal melakukan logout semua perangkat")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Berhasil logout dari semua perangkat"})
}

// Me mengembalikan identitas user dari token yang sedang dipakai.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	u, err := h.svc.users.FindByID(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "User tidak ditemukan")
		return
	}

	response.JSON(w, http.StatusOK, MeResponse{
		UserID:   u.ID,
		Username: u.Username,
		Email:    u.Email,
		Name:     u.Name,
	})
}
