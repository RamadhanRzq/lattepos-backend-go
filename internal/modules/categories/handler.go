package categories

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint category.
// orgID selalu dari RequireOrgMember context, storeID dari path;
// client tidak boleh mengirim organization_id/store_id.
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
	storeID := strings.TrimSpace(r.PathValue("storeId"))
	if storeID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	c, err := h.svc.Create(r.Context(), orgID, storeID, req.Name, req.Slug, req.Description, req.ParentID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrInvalidParent):
			response.Error(w, http.StatusBadRequest, "name wajib diisi, parent harus satu store")
		case errors.Is(err, ErrSlugExists):
			response.Error(w, http.StatusConflict, "Slug sudah dipakai di store ini")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat category")
		}
		return
	}

	response.JSON(w, http.StatusCreated, c)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	storeID := strings.TrimSpace(r.PathValue("storeId"))
	if storeID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID")
		return
	}

	list, err := h.svc.List(r.Context(), orgID, storeID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Invalid store ID")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengambil categories")
		}
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
	storeID := strings.TrimSpace(r.PathValue("storeId"))
	id := strings.TrimSpace(r.PathValue("id"))
	if storeID == "" || id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau category ID")
		return
	}

	c, err := h.svc.Get(r.Context(), orgID, storeID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "Category not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil category")
		return
	}

	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	storeID := strings.TrimSpace(r.PathValue("storeId"))
	id := strings.TrimSpace(r.PathValue("id"))
	if storeID == "" || id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau category ID")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	c, err := h.svc.Update(r.Context(), orgID, storeID, id, req.Name, req.Slug, req.Description, req.ParentID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrInvalidParent):
			response.Error(w, http.StatusBadRequest, "name wajib diisi, parent harus satu store")
		case errors.Is(err, ErrSlugExists):
			response.Error(w, http.StatusConflict, "Slug sudah dipakai di store ini")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Category not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengubah category")
		}
		return
	}

	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	storeID := strings.TrimSpace(r.PathValue("storeId"))
	id := strings.TrimSpace(r.PathValue("id"))
	if storeID == "" || id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau category ID")
		return
	}

	if err := h.svc.Delete(r.Context(), orgID, storeID, id); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Category not found")
		case errors.Is(err, ErrInUse):
			response.Error(w, http.StatusConflict, "Category masih dipakai product")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menghapus category")
		}
		return
	}

	response.JSON(w, http.StatusOK, MessageResponse{Message: "Category berhasil dihapus"})
}
