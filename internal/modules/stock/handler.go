package stock

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

// Handler menangani HTTP concern untuk endpoint stock movement.
// orgID selalu dari RequireOrgMember context, storeID dari path;
// client tidak boleh mengirim organization_id/store_id.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Record(w http.ResponseWriter, r *http.Request) {
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

	var req RecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Body JSON tidak valid")
		return
	}

	createdBy := ""
	if claims, ok := middleware.ClaimsFromContext(r.Context()); ok && claims != nil {
		createdBy = claims.UserID
	}

	m, err := h.svc.Record(r.Context(), orgID, storeID,
		req.ProductID, req.VariantID, req.Type, req.Quantity,
		req.ReferenceType, req.ReferenceID, req.Notes, createdBy, req.AllowNegative)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "product_id, quantity > 0 wajib diisi")
		case errors.Is(err, ErrInvalidType):
			response.Error(w, http.StatusBadRequest, "Type tidak valid (in, out, adjustment, return)")
		case errors.Is(err, ErrInsufficient):
			response.Error(w, http.StatusConflict, "Stok tidak mencukupi")
		case errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusNotFound, "Product not found")
		case errors.Is(err, ErrVariantNotFound):
			response.Error(w, http.StatusNotFound, "Variant not found")
		case errors.Is(err, ErrStoreNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		case errors.Is(err, ErrNoAccess):
			response.Error(w, http.StatusForbidden, "Anda tidak punya akses ke store ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mencatat movement")
		}
		return
	}

	response.JSON(w, http.StatusCreated, m)
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
		response.Error(w, http.StatusBadRequest, "Filter tanggal/type tidak valid")
		return
	}
	list, total, err := h.svc.List(r.Context(), orgID, storeID, userID, filter)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidType):
			response.Error(w, http.StatusBadRequest, "Type tidak valid (in, out, adjustment, return)")
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Filter tidak valid")
		case errors.Is(err, ErrStoreNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		case errors.Is(err, ErrNoAccess):
			response.Error(w, http.StatusForbidden, "Anda tidak punya akses ke store ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar movement")
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
		list = []StockMovement{}
	}
	response.JSON(w, http.StatusOK, ListResponse{Data: list, Total: total, Page: page, Limit: limit})
}

func (h *Handler) StockByProduct(w http.ResponseWriter, r *http.Request) {
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
	userID := ""
	if claims, ok := middleware.ClaimsFromContext(r.Context()); ok && claims != nil {
		userID = claims.UserID
	}

	sum, history, err := h.svc.StockByProduct(r.Context(), orgID, storeID, userID, productID)
	if err != nil {
		switch {
		case errors.Is(err, ErrProductNotFound), errors.Is(err, ErrStoreNotFound):
			response.Error(w, http.StatusNotFound, "Product not found")
		case errors.Is(err, ErrNoAccess):
			response.Error(w, http.StatusForbidden, "Anda tidak punya akses ke store ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengambil stok produk")
		}
		return
	}

	response.JSON(w, http.StatusOK, StockResponse{Summary: *sum, History: history})
}

// parseFilter membaca query product_id/type/from/to/page/limit.
// Tanggal RFC3339 atau YYYY-MM-DD; ok=false bila format rusak.
func parseFilter(r *http.Request) (Filter, bool) {
	q := r.URL.Query()
	var f Filter
	f.ProductID = strings.TrimSpace(q.Get("product_id"))
	f.Type = strings.TrimSpace(q.Get("type"))
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
