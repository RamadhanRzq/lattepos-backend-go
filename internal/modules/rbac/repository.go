package rbac

import "context"

// PermissionRepository adalah kontrak penyimpanan permission (selalu global).
type PermissionRepository interface {
	Create(ctx context.Context, p *Permission) error
	FindByID(ctx context.Context, id string) (*Permission, error)
	FindByName(ctx context.Context, name string) (*Permission, error)
	List(ctx context.Context) ([]*Permission, error)
}

// RoleRepository adalah kontrak penyimpanan role dan relasinya.
// orgID selalu menjadi bagian dari scope: string kosong berarti global,
// sehingga role milik organisasi lain tidak pernah ikut terbaca.
type RoleRepository interface {
	Create(ctx context.Context, r *Role) error
	FindByID(ctx context.Context, id, orgID string) (*Role, error)
	FindByName(ctx context.Context, name, orgID string) (*Role, error)
	List(ctx context.Context, orgID string) ([]*Role, error)
	AssignPermission(ctx context.Context, roleID, permissionID string) error
	GetPermissionsByRoleID(ctx context.Context, roleID string) ([]Permission, error)
	AssignRoleToUser(ctx context.Context, orgID, userID, roleID string) error
	GetRolesByUserID(ctx context.Context, userID string) ([]Role, error)
	GetPermissionsByUserID(ctx context.Context, orgID, userID string) ([]Permission, error)
}
