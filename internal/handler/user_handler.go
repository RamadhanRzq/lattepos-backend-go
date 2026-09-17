package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/domain/user"
	"github.com/ramadhanrzq/backend-go/internal/dto"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// UserHandler menangani endpoint /users.
type UserHandler struct {
	Service user.UserService
}

// List menangani GET /users.
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.Service.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar user")
		return
	}

	response.JSON(w, http.StatusOK, users)
}

// Create menangani POST /users.
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	u, err := h.Service.Create(r.Context(), req.Username, req.Name, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Username, name, email, dan password wajib diisi")
		case errors.Is(err, user.ErrUsernameTaken):
			response.Error(w, http.StatusConflict, "Username sudah dipakai")
		case errors.Is(err, user.ErrEmailTaken):
			response.Error(w, http.StatusConflict, "Email sudah dipakai")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat user")
		}
		return
	}

	response.JSON(w, http.StatusCreated, u)
}

// UserByID menangani GET /users/{id}.
func (h *UserHandler) UserByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	u, err := h.Service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "User not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil user")
		return
	}

	response.JSON(w, http.StatusOK, u)
}
