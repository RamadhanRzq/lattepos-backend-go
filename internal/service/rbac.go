package service

import (
	"context"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/domain/rbac"
)

type RBACService struct {
	permRepo rbac.PermissionRepository
	roleRepo rbac.RoleRepository
}

var _ rbac.RBACService = (*RBACService)(nil)

func NewRBACService(permRepo rbac.PermissionRepository, roleRepo rbac.RoleRepository) *RBACService {
	return &RBACService{
		permRepo: permRepo,
		roleRepo: roleRepo,
	}
}

func (s *RBACService) CreatePermission(ctx context.Context, name, description string) (rbac.Permission, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if name == "" {
		return rbac.Permission{}, rbac.ErrInvalidInput
	}

	p := &rbac.Permission{
		Name:        name,
		Description: description,
	}

	if err := s.permRepo.Create(ctx, p); err != nil {
		return rbac.Permission{}, err
	}
	return *p, nil
}

func (s *RBACService) ListPermissions(ctx context.Context) ([]rbac.Permission, error) {
	list, err := s.permRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]rbac.Permission, len(list))
	for i, p := range list {
		out[i] = *p
	}
	return out, nil
}

func (s *RBACService) CreateRole(ctx context.Context, name, description string) (rbac.Role, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if name == "" {
		return rbac.Role{}, rbac.ErrInvalidInput
	}

	r := &rbac.Role{
		Name:        name,
		Description: description,
	}

	if err := s.roleRepo.Create(ctx, r); err != nil {
		return rbac.Role{}, err
	}
	return *r, nil
}

func (s *RBACService) ListRoles(ctx context.Context) ([]rbac.Role, error) {
	list, err := s.roleRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]rbac.Role, len(list))
	for i, r := range list {
		perms, err := s.roleRepo.GetPermissionsByRoleID(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		r.Permissions = perms
		out[i] = *r
	}
	return out, nil
}

func (s *RBACService) GetRoleByID(ctx context.Context, id string) (rbac.Role, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return rbac.Role{}, rbac.ErrInvalidInput
	}

	r, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		return rbac.Role{}, err
	}

	perms, err := s.roleRepo.GetPermissionsByRoleID(ctx, r.ID)
	if err != nil {
		return rbac.Role{}, err
	}
	r.Permissions = perms

	return *r, nil
}

func (s *RBACService) AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	roleID = strings.TrimSpace(roleID)
	permissionID = strings.TrimSpace(permissionID)

	if roleID == "" || permissionID == "" {
		return rbac.ErrInvalidInput
	}

	if _, err := s.roleRepo.FindByID(ctx, roleID); err != nil {
		return err
	}

	if _, err := s.permRepo.FindByID(ctx, permissionID); err != nil {
		return err
	}

	return s.roleRepo.AssignPermission(ctx, roleID, permissionID)
}

func (s *RBACService) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	userID = strings.TrimSpace(userID)
	roleID = strings.TrimSpace(roleID)

	if userID == "" || roleID == "" {
		return rbac.ErrInvalidInput
	}

	if _, err := s.roleRepo.FindByID(ctx, roleID); err != nil {
		return err
	}

	return s.roleRepo.AssignRoleToUser(ctx, userID, roleID)
}

func (s *RBACService) GetUserPermissions(ctx context.Context, userID string) (map[string]struct{}, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, rbac.ErrInvalidInput
	}

	perms, err := s.roleRepo.GetPermissionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	set := make(map[string]struct{}, len(perms))
	for _, p := range perms {
		set[p.Name] = struct{}{}
	}
	return set, nil
}

func (s *RBACService) HasPermission(ctx context.Context, userID, permissionName string) (bool, error) {
	set, err := s.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	_, ok := set[permissionName]
	return ok, nil
}
