package rbac

import "context"

type RBACService interface {
	CreatePermission(ctx context.Context, name, description string) (Permission, error)
	ListPermissions(ctx context.Context) ([]Permission, error)
	CreateRole(ctx context.Context, name, description string) (Role, error)
	ListRoles(ctx context.Context) ([]Role, error)
	GetRoleByID(ctx context.Context, id string) (Role, error)
	AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error
	AssignRoleToUser(ctx context.Context, userID, roleID string) error
	GetUserPermissions(ctx context.Context, userID string) (map[string]struct{}, error)
	HasPermission(ctx context.Context, userID, permissionName string) (bool, error)
}
