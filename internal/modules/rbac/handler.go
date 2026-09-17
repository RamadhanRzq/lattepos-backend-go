package rbac

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// Handler menangani HTTP concern untuk endpoint RBAC. Endpoint org-scoped
// memakai method *Org* yang mengambil organisasi dari request context,
// sehingga role tidak pernah bocor antar organisasi.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	var req CreatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	p, err := h.svc.CreatePermission(r.Context(), req.Name, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Name wajib diisi")
		case errors.Is(err, ErrPermissionNameTaken):
			response.Error(w, http.StatusConflict, "Permission name sudah dipakai")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat permission")
		}
		return
	}

	response.JSON(w, http.StatusCreated, newPermissionResponse(p))
}

func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListPermissions(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar permission")
		return
	}

	res := make([]PermissionResponse, len(list))
	for i, p := range list {
		res[i] = newPermissionResponse(p)
	}
	response.JSON(w, http.StatusOK, res)
}

// CreateRole membuat role global (org_id NULL).
func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	h.createRole(w, r, "")
}

// CreateOrgRole membuat role milik organisasi di path.
func (h *Handler) CreateOrgRole(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	h.createRole(w, r, orgID)
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request, orgID string) {
	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	role, err := h.svc.CreateRole(r.Context(), orgID, req.Name, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "Name wajib diisi")
		case errors.Is(err, ErrRoleNameTaken):
			response.Error(w, http.StatusConflict, "Role name sudah dipakai")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal membuat role")
		}
		return
	}

	role.Permissions = []Permission{}
	response.JSON(w, http.StatusCreated, newRoleResponse(role))
}

// ListRoles mengembalikan role global saja.
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	h.listRoles(w, r, "")
}

// ListOrgRoles mengembalikan role milik organisasi di path.
func (h *Handler) ListOrgRoles(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	h.listRoles(w, r, orgID)
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request, orgID string) {
	list, err := h.svc.ListRoles(r.Context(), orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar role")
		return
	}

	res := make([]RoleResponse, len(list))
	for i, role := range list {
		res[i] = newRoleResponse(role)
	}
	response.JSON(w, http.StatusOK, res)
}

// GetRoleByID mengambil role global.
func (h *Handler) GetRoleByID(w http.ResponseWriter, r *http.Request) {
	h.getRoleByID(w, r, "")
}

// GetOrgRoleByID mengambil role milik organisasi di path.
func (h *Handler) GetOrgRoleByID(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	h.getRoleByID(w, r, orgID)
}

func (h *Handler) getRoleByID(w http.ResponseWriter, r *http.Request, orgID string) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	role, err := h.svc.GetRoleByID(r.Context(), orgID, id)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			response.Error(w, http.StatusNotFound, "Role not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil role")
		return
	}

	response.JSON(w, http.StatusOK, newRoleResponse(role))
}

// AssignPermission menambahkan permission ke role global.
func (h *Handler) AssignPermission(w http.ResponseWriter, r *http.Request) {
	h.assignPermission(w, r, "")
}

// AssignOrgPermission menambahkan permission ke role milik organisasi di path.
func (h *Handler) AssignOrgPermission(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	h.assignPermission(w, r, orgID)
}

func (h *Handler) assignPermission(w http.ResponseWriter, r *http.Request, orgID string) {
	roleID := r.PathValue("id")
	if roleID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	var req AssignPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.PermissionID == "" {
		response.Error(w, http.StatusBadRequest, "permission_id wajib diisi")
		return
	}

	err := h.svc.AssignPermissionToRole(r.Context(), orgID, roleID, req.PermissionID)
	if err != nil {
		switch {
		case errors.Is(err, ErrRoleNotFound):
			response.Error(w, http.StatusNotFound, "Role not found")
		case errors.Is(err, ErrPermissionNotFound):
			response.Error(w, http.StatusNotFound, "Permission not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menambahkan permission ke role")
		}
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Permission assigned successfully"})
}

// AssignRoleToUser menugaskan role global ke user.
func (h *Handler) AssignRoleToUser(w http.ResponseWriter, r *http.Request) {
	h.assignRoleToUser(w, r, "")
}

// AssignOrgRoleToUser menugaskan role ke user di dalam organisasi di path.
func (h *Handler) AssignOrgRoleToUser(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusBadRequest, "Organization context missing")
		return
	}
	h.assignRoleToUser(w, r, orgID)
}

func (h *Handler) assignRoleToUser(w http.ResponseWriter, r *http.Request, orgID string) {
	userID := r.PathValue("id")
	if userID == "" {
		response.Error(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.RoleID == "" {
		response.Error(w, http.StatusBadRequest, "role_id wajib diisi")
		return
	}

	err := h.svc.AssignRoleToUser(r.Context(), orgID, userID, req.RoleID)
	if err != nil {
		switch {
		case errors.Is(err, ErrRoleNotFound):
			response.Error(w, http.StatusNotFound, "Role not found")
		default:
			response.Error(w, http.StatusInternalServerError, "Gagal menambahkan role ke user")
		}
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Role assigned to user successfully"})
}
