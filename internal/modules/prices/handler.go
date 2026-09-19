package prices

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint price.
// orgID dari RequireOrgMember context; store/product/price dari path.
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
	minQty := req.MinQuantity
	if minQty == 0 {
		minQty = 1
	}

	p, err := h.svc.Create(r.Context(), orgID, storeID, productID,
		req.VariantID, req.PriceType, req.Price, minQty, req.ValidFrom, req.ValidUntil)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "price >= 0, min_quantity >= 1, rentang valid")
		case errors.Is(err, ErrStoreNotFound), errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusNotFound, "Product not found")
		case errors.Is(err, ErrVariantNotFound):
			response.Error(w, http.StatusNotFound, "Variant not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat price")
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
	productID := strings.TrimSpace(r.PathValue("productId"))
	if storeID == "" || productID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau product ID")
		return
	}
	onlyActive := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("active")), "true")

	list, err := h.svc.List(r.Context(), orgID, storeID, productID, onlyActive)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Invalid store ID atau product ID")
		case errors.Is(err, ErrStoreNotFound), errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusNotFound, "Product not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar price")
		}
		return
	}

	if list == nil {
		list = []ProductPrice{}
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
	priceID := strings.TrimSpace(r.PathValue("priceId"))
	if storeID == "" || productID == "" || priceID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID, product ID, atau price ID")
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
	minQty := req.MinQuantity
	if minQty == 0 {
		minQty = 1
	}

	p, err := h.svc.Update(r.Context(), orgID, storeID, productID, priceID,
		req.VariantID, req.PriceType, req.Price, minQty, *req.IsActive, req.ValidFrom, req.ValidUntil)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "price >= 0, min_quantity >= 1, rentang valid")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Price not found")
		case errors.Is(err, ErrStoreNotFound), errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusNotFound, "Product not found")
		case errors.Is(err, ErrVariantNotFound):
			response.Error(w, http.StatusNotFound, "Variant not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengubah price")
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
	productID := strings.TrimSpace(r.PathValue("productId"))
	priceID := strings.TrimSpace(r.PathValue("priceId"))
	if storeID == "" || productID == "" || priceID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID, product ID, atau price ID")
		return
	}

	if err := h.svc.Delete(r.Context(), orgID, storeID, productID, priceID); err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Invalid store ID, product ID, atau price ID")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Price not found")
		case errors.Is(err, ErrStoreNotFound), errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusNotFound, "Product not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menghapus price")
		}
		return
	}

	response.JSON(w, http.StatusOK, MessageResponse{Message: "Price berhasil dihapus"})
}
