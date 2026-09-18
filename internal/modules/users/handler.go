package users

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint user.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List menangani GET /api/v1/users.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar user")
		return
	}

	response.JSON(w, http.StatusOK, list)
}

// Create menangani POST /api/v1/users.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	u, err := h.svc.Create(r.Context(), req.Username, req.Name, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Username, name, email, dan password wajib diisi")
		case errors.Is(err, ErrUsernameTaken):
			response.Error(w, http.StatusConflict, "Username sudah dipakai")
		case errors.Is(err, ErrEmailTaken):
			response.Error(w, http.StatusConflict, "Email sudah dipakai")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat user")
		}
		return
	}

	response.JSON(w, http.StatusCreated, u)
}

// ByID menangani GET /users/{id}.
func (h *Handler) ByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	u, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "User not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil user")
		return
	}

	response.JSON(w, http.StatusOK, u)
}

// ListOrg menangani GET /api/v1/org/{slug}/users dan hanya menampilkan anggota
// organisasi tersebut.
func (h *Handler) ListOrg(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	list, err := h.svc.ListByOrg(r.Context(), orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar user")
		return
	}

	response.JSON(w, http.StatusOK, list)
}

// OrgByID menangani GET /api/v1/org/{slug}/users/{id} dan menolak user yang
// bukan anggota organisasi tersebut.
func (h *Handler) OrgByID(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	u, err := h.svc.GetByIDInOrg(r.Context(), orgID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "User not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil user")
		return
	}

	response.JSON(w, http.StatusOK, u)
}
