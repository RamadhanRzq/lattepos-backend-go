package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/ramadhanrzq/backend-go/internal/domain/rbac"
)

type PermissionRepository struct {
	db *sql.DB
}

var _ rbac.PermissionRepository = (*PermissionRepository)(nil)

func NewPermissionRepository(db *sql.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

const permissionColumns = "id, name, description, created_at, updated_at"

func (r *PermissionRepository) Create(ctx context.Context, p *rbac.Permission) error {
	query := `
		INSERT INTO permissions (name, description)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query, p.Name, p.Description).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return rbac.ErrPermissionNameTaken
		}
		return fmt.Errorf("postgres: create permission: %w", err)
	}
	return nil
}

func (r *PermissionRepository) FindByID(ctx context.Context, id string) (*rbac.Permission, error) {
	query := `SELECT ` + permissionColumns + ` FROM permissions WHERE id = $1`
	return r.queryOne(ctx, query, id)
}

func (r *PermissionRepository) FindByName(ctx context.Context, name string) (*rbac.Permission, error) {
	query := `SELECT ` + permissionColumns + ` FROM permissions WHERE lower(name) = lower($1) LIMIT 1`
	return r.queryOne(ctx, query, name)
}

func (r *PermissionRepository) List(ctx context.Context) ([]*rbac.Permission, error) {
	query := `SELECT ` + permissionColumns + ` FROM permissions ORDER BY name ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: list permissions: %w", err)
	}
	defer rows.Close()

	var list []*rbac.Permission
	for rows.Next() {
		var p rbac.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan permission: %w", err)
		}
		list = append(list, &p)
	}
	return list, rows.Err()
}

func (r *PermissionRepository) queryOne(ctx context.Context, query string, args ...any) (*rbac.Permission, error) {
	var p rbac.Permission
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rbac.ErrPermissionNotFound
		}
		return nil, fmt.Errorf("postgres: query permission: %w", err)
	}
	return &p, nil
}

type RoleRepository struct {
	db *sql.DB
}

var _ rbac.RoleRepository = (*RoleRepository)(nil)

func NewRoleRepository(db *sql.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

const roleColumns = "id, name, description, created_at, updated_at"

func (r *RoleRepository) Create(ctx context.Context, role *rbac.Role) error {
	query := `
		INSERT INTO roles (name, description)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query, role.Name, role.Description).
		Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return rbac.ErrRoleNameTaken
		}
		return fmt.Errorf("postgres: create role: %w", err)
	}
	return nil
}

func (r *RoleRepository) FindByID(ctx context.Context, id string) (*rbac.Role, error) {
	query := `SELECT ` + roleColumns + ` FROM roles WHERE id = $1`
	return r.queryOne(ctx, query, id)
}

func (r *RoleRepository) FindByName(ctx context.Context, name string) (*rbac.Role, error) {
	query := `SELECT ` + roleColumns + ` FROM roles WHERE lower(name) = lower($1) LIMIT 1`
	return r.queryOne(ctx, query, name)
}

func (r *RoleRepository) List(ctx context.Context) ([]*rbac.Role, error) {
	query := `SELECT ` + roleColumns + ` FROM roles ORDER BY name ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: list roles: %w", err)
	}
	defer rows.Close()

	var list []*rbac.Role
	for rows.Next() {
		var role rbac.Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan role: %w", err)
		}
		list = append(list, &role)
	}
	return list, rows.Err()
}

func (r *RoleRepository) AssignPermission(ctx context.Context, roleID, permissionID string) error {
	query := `
		INSERT INTO role_permissions (role_id, permission_id)
		VALUES ($1, $2)
		ON CONFLICT (role_id, permission_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, strings.TrimSpace(roleID), strings.TrimSpace(permissionID))
	if err != nil {
		return fmt.Errorf("postgres: assign permission to role: %w", err)
	}
	return nil
}

func (r *RoleRepository) GetPermissionsByRoleID(ctx context.Context, roleID string) ([]rbac.Permission, error) {
	query := `
		SELECT p.id, p.name, p.description, p.created_at, p.updated_at
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1
		ORDER BY p.name ASC`

	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("postgres: get permissions by role: %w", err)
	}
	defer rows.Close()

	var list []rbac.Permission
	for rows.Next() {
		var p rbac.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan role permission: %w", err)
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *RoleRepository) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	query := `
		INSERT INTO user_roles (user_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, role_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, strings.TrimSpace(userID), strings.TrimSpace(roleID))
	if err != nil {
		return fmt.Errorf("postgres: assign role to user: %w", err)
	}
	return nil
}

func (r *RoleRepository) GetRolesByUserID(ctx context.Context, userID string) ([]rbac.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name ASC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: get roles by user: %w", err)
	}
	defer rows.Close()

	var list []rbac.Role
	for rows.Next() {
		var role rbac.Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan user role: %w", err)
		}
		list = append(list, role)
	}
	return list, rows.Err()
}

func (r *RoleRepository) GetPermissionsByUserID(ctx context.Context, userID string) ([]rbac.Permission, error) {
	query := `
		SELECT DISTINCT p.id, p.name, p.description, p.created_at, p.updated_at
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name ASC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: get permissions by user: %w", err)
	}
	defer rows.Close()

	var list []rbac.Permission
	for rows.Next() {
		var p rbac.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan user permission: %w", err)
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *RoleRepository) queryOne(ctx context.Context, query string, args ...any) (*rbac.Role, error) {
	var role rbac.Role
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rbac.ErrRoleNotFound
		}
		return nil, fmt.Errorf("postgres: query role: %w", err)
	}
	return &role, nil
}
