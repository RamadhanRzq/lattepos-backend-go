package variants

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint variant.
// orgID dari RequireOrgMember context; store/product/variant dari path.
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
	productID := strings.TrimSpace(r.PathValue("productId"))
	if storeID == "" || productID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau product ID")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	v, err := h.svc.Create(r.Context(), orgID, storeID, productID, req.Name, req.SKU, req.Stock)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "name wajib diisi, stock >= 0")
		case errors.Is(err, ErrSKUExists):
			response.Error(w, http.StatusConflict, "SKU sudah dipakai di store ini")
		case errors.Is(err, ErrStoreNotFound), errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusNotFound, "Product not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat variant: "+err.Error())
		}
		return
	}

	response.JSON(w, http.StatusCreated, v)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	storeID := strings.TrimSpace(r.PathValue("storeId"))
	productID := strings.TrimSpace(r.PathValue("productId"))
	if storeID == "" || productID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau product ID")
		return
	}

	list, err := h.svc.List(r.Context(), orgID, storeID, productID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Invalid store ID atau product ID")
		case errors.Is(err, ErrStoreNotFound), errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusNotFound, "Product not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar variant")
		}
		return
	}

	if list == nil {
		list = []ProductVariant{}
	}
	response.JSON(w, http.StatusOK, ListResponse{Data: list})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	storeID := strings.TrimSpace(r.PathValue("storeId"))
	productID := strings.TrimSpace(r.PathValue("productId"))
	variantID := strings.TrimSpace(r.PathValue("variantId"))
	if storeID == "" || productID == "" || variantID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID, product ID, atau variant ID")
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

	v, err := h.svc.Update(r.Context(), orgID, storeID, productID, variantID,
		req.Name, req.SKU, req.Stock, *req.IsActive)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "name wajib diisi, stock >= 0")
		case errors.Is(err, ErrSKUExists):
			response.Error(w, http.StatusConflict, "SKU sudah dipakai di store ini")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Variant not found")
		case errors.Is(err, ErrStoreNotFound), errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusNotFound, "Product not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengubah variant")
		}
		return
	}

	response.JSON(w, http.StatusOK, v)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	storeID := strings.TrimSpace(r.PathValue("storeId"))
	productID := strings.TrimSpace(r.PathValue("productId"))
	variantID := strings.TrimSpace(r.PathValue("variantId"))
	if storeID == "" || productID == "" || variantID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID, product ID, atau variant ID")
		return
	}

	if err := h.svc.Delete(r.Context(), orgID, storeID, productID, variantID); err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Invalid store ID, product ID, atau variant ID")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Variant not found")
		case errors.Is(err, ErrStoreNotFound), errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusNotFound, "Product not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menghapus variant")
		}
		return
	}

	response.JSON(w, http.StatusOK, MessageResponse{Message: "Variant berhasil dihapus"})
}
