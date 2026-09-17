package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/domain/rbac"
	"github.com/ramadhanrzq/backend-go/internal/service"
)

type mockPermRepo struct {
	createFn   func(ctx context.Context, p *rbac.Permission) error
	findByIDFn func(ctx context.Context, id string) (*rbac.Permission, error)
	findByNameFn func(ctx context.Context, name string) (*rbac.Permission, error)
	listFn     func(ctx context.Context) ([]*rbac.Permission, error)
}

func (m *mockPermRepo) Create(ctx context.Context, p *rbac.Permission) error {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	return nil
}

func (m *mockPermRepo) FindByID(ctx context.Context, id string) (*rbac.Permission, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, rbac.ErrPermissionNotFound
}

func (m *mockPermRepo) FindByName(ctx context.Context, name string) (*rbac.Permission, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(ctx, name)
	}
	return nil, rbac.ErrPermissionNotFound
}

func (m *mockPermRepo) List(ctx context.Context) ([]*rbac.Permission, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

type mockRoleRepo struct {
	createFn             func(ctx context.Context, r *rbac.Role) error
	findByIDFn           func(ctx context.Context, id string) (*rbac.Role, error)
	findByNameFn         func(ctx context.Context, name string) (*rbac.Role, error)
	listFn               func(ctx context.Context) ([]*rbac.Role, error)
	assignPermissionFn   func(ctx context.Context, roleID, permissionID string) error
	getPermsByRoleIDFn   func(ctx context.Context, roleID string) ([]rbac.Permission, error)
	assignRoleToUserFn   func(ctx context.Context, userID, roleID string) error
	getRolesByUserIDFn   func(ctx context.Context, userID string) ([]rbac.Role, error)
	getPermsByUserIDFn   func(ctx context.Context, userID string) ([]rbac.Permission, error)
}

func (m *mockRoleRepo) Create(ctx context.Context, r *rbac.Role) error {
	if m.createFn != nil {
		return m.createFn(ctx, r)
	}
	return nil
}

func (m *mockRoleRepo) FindByID(ctx context.Context, id string) (*rbac.Role, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, rbac.ErrRoleNotFound
}

func (m *mockRoleRepo) FindByName(ctx context.Context, name string) (*rbac.Role, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(ctx, name)
	}
	return nil, rbac.ErrRoleNotFound
}

func (m *mockRoleRepo) List(ctx context.Context) ([]*rbac.Role, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

func (m *mockRoleRepo) AssignPermission(ctx context.Context, roleID, permissionID string) error {
	if m.assignPermissionFn != nil {
		return m.assignPermissionFn(ctx, roleID, permissionID)
	}
	return nil
}

func (m *mockRoleRepo) GetPermissionsByRoleID(ctx context.Context, roleID string) ([]rbac.Permission, error) {
	if m.getPermsByRoleIDFn != nil {
		return m.getPermsByRoleIDFn(ctx, roleID)
	}
	return nil, nil
}

func (m *mockRoleRepo) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	if m.assignRoleToUserFn != nil {
		return m.assignRoleToUserFn(ctx, userID, roleID)
	}
	return nil
}

