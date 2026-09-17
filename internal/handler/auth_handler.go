package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/domain/user"
	"github.com/ramadhanrzq/backend-go/internal/dto"
	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// AuthHandler menangani endpoint autentikasi: /login dan /me.
type AuthHandler struct {
	Auth user.AuthService
}

// Login memvalidasi kredensial dan menerbitkan JWT.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Username == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "Username dan password wajib diisi")
		return
	}

	u, token, err := h.Auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, "Username atau password salah")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal membuat token")
		return
	}

	response.JSON(w, http.StatusOK, dto.LoginResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresAt: time.Now().Add(h.Auth.TokenExpiration()),
		User:      u,
	})
}

// Me mengembalikan identitas user dari token yang sedang dipakai.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	response.JSON(w, http.StatusOK, dto.MeResponse{
		UserID:    claims.UserID,
		Username:  claims.Username,
		Email:     claims.Email,
		Name:      claims.Name,
		ExpiresAt: claims.ExpiresAt.Time,
	})
}
