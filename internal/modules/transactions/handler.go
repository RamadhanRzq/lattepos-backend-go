package transactions

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern laporan transaksi harian.
// orgID dari RequireOrgMember context, storeID dari path, userID dari JWT.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Daily melayani GET .../transactions/daily?date=YYYY-MM-DD.
// Tanpa date = hari ini. Format tanggal rusak = 400.
func (h *Handler) Daily(w http.ResponseWriter, r *http.Request) {
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

	report, err := h.svc.Daily(r.Context(), orgID, storeID, userID, r.URL.Query().Get("date"))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Format date tidak valid (YYYY-MM-DD)")
		case errors.Is(err, ErrStoreNotFound):
			response.Error(w, http.StatusNotFound, "Store not found")
		case errors.Is(err, ErrNoAccess):
			response.Error(w, http.StatusForbidden, "Anda tidak punya akses ke store ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal mengambil laporan harian")
		}
		return
	}

	response.JSON(w, http.StatusOK, report)
}
