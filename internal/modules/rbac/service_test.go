package rbac_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/rbac"
)

type mockPermRepo struct {
	createFn     func(ctx context.Context, p *rbac.Permission) error
	findByIDFn   func(ctx context.Context, id string) (*rbac.Permission, error)
	findByNameFn func(ctx context.Context, name string) (*rbac.Permission, error)
	listFn       func(ctx context.Context) ([]*rbac.Permission, error)
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
	createFn           func(ctx context.Context, r *rbac.Role) error
	findByIDFn         func(ctx context.Context, id, orgID string) (*rbac.Role, error)
	findByNameFn       func(ctx context.Context, name, orgID string) (*rbac.Role, error)
	listFn             func(ctx context.Context, orgID string) ([]*rbac.Role, error)
	assignPermissionFn func(ctx context.Context, roleID, permissionID string) error
	getPermsByRoleIDFn func(ctx context.Context, roleID string) ([]rbac.Permission, error)
	assignRoleToUserFn func(ctx context.Context, orgID, userID, roleID string) error
	getRolesByUserIDFn func(ctx context.Context, userID string) ([]rbac.Role, error)
	getPermsByUserIDFn func(ctx context.Context, orgID, userID string) ([]rbac.Permission, error)
}

func (m *mockRoleRepo) Create(ctx context.Context, r *rbac.Role) error {
	if m.createFn != nil {
		return m.createFn(ctx, r)
	}
	return nil
}

func (m *mockRoleRepo) FindByID(ctx context.Context, id, orgID string) (*rbac.Role, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id, orgID)
	}
	return nil, rbac.ErrRoleNotFound
}

func (m *mockRoleRepo) FindByName(ctx context.Context, name, orgID string) (*rbac.Role, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(ctx, name, orgID)
	}
	return nil, rbac.ErrRoleNotFound
}

func (m *mockRoleRepo) List(ctx context.Context, orgID string) ([]*rbac.Role, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID)
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

func (m *mockRoleRepo) AssignRoleToUser(ctx context.Context, orgID, userID, roleID string) error {
	if m.assignRoleToUserFn != nil {
		return m.assignRoleToUserFn(ctx, orgID, userID, roleID)
	}
	return nil
}

