package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/domain/organization"
	"github.com/ramadhanrzq/backend-go/internal/dto"
	"github.com/ramadhanrzq/backend-go/internal/middleware"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

type OrgHandler struct {
	Service    organization.OrgService
	JWTManager *appjwt.Manager
}

func (h *OrgHandler) CreateOrg(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req dto.CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	org, err := h.Service.Create(r.Context(), req.Name, req.Slug, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, organization.ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Name dan slug wajib diisi")
		case errors.Is(err, organization.ErrInvalidSlug):
			response.Error(w, http.StatusBadRequest, "Slug harus minimal 3 karakter dan berupa huruf kecil, angka, atau tanda minus")
		case errors.Is(err, organization.ErrSlugTaken):
			response.Error(w, http.StatusConflict, "Organization slug sudah digunakan")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat organization")
		}
		return
	}

	response.JSON(w, http.StatusCreated, dto.OrgResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		CreatedAt: org.CreatedAt,
	})
}

func (h *OrgHandler) ListOrgs(w http.ResponseWriter, r *http.Request) {
	list, err := h.Service.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar organization")
		return
	}

	res := make([]dto.OrgResponse, len(list))
	for i, o := range list {
		res[i] = dto.OrgResponse{
			ID:        o.ID,
			Name:      o.Name,
			Slug:      o.Slug,
			CreatedAt: o.CreatedAt,
		}
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *OrgHandler) GetOrgBySlug(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		response.Error(w, http.StatusBadRequest, "Invalid slug")
		return
	}

	org, err := h.Service.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, organization.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "Organization not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil organization")
		return
	}

	response.JSON(w, http.StatusOK, dto.OrgResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		CreatedAt: org.CreatedAt,
	})
}

func (h *OrgHandler) SelectOrg(w http.ResponseWriter, r *http.Request) {
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

	org, err := h.Service.GetBySlug(r.Context(), slug)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Organization not found")
		return
	}

	isMember, err := h.Service.IsUserMember(r.Context(), org.ID, claims.UserID)
	if err != nil || !isMember {
		response.Error(w, http.StatusForbidden, "Forbidden: you are not a member of this organization")
		return
	}

	token, err := h.JWTManager.GenerateWithOrg(claims.UserID, claims.Username, claims.Email, claims.Name, org.ID, org.Slug)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal menerbitkan token organisasi")
		return
	}

	response.JSON(w, http.StatusOK, dto.SelectOrgResponse{
		Token:     token,
		TokenType: "Bearer",
		Org: dto.OrgResponse{
			ID:        org.ID,
			Name:      org.Name,
			Slug:      org.Slug,
			CreatedAt: org.CreatedAt,
		},
	})
}

func (h *OrgHandler) ListOrgMembers(w http.ResponseWriter, r *http.Request) {
	org, ok := middleware.OrgFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	members, err := h.Service.ListMembers(r.Context(), org.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil anggota organisasi")
		return
	}

	response.JSON(w, http.StatusOK, members)
}

func (h *OrgHandler) AddOrgMember(w http.ResponseWriter, r *http.Request) {
	org, ok := middleware.OrgFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	var req dto.AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	req.UserID = strings.TrimSpace(req.UserID)
	if req.UserID == "" {
		response.Error(w, http.StatusBadRequest, "user_id wajib diisi")
		return
	}

	err := h.Service.AddMember(r.Context(), org.ID, req.UserID)
	if err != nil {
		switch {
		case errors.Is(err, organization.ErrAlreadyMember):
			response.Error(w, http.StatusConflict, "User sudah menjadi anggota organisasi ini")
		case errors.Is(err, organization.ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "user_id tidak valid")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menambahkan anggota ke organisasi")
		}
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Anggota berhasil ditambahkan"})
}

func (h *OrgHandler) RemoveOrgMember(w http.ResponseWriter, r *http.Request) {
	org, ok := middleware.OrgFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}

	uid := r.PathValue("uid")
	if uid == "" {
		response.Error(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	err := h.Service.RemoveMember(r.Context(), org.ID, uid)
	if err != nil {
		switch {
		case errors.Is(err, organization.ErrNotMember):
			response.Error(w, http.StatusNotFound, "User bukan anggota organisasi ini")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menghapus anggota dari organisasi")
		}
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Anggota berhasil dihapus"})
}
