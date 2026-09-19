package organizations

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint organization.
type Handler struct {
	svc        *Service
	jwtManager *appjwt.Manager
}

func NewHandler(svc *Service, jwtManager *appjwt.Manager) *Handler {
	return &Handler{svc: svc, jwtManager: jwtManager}
}

func (h *Handler) CreateOrg(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	org, err := h.svc.Create(r.Context(), req.Name, req.Slug, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Name dan slug wajib diisi")
		case errors.Is(err, ErrInvalidSlug):
			response.Error(w, http.StatusBadRequest, "Slug harus minimal 3 karakter dan berupa huruf kecil, angka, atau tanda minus")
		case errors.Is(err, ErrSlugTaken):
			response.Error(w, http.StatusConflict, "Organization slug sudah digunakan")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat organization")
		}
		return
	}

	response.JSON(w, http.StatusCreated, newResponse(*org))
}

func (h *Handler) ListOrgs(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar organization")
		return
	}

	res := make([]Response, len(list))
	for i, o := range list {
		res[i] = newResponse(o)
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *Handler) GetOrgBySlug(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		response.Error(w, http.StatusBadRequest, "Invalid slug")
		return
	}

	org, err := h.svc.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "Organization not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil organization")
		return
	}

	response.JSON(w, http.StatusOK, newResponse(*org))
}

func (h *Handler) SelectOrg(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	slug := r.PathValue("slug")
	if slug == "" {
		response.Error(w, http.StatusBadRequest, "Invalid slug")
		return
	}

	org, err := h.svc.GetBySlug(r.Context(), slug)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Organization not found")
		return
	}

	isMember, err := h.svc.IsUserMember(r.Context(), org.ID, claims.UserID)
	if err != nil || !isMember {
		response.Error(w, http.StatusForbidden, "Forbidden: you are not a member of this organization")
		return
	}

	token, err := h.jwtManager.GenerateAccessWithOrg(claims.UserID, org.ID, org.Slug)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal menerbitkan token organisasi")
		return
	}

	response.JSON(w, http.StatusOK, SelectResponse{
		Token:     token,
		TokenType: "Bearer",
		Org:       newResponse(*org),
	})
}

func (h *Handler) ListOrgMembers(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	members, err := h.svc.ListMembers(r.Context(), orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil anggota organisasi")
		return
	}

	response.JSON(w, http.StatusOK, members)
}

func (h *Handler) AddOrgMember(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	req.UserID = strings.TrimSpace(req.UserID)
	if req.UserID == "" {
		response.Error(w, http.StatusBadRequest, "user_id wajib diisi")
		return
	}

	err := h.svc.AddMember(r.Context(), orgID, req.UserID)
	if err != nil {
		switch {
		case errors.Is(err, ErrAlreadyMember):
			response.Error(w, http.StatusConflict, "User sudah menjadi anggota organisasi ini")
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "user_id tidak valid")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menambahkan anggota ke organisasi")
		}
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Anggota berhasil ditambahkan"})
}

func (h *Handler) RemoveOrgMember(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	uid := r.PathValue("uid")
	if uid == "" {
		response.Error(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	err := h.svc.RemoveMember(r.Context(), orgID, uid)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotMember):
			response.Error(w, http.StatusNotFound, "User bukan anggota organisasi ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menghapus anggota dari organisasi")
		}
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Anggota berhasil dihapus"})
}
