package products

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint product.
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

	createdBy := ""
	if claims, ok := middleware.ClaimsFromContext(r.Context()); ok && claims != nil {
		createdBy = claims.UserID
	}

	p, err := h.svc.Create(r.Context(), orgID, storeID, createdBy,
		req.Name, req.SKU, req.Description, req.Price, req.Stock, req.Unit, req.CategoryID, req.ImageURL)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "name dan sku wajib diisi")
		case errors.Is(err, ErrInvalidPrice):
			response.Error(w, http.StatusBadRequest, "price harus >= 0")
		case errors.Is(err, ErrInvalidStock):
			response.Error(w, http.StatusBadRequest, "stock harus >= 0")
		case errors.Is(err, ErrSKUDuplicate):
			response.Error(w, http.StatusConflict, "SKU sudah dipakai di store ini")
		case errors.Is(err, ErrStoreNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat product")
		}
		return
	}

	response.JSON(w, http.StatusCreated, p)
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

	filter := parseFilter(r)
	list, total, err := h.svc.List(r.Context(), orgID, storeID, filter)
	if err != nil {
		if errors.Is(err, ErrStoreNotFound) {
			response.Error(w, http.StatusNotFound, "Store not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar product")
		return
	}

	page, limit := filter.Page, filter.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if list == nil {
		list = []Product{}
	}
	response.JSON(w, http.StatusOK, ListResponse{Data: list, Total: total, Page: page, Limit: limit})
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
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau product ID")
		return
	}

	p, err := h.svc.GetDetail(r.Context(), orgID, storeID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "Product not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil product")
		return
	}

	response.JSON(w, http.StatusOK, p)
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
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau product ID")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if req.IsActive == nil {
		response.Error(w, http.StatusBadRequest, "is_active wajib diisi")
		return
	}

	p, err := h.svc.Update(r.Context(), orgID, storeID, id,
		req.Name, req.SKU, req.Description, req.Price, req.Stock, req.Unit, req.CategoryID, req.ImageURL, *req.IsActive)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "name dan sku wajib diisi")
		case errors.Is(err, ErrInvalidPrice):
			response.Error(w, http.StatusBadRequest, "price harus >= 0")
		case errors.Is(err, ErrInvalidStock):
			response.Error(w, http.StatusBadRequest, "stock harus >= 0")
		case errors.Is(err, ErrSKUDuplicate):
			response.Error(w, http.StatusConflict, "SKU sudah dipakai di store ini")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Product not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengubah product")
		}
		return
	}

	response.JSON(w, http.StatusOK, p)
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
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau product ID")
		return
	}

	if err := h.svc.Delete(r.Context(), orgID, storeID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "Product not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal menghapus product")
		return
	}

	response.JSON(w, http.StatusOK, MessageResponse{Message: "Product berhasil dihapus"})
}

// parseFilter membaca query search/category_id/is_active/page/limit.
func parseFilter(r *http.Request) Filter {
	q := r.URL.Query()
	var f Filter
	f.Search = strings.TrimSpace(q.Get("search"))
	if c := strings.TrimSpace(q.Get("category_id")); c != "" {
		f.CategoryID = &c
	}
	if v := strings.TrimSpace(q.Get("is_active")); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			f.IsActive = &b
		}
	}
	if p, err := strconv.Atoi(strings.TrimSpace(q.Get("page"))); err == nil {
		f.Page = p
	}
	if l, err := strconv.Atoi(strings.TrimSpace(q.Get("limit"))); err == nil {
		f.Limit = l
	}
	return f
}
