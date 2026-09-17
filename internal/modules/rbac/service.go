package rbac

import (
	"context"
	"strings"
)

// Service menangani business rules RBAC: permission, role, dan penugasannya.
// Role dan user_roles memiliki scope organisasi: seluruh operasi role/penugasan
// yang ter-scope menerima orgID (kosong berarti scope global).
type Service struct {
	permRepo PermissionRepository
	roleRepo RoleRepository
}

func NewService(permRepo PermissionRepository, roleRepo RoleRepository) *Service {
	return &Service{
		permRepo: permRepo,
		roleRepo: roleRepo,
	}
}

func (s *Service) CreatePermission(ctx context.Context, name, description string) (Permission, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if name == "" {
		return Permission{}, ErrInvalidInput
	}

	p := &Permission{
		Name:        name,
		Description: description,
	}

	if err := s.permRepo.Create(ctx, p); err != nil {
		return Permission{}, err
	}
	return *p, nil
}

func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	list, err := s.permRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]Permission, len(list))
	for i, p := range list {
		out[i] = *p
	}
	return out, nil
}

// CreateRole membuat role pada scope yang diberikan (orgID kosong = role global).
func (s *Service) CreateRole(ctx context.Context, orgID, name, description string) (Role, error) {
	orgID = strings.TrimSpace(orgID)
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if name == "" {
		return Role{}, ErrInvalidInput
	}

	r := &Role{
		Name:        name,
		Description: description,
		OrgID:       orgID,
	}

	if err := s.roleRepo.Create(ctx, r); err != nil {
		return Role{}, err
	}
	return *r, nil
}

// ListRoles mengembalikan role pada satu scope saja, sehingga role milik
// organisasi lain tidak ikut terlihat.
func (s *Service) ListRoles(ctx context.Context, orgID string) ([]Role, error) {
	list, err := s.roleRepo.List(ctx, strings.TrimSpace(orgID))
	if err != nil {
		return nil, err
	}

	out := make([]Role, len(list))
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

// GetRoleByID hanya mengembalikan role yang berada di scope orgID.
func (s *Service) GetRoleByID(ctx context.Context, orgID, id string) (Role, error) {
	orgID = strings.TrimSpace(orgID)
	id = strings.TrimSpace(id)
	if id == "" {
		return Role{}, ErrInvalidInput
	}

	r, err := s.roleRepo.FindByID(ctx, id, orgID)
	if err != nil {
		return Role{}, err
	}

	perms, err := s.roleRepo.GetPermissionsByRoleID(ctx, r.ID)
	if err != nil {
		return Role{}, err
	}
	r.Permissions = perms

	return *r, nil
}

// AssignPermissionToRole menolak role yang berada di luar scope orgID.
func (s *Service) AssignPermissionToRole(ctx context.Context, orgID, roleID, permissionID string) error {
	orgID = strings.TrimSpace(orgID)
	roleID = strings.TrimSpace(roleID)
	permissionID = strings.TrimSpace(permissionID)

	if roleID == "" || permissionID == "" {
		return ErrInvalidInput
	}

	if _, err := s.roleRepo.FindByID(ctx, roleID, orgID); err != nil {
		return err
	}

	if _, err := s.permRepo.FindByID(ctx, permissionID); err != nil {
		return err
	}

	return s.roleRepo.AssignPermission(ctx, roleID, permissionID)
}

// AssignRoleToUser menugaskan role pada scope orgID; role dari organisasi lain ditolak.
func (s *Service) AssignRoleToUser(ctx context.Context, orgID, userID, roleID string) error {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)
	roleID = strings.TrimSpace(roleID)

	if userID == "" || roleID == "" {
		return ErrInvalidInput
	}

	if _, err := s.roleRepo.FindByID(ctx, roleID, orgID); err != nil {
		return err
	}

	return s.roleRepo.AssignRoleToUser(ctx, orgID, userID, roleID)
}

// GetUserPermissions menghitung permission efektif user: role global selalu
// berlaku, role ber-scope hanya berlaku di dalam organisasinya.
func (s *Service) GetUserPermissions(ctx context.Context, orgID, userID string) (map[string]struct{}, error) {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidInput
	}

	perms, err := s.roleRepo.GetPermissionsByUserID(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}

	set := make(map[string]struct{}, len(perms))
	for _, p := range perms {
		set[p.Name] = struct{}{}
	}
	return set, nil
}

func (s *Service) HasPermission(ctx context.Context, orgID, userID, permissionName string) (bool, error) {
	set, err := s.GetUserPermissions(ctx, orgID, userID)
	if err != nil {
		return false, err
	}

	_, ok := set[permissionName]
	return ok, nil
}