func (m *mockRoleRepo) GetRolesByUserID(ctx context.Context, userID string) ([]rbac.Role, error) {
	if m.getRolesByUserIDFn != nil {
		return m.getRolesByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockRoleRepo) GetPermissionsByUserID(ctx context.Context, userID string) ([]rbac.Permission, error) {
	if m.getPermsByUserIDFn != nil {
		return m.getPermsByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func TestRBACService_CreatePermission(t *testing.T) {
	tests := []struct {
		name        string
		permName    string
		description string
		createErr   error
		wantErr     error
	}{
		{
			name:        "valid permission",
			permName:    "users:read",
			description: "Read users",
			createErr:   nil,
			wantErr:     nil,
		},
		{
			name:        "empty name returns ErrInvalidInput",
			permName:    "   ",
			description: "empty",
			wantErr:     rbac.ErrInvalidInput,
		},
		{
			name:        "duplicate name returns ErrPermissionNameTaken",
			permName:    "users:read",
			description: "duplicate",
			createErr:   rbac.ErrPermissionNameTaken,
			wantErr:     rbac.ErrPermissionNameTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permMock := &mockPermRepo{
				createFn: func(ctx context.Context, p *rbac.Permission) error {
					if tt.createErr != nil {
						return tt.createErr
					}
					p.ID = "p-123"
					p.CreatedAt = time.Now()
					p.UpdatedAt = time.Now()
					return nil
				},
			}
			roleMock := &mockRoleRepo{}
			svc := service.NewRBACService(permMock, roleMock)

			got, err := svc.CreatePermission(context.Background(), tt.permName, tt.description)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr == nil && got.ID == "" {
				t.Fatalf("expected non-empty ID")
			}
		})
	}
}

func TestRBACService_CreateRole(t *testing.T) {
	tests := []struct {
		name        string
		roleName    string
		description string
		createErr   error
		wantErr     error
	}{
		{
			name:        "valid role",
			roleName:    "admin",
			description: "Admin role",
			createErr:   nil,
			wantErr:     nil,
		},
		{
			name:        "empty name returns ErrInvalidInput",
			roleName:    "",
			description: "empty",
			wantErr:     rbac.ErrInvalidInput,
		},
		{
			name:        "duplicate name returns ErrRoleNameTaken",
			roleName:    "admin",
			description: "dup",
			createErr:   rbac.ErrRoleNameTaken,
			wantErr:     rbac.ErrRoleNameTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roleMock := &mockRoleRepo{
				createFn: func(ctx context.Context, r *rbac.Role) error {
					if tt.createErr != nil {
						return tt.createErr
					}
					r.ID = "r-123"
					return nil
				},
			}
			svc := service.NewRBACService(&mockPermRepo{}, roleMock)

			got, err := svc.CreateRole(context.Background(), tt.roleName, tt.description)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr == nil && got.ID == "" {
				t.Fatalf("expected non-empty ID")
			}
		})
	}
}

func TestRBACService_AssignPermissionToRole(t *testing.T) {
	tests := []struct {
		name         string
		roleID       string
		permissionID string
		findRoleErr  error
		findPermErr  error
		assignErr    error
		wantErr      error
	}{
		{
			name:         "successful assign",
			roleID:       "role-1",
			permissionID: "perm-1",
			wantErr:      nil,
		},
		{
			name:         "empty role id",
			roleID:       "",
			permissionID: "perm-1",
			wantErr:      rbac.ErrInvalidInput,
		},
		{
			name:         "role not found",
			roleID:       "role-404",
			permissionID: "perm-1",
			findRoleErr:  rbac.ErrRoleNotFound,
			wantErr:      rbac.ErrRoleNotFound,
		},
		{
			name:         "permission not found",
			roleID:       "role-1",
			permissionID: "perm-404",
			findPermErr:  rbac.ErrPermissionNotFound,
			wantErr:      rbac.ErrPermissionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permMock := &mockPermRepo{
				findByIDFn: func(ctx context.Context, id string) (*rbac.Permission, error) {
					if tt.findPermErr != nil {
						return nil, tt.findPermErr
					}
					return &rbac.Permission{ID: id, Name: "perm"}, nil
				},
			}
			roleMock := &mockRoleRepo{
				findByIDFn: func(ctx context.Context, id string) (*rbac.Role, error) {
					if tt.findRoleErr != nil {
						return nil, tt.findRoleErr
					}
					return &rbac.Role{ID: id, Name: "role"}, nil
				},
				assignPermissionFn: func(ctx context.Context, roleID, permissionID string) error {
					return tt.assignErr
				},
			}

			svc := service.NewRBACService(permMock, roleMock)
			err := svc.AssignPermissionToRole(context.Background(), tt.roleID, tt.permissionID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestRBACService_HasPermission(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		permissionName string
		userPerms      []rbac.Permission
		repoErr        error
		wantResult     bool
		wantErr        error
	}{
		{
			name:           "user has permission",
			userID:         "user-1",
			permissionName: "users:read",
			userPerms: []rbac.Permission{
				{ID: "p-1", Name: "users:read"},
				{ID: "p-2", Name: "orders:create"},
			},
			wantResult: true,
			wantErr:    nil,
		},
		{
			name:           "user does not have permission",
			userID:         "user-1",
			permissionName: "users:delete",
			userPerms: []rbac.Permission{
				{ID: "p-1", Name: "users:read"},
			},
			wantResult: false,
			wantErr:    nil,
		},
		{
			name:           "empty user ID",
			userID:         "",
			permissionName: "users:read",
			wantResult:     false,
			wantErr:        rbac.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roleMock := &mockRoleRepo{
				getPermsByUserIDFn: func(ctx context.Context, userID string) ([]rbac.Permission, error) {
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return tt.userPerms, nil
				},
			}
			svc := service.NewRBACService(&mockPermRepo{}, roleMock)

			got, err := svc.HasPermission(context.Background(), tt.userID, tt.permissionName)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if got != tt.wantResult {
				t.Fatalf("expected %v, got %v", tt.wantResult, got)
			}
		})
	}
}
