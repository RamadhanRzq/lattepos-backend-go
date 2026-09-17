package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/domain/rbac"
	"github.com/ramadhanrzq/backend-go/internal/dto"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

type RBACHandler struct {
	Service rbac.RBACService
}

func (h *RBACHandler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	p, err := h.Service.CreatePermission(r.Context(), req.Name, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, rbac.ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Name wajib diisi")
		case errors.Is(err, rbac.ErrPermissionNameTaken):
			response.Error(w, http.StatusConflict, "Permission name sudah dipakai")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat permission")
		}
		return
	}

	response.JSON(w, http.StatusCreated, dto.PermissionResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
	})
}

func (h *RBACHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	list, err := h.Service.ListPermissions(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar permission")
		return
	}

	res := make([]dto.PermissionResponse, len(list))
	for i, p := range list {
		res[i] = dto.PermissionResponse{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
		}
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *RBACHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	role, err := h.Service.CreateRole(r.Context(), req.Name, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, rbac.ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Name wajib diisi")
		case errors.Is(err, rbac.ErrRoleNameTaken):
			response.Error(w, http.StatusConflict, "Role name sudah dipakai")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat role")
		}
		return
	}

	response.JSON(w, http.StatusCreated, dto.RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		Permissions: []dto.PermissionResponse{},
		CreatedAt:   role.CreatedAt,
	})
}

func (h *RBACHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	list, err := h.Service.ListRoles(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar role")
		return
	}

	res := make([]dto.RoleResponse, len(list))
	for i, r := range list {
		res[i] = mapRoleToResponse(r)
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *RBACHandler) GetRoleByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	role, err := h.Service.GetRoleByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, rbac.ErrRoleNotFound) {
			response.Error(w, http.StatusNotFound, "Role not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil role")
		return
	}

	response.JSON(w, http.StatusOK, mapRoleToResponse(role))
}

func (h *RBACHandler) AssignPermission(w http.ResponseWriter, r *http.Request) {
	roleID := r.PathValue("id")
	if roleID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	var req dto.AssignPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.PermissionID == "" {
		response.Error(w, http.StatusBadRequest, "permission_id wajib diisi")
		return
	}

	err := h.Service.AssignPermissionToRole(r.Context(), roleID, req.PermissionID)
	if err != nil {
		switch {
		case errors.Is(err, rbac.ErrRoleNotFound):
			response.Error(w, http.StatusNotFound, "Role not found")
		case errors.Is(err, rbac.ErrPermissionNotFound):
			response.Error(w, http.StatusNotFound, "Permission not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menambahkan permission ke role")
		}
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Permission assigned successfully"})
}

func (h *RBACHandler) AssignRoleToUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req dto.AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.RoleID == "" {
		response.Error(w, http.StatusBadRequest, "role_id wajib diisi")
		return
	}

	err := h.Service.AssignRoleToUser(r.Context(), userID, req.RoleID)
	if err != nil {
		switch {
		case errors.Is(err, rbac.ErrRoleNotFound):
			response.Error(w, http.StatusNotFound, "Role not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menambahkan role ke user")
		}
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Role assigned to user successfully"})
}

func mapRoleToResponse(r rbac.Role) dto.RoleResponse {
	perms := make([]dto.PermissionResponse, len(r.Permissions))
	for i, p := range r.Permissions {
		perms[i] = dto.PermissionResponse{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
		}
	}
	return dto.RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Permissions: perms,
		CreatedAt:   r.CreatedAt,
	}
}
