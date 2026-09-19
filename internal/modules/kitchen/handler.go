package kitchen

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint kitchen.
// orgID selalu dari RequireOrgMember context, storeID dari path.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Queue(w http.ResponseWriter, r *http.Request) {
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
	userID := claimsUserID(r)
	list, err := h.svc.Queue(r.Context(), orgID, storeID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrStoreNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		case errors.Is(err, ErrNoAccess):
			response.Error(w, http.StatusForbidden, "Anda tidak punya akses ke store ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengambil antrian dapur")
		}
		return
	}
	if list == nil {
		list = []KitchenSale{}
	}
	response.JSON(w, http.StatusOK, QueueResponse{Data: list})
}

func (h *Handler) ByID(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	storeID := strings.TrimSpace(r.PathValue("storeId"))
	id := strings.TrimSpace(r.PathValue("kitchenSaleId"))
	if storeID == "" || id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau kitchen ID")
		return
	}
	ks, err := h.svc.Get(r.Context(), orgID, storeID, claimsUserID(r), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrStoreNotFound), errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Kitchen sale not found")
		case errors.Is(err, ErrNoAccess):
			response.Error(w, http.StatusForbidden, "Anda tidak punya akses ke store ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengambil kitchen sale")
		}
		return
	}
	response.JSON(w, http.StatusOK, ks)
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	storeID := strings.TrimSpace(r.PathValue("storeId"))
	id := strings.TrimSpace(r.PathValue("kitchenSaleId"))
	if storeID == "" || id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau kitchen ID")
		return
	}
	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Body tidak valid")
		return
	}
	ks, err := h.svc.UpdateStatus(r.Context(), orgID, storeID, claimsUserID(r), id, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidStatus), errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Status tidak valid")
		case errors.Is(err, ErrInvalidTransition):
			response.Error(w, http.StatusConflict, "Transisi status tidak diizinkan")
		case errors.Is(err, ErrSaleCancelled):
			response.Error(w, http.StatusConflict, "Sale sudah dibatalkan")
		case errors.Is(err, ErrSaleNotFound):
			response.Error(w, http.StatusNotFound, "Sale not found")
		case errors.Is(err, ErrStoreNotFound), errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Kitchen sale not found")
		case errors.Is(err, ErrNoAccess):
			response.Error(w, http.StatusForbidden, "Anda tidak punya akses ke store ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengubah status dapur")
		}
		return
	}
	response.JSON(w, http.StatusOK, ks)
}

func (h *Handler) UpdateItemStatus(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	storeID := strings.TrimSpace(r.PathValue("storeId"))
	kitchenID := strings.TrimSpace(r.PathValue("kitchenSaleId"))
	itemID := strings.TrimSpace(r.PathValue("itemId"))
	if storeID == "" || kitchenID == "" || itemID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID, kitchen ID, atau item ID")
		return
	}
	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Body tidak valid")
		return
	}
	it, err := h.svc.UpdateItemStatus(r.Context(), orgID, storeID, claimsUserID(r), kitchenID, itemID, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidStatus), errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Status tidak valid")
		case errors.Is(err, ErrInvalidTransition):
			response.Error(w, http.StatusConflict, "Transisi status tidak diizinkan")
		case errors.Is(err, ErrSaleCancelled):
			response.Error(w, http.StatusConflict, "Sale sudah dibatalkan")
		case errors.Is(err, ErrSaleNotFound):
			response.Error(w, http.StatusNotFound, "Sale not found")
		case errors.Is(err, ErrStoreNotFound), errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Kitchen item not found")
		case errors.Is(err, ErrNoAccess):
			response.Error(w, http.StatusForbidden, "Anda tidak punya akses ke store ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengubah status item dapur")
		}
		return
	}
	response.JSON(w, http.StatusOK, it)
}

func claimsUserID(r *http.Request) string {
	if claims, ok := middleware.ClaimsFromContext(r.Context()); ok && claims != nil {
		return claims.UserID
	}
	return ""
}
