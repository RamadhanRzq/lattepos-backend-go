package recipes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint recipe.
// orgID selalu dari RequireOrgMember context, storeID/productID dari path;
// client tidak boleh mengirim organization_id/store_id.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, storeID, productID, ok := recipeScope(w, r)
	if !ok {
		return
	}
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Body JSON tidak valid")
		return
	}
	createdBy := ""
	if claims, ok := middleware.ClaimsFromContext(r.Context()); ok && claims != nil {
		createdBy = claims.UserID
	}

	rec, err := h.svc.Create(r.Context(), orgID, storeID, productID, createdBy,
		req.Name, req.Version, req.YieldQuantity, req.IsActive, req.Notes, req.Items)
	if err != nil {
		writeRecipeError(w, err, "Gagal membuat resep")
		return
	}
	response.JSON(w, http.StatusCreated, rec)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, storeID, productID, ok := recipeScope(w, r)
	if !ok {
		return
	}
	list, err := h.svc.List(r.Context(), orgID, storeID, productID)
	if err != nil {
		writeRecipeError(w, err, "Gagal mengambil daftar resep")
		return
	}
	response.JSON(w, http.StatusOK, list)
}

func (h *Handler) ByID(w http.ResponseWriter, r *http.Request) {
	orgID, storeID, productID, ok := recipeScope(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(r.PathValue("recipeId"))
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid recipe ID")
		return
	}
	rec, err := h.svc.Get(r.Context(), orgID, storeID, productID, id)
	if err != nil {
		writeRecipeError(w, err, "Gagal mengambil resep")
		return
	}
	response.JSON(w, http.StatusOK, rec)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	orgID, storeID, productID, ok := recipeScope(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(r.PathValue("recipeId"))
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid recipe ID")
		return
	}
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Body JSON tidak valid")
		return
	}

	rec, err := h.svc.Update(r.Context(), orgID, storeID, productID, id,
		req.Name, req.Version, req.YieldQuantity, req.IsActive, req.Notes, req.Items)
	if err != nil {
		writeRecipeError(w, err, "Gagal mengubah resep")
		return
	}
	response.JSON(w, http.StatusOK, rec)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID, storeID, productID, ok := recipeScope(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(r.PathValue("recipeId"))
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid recipe ID")
		return
	}
	if err := h.svc.Delete(r.Context(), orgID, storeID, productID, id); err != nil {
		writeRecipeError(w, err, "Gagal menghapus resep")
		return
	}
	response.JSON(w, http.StatusOK, MessageResponse{Message: "Resep berhasil dihapus"})
}

// recipeScope mengambil org/store/product dari context + path.
func recipeScope(w http.ResponseWriter, r *http.Request) (orgID, storeID, productID string, ok bool) {
	orgID, hasOrg := middleware.OrgIDFromContext(r.Context())
	if !hasOrg {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return "", "", "", false
	}
	storeID = strings.TrimSpace(r.PathValue("storeId"))
	productID = strings.TrimSpace(r.PathValue("productId"))
	if storeID == "" || productID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID atau product ID")
		return "", "", "", false
	}
	return orgID, storeID, productID, true
}

// Activate menyalakan/mematikan satu versi resep. Body: {"is_active": true}.
func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	orgID, storeID, productID, ok := recipeScope(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(r.PathValue("recipeId"))
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid recipe ID")
		return
	}
	var req struct {
		IsActive *bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IsActive == nil {
		response.Error(w, http.StatusBadRequest, "is_active wajib diisi")
		return
	}
	rec, err := h.svc.Activate(r.Context(), orgID, storeID, productID, id, *req.IsActive)
	if err != nil {
		writeRecipeError(w, err, "Gagal mengubah status resep")
		return
	}
	response.JSON(w, http.StatusOK, rec)
}

// ListByStore melayani GET .../stores/{storeId}/recipes: seluruh resep satu store.
func (h *Handler) ListByStore(w http.ResponseWriter, r *http.Request) {
	orgID, hasOrg := middleware.OrgIDFromContext(r.Context())
	if !hasOrg {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	storeID := strings.TrimSpace(r.PathValue("storeId"))
	if storeID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid store ID")
		return
	}
	list, err := h.svc.List(r.Context(), orgID, storeID, "")
	if err != nil {
		writeRecipeError(w, err, "Gagal mengambil daftar resep")
		return
	}
	response.JSON(w, http.StatusOK, list)
}

// writeRecipeError memetakan error domain recipe ke status HTTP.
func writeRecipeError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		response.Error(w, http.StatusBadRequest, "name, minimal satu bahan, quantity > 0 wajib diisi")
	case errors.Is(err, ErrIngredientSelf):
		response.Error(w, http.StatusBadRequest, "Bahan tidak boleh produk itu sendiri")
	case errors.Is(err, ErrIngredientDuplicate):
		response.Error(w, http.StatusBadRequest, "Bahan duplikat dalam satu resep")
	case errors.Is(err, ErrVersionExists):
		response.Error(w, http.StatusConflict, "Version sudah dipakai product ini")
	case errors.Is(err, ErrProductNotFound):
		response.Error(w, http.StatusNotFound, "Product not found")
	case errors.Is(err, ErrIngredientNotFound):
		response.Error(w, http.StatusNotFound, "Bahan tidak ditemukan di store ini")
	case errors.Is(err, ErrStoreNotFound):
		response.Error(w, http.StatusNotFound, "Store not found")
	case errors.Is(err, ErrNotFound):
		response.Error(w, http.StatusNotFound, "Recipe not found")
	default:
		response.Error(w, http.StatusInternalServerError, fallback)
	}
}
