package sales

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint sale.
// orgID selalu dari RequireOrgMember context, storeID dari path;
// client tidak boleh mengirim organization_id/store_id maupun total.
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

	userID := ""
	if claims, ok := middleware.ClaimsFromContext(r.Context()); ok && claims != nil {
		userID = claims.UserID
	}

	lines := make([]SaleLine, len(req.Items))
	for i, it := range req.Items {
		lines[i] = SaleLine{ProductID: it.ProductID, VariantID: it.VariantID, Quantity: it.Quantity}
	}
	s, err := h.svc.Create(r.Context(), orgID, storeID, userID,
		req.PaymentMethod, req.DiscountAmount, req.TaxAmount, req.Notes, lines)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "payment_method dan items (product_id, quantity > 0) wajib diisi")
		case errors.Is(err, ErrInvalidStatus):
			response.Error(w, http.StatusBadRequest, "Status tidak valid")
		case errors.Is(err, ErrStoreNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		case errors.Is(err, ErrNoAccess):
			response.Error(w, http.StatusForbidden, "Anda tidak punya akses ke store ini")
		case errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusBadRequest, "Product tidak ditemukan di store ini")
		case errors.Is(err, ErrInsufficientStock):
			response.Error(w, http.StatusConflict, "Stok tidak cukup untuk transaksi ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat sale")
		}
		return
	}

	response.JSON(w, http.StatusCreated, s)
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
	userID := ""
	if claims, ok := middleware.ClaimsFromContext(r.Context()); ok && claims != nil {
		userID = claims.UserID
	}

	filter, okFilter := parseFilter(r)
	if !okFilter {
		response.Error(w, http.StatusBadRequest, "Filter tanggal tidak valid")
		return
	}
	list, total, err := h.svc.List(r.Context(), orgID, storeID, userID, filter)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidStatus):
			response.Error(w, http.StatusBadRequest, "Status tidak valid")
		case errors.Is(err, ErrStoreNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		case errors.Is(err, ErrNoAccess):
			response.Error(w, http.StatusForbidden, "Anda tidak punya akses ke store ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar sale")
		}
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
		list = []Sale{}
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
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau sale ID")
		return
	}
	userID := ""
	if claims, ok := middleware.ClaimsFromContext(r.Context()); ok && claims != nil {
		userID = claims.UserID
	}

	s, err := h.svc.Get(r.Context(), orgID, storeID, userID, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrStoreNotFound), errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Sale not found")
		case errors.Is(err, ErrNoAccess):
			response.Error(w, http.StatusForbidden, "Anda tidak punya akses ke store ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengambil sale")
		}
		return
	}

	response.JSON(w, http.StatusOK, s)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	storeID := strings.TrimSpace(r.PathValue("storeId"))
	id := strings.TrimSpace(r.PathValue("id"))
	if storeID == "" || id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau sale ID")
		return
	}
	userID := ""
	if claims, ok := middleware.ClaimsFromContext(r.Context()); ok && claims != nil {
		userID = claims.UserID
	}

	s, err := h.svc.Cancel(r.Context(), orgID, storeID, userID, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidStatus):
			response.Error(w, http.StatusConflict, "Sale yang sudah completed/cancelled tidak bisa dibatalkan")
		case errors.Is(err, ErrStoreNotFound), errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "Sale not found")
		case errors.Is(err, ErrNoAccess):
			response.Error(w, http.StatusForbidden, "Anda tidak punya akses ke store ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membatalkan sale")
		}
		return
	}

	response.JSON(w, http.StatusOK, s)
}

// parseFilter membaca query status/from/to/page/limit.
// Tanggal RFC3339 atau YYYY-MM-DD; ok=false bila format rusak.
func parseFilter(r *http.Request) (Filter, bool) {
	q := r.URL.Query()
	var f Filter
	f.Status = strings.TrimSpace(q.Get("status"))
	var err error
	if v := strings.TrimSpace(q.Get("from")); v != "" {
		if f.From, err = parseDate(v); err != nil {
			return f, false
		}
	}
	if v := strings.TrimSpace(q.Get("to")); v != "" {
		if f.To, err = parseDate(v); err != nil {
			return f, false
		}
	}
	if p, err := strconv.Atoi(strings.TrimSpace(q.Get("page"))); err == nil {
		f.Page = p
	}
	if l, err := strconv.Atoi(strings.TrimSpace(q.Get("limit"))); err == nil {
		f.Limit = l
	}
	return f, true
}

func parseDate(v string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", v)
}
