package rbac

// CreatePermissionRequest adalah payload POST /permissions.
type CreatePermissionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CreateRoleRequest adalah payload POST /roles.
type CreateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AssignPermissionRequest adalah payload POST /roles/{id}/permissions.
type AssignPermissionRequest struct {
	PermissionID string `json:"permission_id"`
}

// AssignRoleRequest adalah payload POST /users/{id}/roles.
type AssignRoleRequest struct {
	RoleID string `json:"role_id"`
}
