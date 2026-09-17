package stores

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint store dan akses user-store.
// orgID selalu dari RequireOrgMember context; client tidak boleh mengirim organization_id.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	store, err := h.svc.Create(r.Context(), orgID, req.Name, req.Code, req.Address, req.Phone)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "name dan code wajib diisi")
		case errors.Is(err, ErrInvalidCode):
			response.Error(w, http.StatusBadRequest, "code hanya boleh huruf, angka, dash, underscore")
		case errors.Is(err, ErrCodeTaken):
			response.Error(w, http.StatusConflict, "Code sudah dipakai di organisasi ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat store")
		}
		return
	}

	response.JSON(w, http.StatusCreated, store)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	list, err := h.svc.List(r.Context(), orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar store")
		return
	}

	response.JSON(w, http.StatusOK, list)
}

func (h *Handler) ByID(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID")
		return
	}

	store, err := h.svc.Get(r.Context(), orgID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "Store not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil store")
		return
	}

	response.JSON(w, http.StatusOK, store)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	store, err := h.svc.Update(r.Context(), orgID, id, req.Name, req.Code, req.Address, req.Phone)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "name dan code wajib diisi")
		case errors.Is(err, ErrInvalidCode):
			response.Error(w, http.StatusBadRequest, "code hanya boleh huruf, angka, dash, underscore")
		case errors.Is(err, ErrCodeTaken):
			response.Error(w, http.StatusConflict, "Code sudah dipakai di organisasi ini")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengubah store")
		}
		return
	}

	response.JSON(w, http.StatusOK, store)
}

func (h *Handler) SetStatus(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID")
		return
	}

	var req StatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if req.IsActive == nil {
		response.Error(w, http.StatusBadRequest, "is_active wajib diisi")
		return
	}

	store, err := h.svc.SetActive(r.Context(), orgID, id, *req.IsActive)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "Store not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal mengubah status store")
		return
	}

	response.JSON(w, http.StatusOK, store)
}

func (h *Handler) AssignUser(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	storeID := strings.TrimSpace(r.PathValue("id"))
	if storeID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID")
		return
	}

	var req AssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	req.UserID = strings.TrimSpace(req.UserID)
	if req.UserID == "" {
		response.Error(w, http.StatusBadRequest, "user_id wajib diisi")
		return
	}

	err := h.svc.AssignUser(r.Context(), orgID, storeID, req.UserID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		case errors.Is(err, ErrUserNotFound):
			response.Error(w, http.StatusNotFound, "User not found")
		case errors.Is(err, ErrCrossOrg):
			response.Error(w, http.StatusForbidden, "User dan store harus dalam organisasi yang sama")
		case errors.Is(err, ErrAlreadyAssigned):
			response.Error(w, http.StatusConflict, "User sudah memiliki akses ke store ini")
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "store_id dan user_id tidak valid")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menambahkan user ke store")
		}
		return
	}

	response.JSON(w, http.StatusOK, MessageResponse{Message: "User berhasil ditambahkan ke store"})
}

func (h *Handler) RemoveUser(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	storeID := strings.TrimSpace(r.PathValue("id"))
	userID := strings.TrimSpace(r.PathValue("userId"))
	if storeID == "" || userID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau user ID")
		return
	}

	err := h.svc.RemoveUser(r.Context(), orgID, storeID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		case errors.Is(err, ErrNotAssigned):
			response.Error(w, http.StatusNotFound, "User tidak memiliki akses ke store ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menghapus user dari store")
		}
		return
	}

	response.JSON(w, http.StatusOK, MessageResponse{Message: "User berhasil dihapus dari store"})
}

func (h *Handler) ListStoreUsers(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	storeID := strings.TrimSpace(r.PathValue("id"))
	if storeID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID")
		return
	}

	members, err := h.svc.ListStoreUsers(r.Context(), orgID, storeID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "Store not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil user store")
		return
	}

	response.JSON(w, http.StatusOK, members)
}

// ListUserStores menangani GET /api/v1/org/{slug}/users/{id}/stores:
// user lintas org ditolak NotFound oleh service (membership check), bukan 500.
func (h *Handler) ListUserStores(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	userID := strings.TrimSpace(r.PathValue("id"))
	if userID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	list, err := h.svc.ListUserStores(r.Context(), orgID, userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil store user")
		return
	}

	response.JSON(w, http.StatusOK, list)
}
