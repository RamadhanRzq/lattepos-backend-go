package tables

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint table.
// orgID selalu dari RequireOrgMember context, storeID dari path;
// client tidak boleh mengirim organization_id/store_id.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, storeID, ok := scope(r)
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context atau store ID tidak valid")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	t, err := h.svc.Create(r.Context(), orgID, storeID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "name wajib diisi, capacity tidak boleh negatif")
		case errors.Is(err, ErrInvalidStatus):
			response.Error(w, http.StatusBadRequest, "Status harus available, occupied, atau reserved")
		case errors.Is(err, ErrNameExists):
			response.Error(w, http.StatusConflict, "Nama meja sudah dipakai di store ini")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat table")
		}
		return
	}

	response.JSON(w, http.StatusCreated, t)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, storeID, ok := scope(r)
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context atau store ID tidak valid")
		return
	}

	q := r.URL.Query()
	filter := Filter{
		Status: strings.TrimSpace(q.Get("status")),
		Area:   strings.TrimSpace(q.Get("area")),
	}
	list, err := h.svc.List(r.Context(), orgID, storeID, filter)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidStatus):
			response.Error(w, http.StatusBadRequest, "Status filter tidak valid")
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Parameter tidak valid")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar table")
		}
		return
	}
	if list == nil {
		list = []Table{}
	}
	response.JSON(w, http.StatusOK, list)
}

func (h *Handler) ByID(w http.ResponseWriter, r *http.Request) {
	orgID, storeID, id, ok := scopeID(r)
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context, store ID, atau table ID tidak valid")
		return
	}

	t, err := h.svc.Get(r.Context(), orgID, storeID, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Table not found")
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Parameter tidak valid")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengambil table")
		}
		return
	}
	response.JSON(w, http.StatusOK, t)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	orgID, storeID, id, ok := scopeID(r)
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context, store ID, atau table ID tidak valid")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	t, err := h.svc.Update(r.Context(), orgID, storeID, id, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "name wajib diisi, capacity tidak boleh negatif")
		case errors.Is(err, ErrNameExists):
			response.Error(w, http.StatusConflict, "Nama meja sudah dipakai di store ini")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Table not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengubah table")
		}
		return
	}
	response.JSON(w, http.StatusOK, t)
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	orgID, storeID, id, ok := scopeID(r)
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context, store ID, atau table ID tidak valid")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	t, err := h.svc.UpdateStatus(r.Context(), orgID, storeID, id, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidStatus):
			response.Error(w, http.StatusBadRequest, "Status harus available, occupied, atau reserved")
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Parameter tidak valid")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Table not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengubah status table")
		}
		return
	}
	response.JSON(w, http.StatusOK, t)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID, storeID, id, ok := scopeID(r)
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context, store ID, atau table ID tidak valid")
		return
	}

	if err := h.svc.Delete(r.Context(), orgID, storeID, id); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Table not found")
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Parameter tidak valid")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menghapus table")
		}
		return
	}
	response.JSON(w, http.StatusOK, MessageResponse{Message: "Table berhasil dihapus"})
}

// scope mengambil orgID dari context dan storeID dari path.
func scope(r *http.Request) (orgID, storeID string, ok bool) {
	orgID, ok = middleware.OrgIDFromContext(r.Context())
	if !ok {
		return "", "", false
	}
	storeID = strings.TrimSpace(r.PathValue("storeId"))
	if storeID == "" {
		return "", "", false
	}
	return orgID, storeID, true
}

// scopeID seperti scope plus id dari path.
func scopeID(r *http.Request) (orgID, storeID, id string, ok bool) {
	orgID, storeID, ok = scope(r)
	if !ok {
		return "", "", "", false
	}
	id = strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		return "", "", "", false
	}
	return orgID, storeID, id, true
}