func (m *mockRoleRepo) GetRolesByUserID(ctx context.Context, userID string) ([]rbac.Role, error) {
	if m.getRolesByUserIDFn != nil {
		return m.getRolesByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockRoleRepo) GetPermissionsByUserID(ctx context.Context, orgID, userID string) ([]rbac.Permission, error) {
	if m.getPermsByUserIDFn != nil {
		return m.getPermsByUserIDFn(ctx, orgID, userID)
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
			svc := rbac.NewService(permMock, roleMock)

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
			svc := rbac.NewService(&mockPermRepo{}, roleMock)

			got, err := svc.CreateRole(context.Background(), "", tt.roleName, tt.description)
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
		orgID        string
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
				findByIDFn: func(ctx context.Context, id, orgID string) (*rbac.Role, error) {
					if tt.findRoleErr != nil {
						return nil, tt.findRoleErr
					}
					return &rbac.Role{ID: id, Name: "role", OrgID: orgID}, nil
				},
				assignPermissionFn: func(ctx context.Context, roleID, permissionID string) error {
					return tt.assignErr
				},
			}

			svc := rbac.NewService(permMock, roleMock)
			err := svc.AssignPermissionToRole(context.Background(), tt.orgID, tt.roleID, tt.permissionID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestRBACService_HasPermission(t *testing.T) {
	tests := []struct {
		name           string
		orgID          string
		userID         string
		permissionName string
		userPerms      []rbac.Permission
		repoErr        error
		wantResult     bool
		wantErr        error
	}{
		{
			name:           "user has permission",
			orgID:          "",
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
			orgID:          "",
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
			orgID:          "",
			userID:         "",
			permissionName: "users:read",
			wantResult:     false,
			wantErr:        rbac.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roleMock := &mockRoleRepo{
				getPermsByUserIDFn: func(ctx context.Context, orgID, userID string) ([]rbac.Permission, error) {
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return tt.userPerms, nil
				},
			}
			svc := rbac.NewService(&mockPermRepo{}, roleMock)

			got, err := svc.HasPermission(context.Background(), tt.orgID, tt.userID, tt.permissionName)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if got != tt.wantResult {
				t.Fatalf("expected %v, got %v", tt.wantResult, got)
			}
		})
	}
}

func TestService_OrgScopeIsolation(t *testing.T) {
	const (
		orgA   = "org-a"
		orgB   = "org-b"
		roleIn = "role-a"
	)

	t.Run("role milik organisasi lain tidak bisa dibaca", func(t *testing.T) {
		roleMock := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id, orgID string) (*rbac.Role, error) {
				if orgID != orgA {
					return nil, rbac.ErrRoleNotFound
				}
				return &rbac.Role{ID: id, Name: "admin", OrgID: orgID}, nil
			},
		}
		svc := rbac.NewService(&mockPermRepo{}, roleMock)

		if _, err := svc.GetRoleByID(context.Background(), orgA, roleIn); err != nil {
			t.Fatalf("role di organisasinya sendiri harus terbaca: %v", err)
		}
		if _, err := svc.GetRoleByID(context.Background(), orgB, roleIn); !errors.Is(err, rbac.ErrRoleNotFound) {
			t.Fatalf("role organisasi lain harus ErrRoleNotFound, got %v", err)
		}
	})

	t.Run("role organisasi lain tidak bisa diberi permission", func(t *testing.T) {
		permMock := &mockPermRepo{
			findByIDFn: func(ctx context.Context, id string) (*rbac.Permission, error) {
				return &rbac.Permission{ID: id, Name: "users:read"}, nil
			},
		}
		roleMock := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id, orgID string) (*rbac.Role, error) {
				if orgID != orgA {
					return nil, rbac.ErrRoleNotFound
				}
				return &rbac.Role{ID: id, Name: "admin", OrgID: orgID}, nil
			},
		}
		svc := rbac.NewService(permMock, roleMock)

		if err := svc.AssignPermissionToRole(context.Background(), orgB, roleIn, "perm-1"); !errors.Is(err, rbac.ErrRoleNotFound) {
			t.Fatalf("expected ErrRoleNotFound saat lintas organisasi, got %v", err)
		}
	})

	t.Run("role baru mengikuti scope yang diminta", func(t *testing.T) {
		var created *rbac.Role
		roleMock := &mockRoleRepo{
			createFn: func(ctx context.Context, r *rbac.Role) error {
				r.ID = "role-new"
				created = r
				return nil
			},
		}
		svc := rbac.NewService(&mockPermRepo{}, roleMock)

		if _, err := svc.CreateRole(context.Background(), orgA, "admin", "admin org A"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created == nil || created.OrgID != orgA {
			t.Fatalf("role harus dibuat pada scope %q, got %+v", orgA, created)
		}

		if _, err := svc.CreateRole(context.Background(), "", "admin", "admin global"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.OrgID != "" {
			t.Fatalf("role tanpa org harus global, got %q", created.OrgID)
		}
	})

	t.Run("penugasan role mencatat scope organisasinya", func(t *testing.T) {
		var gotOrg, gotUser, gotRole string
		roleMock := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id, orgID string) (*rbac.Role, error) {
				return &rbac.Role{ID: id, Name: "admin", OrgID: orgID}, nil
			},
			assignRoleToUserFn: func(ctx context.Context, orgID, userID, roleID string) error {
				gotOrg, gotUser, gotRole = orgID, userID, roleID
				return nil
			},
		}
		svc := rbac.NewService(&mockPermRepo{}, roleMock)

		if err := svc.AssignRoleToUser(context.Background(), orgA, "user-1", "role-1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotOrg != orgA || gotUser != "user-1" || gotRole != "role-1" {
			t.Fatalf("penugasan harus membawa scope, got org=%q user=%q role=%q", gotOrg, gotUser, gotRole)
		}
	})

	t.Run("permission efektif dihitung per scope", func(t *testing.T) {
		var seenOrg string
		roleMock := &mockRoleRepo{
			getPermsByUserIDFn: func(ctx context.Context, orgID, userID string) ([]rbac.Permission, error) {
				seenOrg = orgID
				if orgID == orgA {
					return []rbac.Permission{{ID: "p-1", Name: "users:read"}}, nil
				}
				return nil, nil
			},
		}
		svc := rbac.NewService(&mockPermRepo{}, roleMock)

		ok, err := svc.HasPermission(context.Background(), orgA, "user-1", "users:read")
		if err != nil || !ok {
			t.Fatalf("permission harus berlaku di organisasinya, got ok=%v err=%v", ok, err)
		}
		if seenOrg != orgA {
			t.Fatalf("scope harus diteruskan ke repository, got %q", seenOrg)
		}

		ok, err = svc.HasPermission(context.Background(), orgB, "user-1", "users:read")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Fatalf("permission dari organisasi lain tidak boleh berlaku")
		}
	})
}
