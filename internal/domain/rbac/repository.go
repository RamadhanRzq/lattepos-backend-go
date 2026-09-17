package rbac

import "context"

type PermissionRepository interface {
	Create(ctx context.Context, p *Permission) error
	FindByID(ctx context.Context, id string) (*Permission, error)
	FindByName(ctx context.Context, name string) (*Permission, error)
	List(ctx context.Context) ([]*Permission, error)
}

type RoleRepository interface {
	Create(ctx context.Context, r *Role) error
	FindByID(ctx context.Context, id string) (*Role, error)
	FindByName(ctx context.Context, name string) (*Role, error)
	List(ctx context.Context) ([]*Role, error)
	AssignPermission(ctx context.Context, roleID, permissionID string) error
	GetPermissionsByRoleID(ctx context.Context, roleID string) ([]Permission, error)
	AssignRoleToUser(ctx context.Context, userID, roleID string) error
	GetRolesByUserID(ctx context.Context, userID string) ([]Role, error)
	GetPermissionsByUserID(ctx context.Context, userID string) ([]Permission, error)
}
