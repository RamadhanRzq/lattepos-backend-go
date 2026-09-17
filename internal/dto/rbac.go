package dto

import "time"

type CreatePermissionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AssignPermissionRequest struct {
	PermissionID string `json:"permission_id"`
}

type AssignRoleRequest struct {
	RoleID string `json:"role_id"`
}

type PermissionResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type RoleResponse struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Permissions []PermissionResponse `json:"permissions"`
	CreatedAt   time.Time            `json:"created_at"`
}
