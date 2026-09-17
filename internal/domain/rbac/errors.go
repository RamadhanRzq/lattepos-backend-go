package rbac

import "errors"

var (
	ErrNotFound            = errors.New("rbac: not found")
	ErrPermissionNotFound  = errors.New("rbac: permission not found")
	ErrRoleNotFound        = errors.New("rbac: role not found")
	ErrPermissionNameTaken = errors.New("rbac: permission name already taken")
	ErrRoleNameTaken       = errors.New("rbac: role name already taken")
	ErrInvalidInput        = errors.New("rbac: invalid input")
	ErrAlreadyAssigned     = errors.New("rbac: already assigned")
)
