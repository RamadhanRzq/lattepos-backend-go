package rbac

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

type permissionRepository struct {
	db *sql.DB
}

var _ PermissionRepository = (*permissionRepository)(nil)

func NewPermissionRepository(db *sql.DB) PermissionRepository {
	return &permissionRepository{db: db}
}

const permissionColumns = "id, name, description, created_at, updated_at"

func (r *permissionRepository) Create(ctx context.Context, p *Permission) error {
	query := `
		INSERT INTO permissions (name, description)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query, p.Name, p.Description).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrPermissionNameTaken
		}
		return fmt.Errorf("postgres: create permission: %w", err)
	}
	return nil
}

func (r *permissionRepository) FindByID(ctx context.Context, id string) (*Permission, error) {
	query := `SELECT ` + permissionColumns + ` FROM permissions WHERE id = $1`
	return r.queryOne(ctx, query, id)
}

func (r *permissionRepository) FindByName(ctx context.Context, name string) (*Permission, error) {
	query := `SELECT ` + permissionColumns + ` FROM permissions WHERE lower(name) = lower($1) LIMIT 1`
	return r.queryOne(ctx, query, name)
}

func (r *permissionRepository) List(ctx context.Context) ([]*Permission, error) {
	query := `SELECT ` + permissionColumns + ` FROM permissions ORDER BY name ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: list permissions: %w", err)
	}
	defer rows.Close()

	var list []*Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan permission: %w", err)
		}
		list = append(list, &p)
	}
	return list, rows.Err()
}

func (r *permissionRepository) queryOne(ctx context.Context, query string, args ...any) (*Permission, error) {
	var p Permission
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPermissionNotFound
		}
		return nil, fmt.Errorf("postgres: query permission: %w", err)
	}
	return &p, nil
}

type roleRepository struct {
	db *sql.DB
}

var _ RoleRepository = (*roleRepository)(nil)

func NewRoleRepository(db *sql.DB) RoleRepository {
	return &roleRepository{db: db}
}

// org_id ikut di-select supaya caller bisa membedakan role global dan role organisasi.
const roleColumns = "id, name, description, org_id, created_at, updated_at"

// scopeClause mencocokkan role pada satu scope saja: orgID kosong berarti role global.
const roleScopeClause = `org_id IS NOT DISTINCT FROM NULLIF($1, '')::uuid`

func (r *roleRepository) Create(ctx context.Context, role *Role) error {
	query := `
		INSERT INTO roles (name, description, org_id)
		VALUES ($1, $2, NULLIF($3, '')::uuid)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query, role.Name, role.Description, role.OrgID).
		Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrRoleNameTaken
		}
		return fmt.Errorf("postgres: create role: %w", err)
	}
	return nil
}

// FindByID mencari role di dalam satu scope. Role milik organisasi lain tidak
// akan pernah ditemukan, sehingga tidak bisa dibaca maupun diubah lintas organisasi.
func (r *roleRepository) FindByID(ctx context.Context, id, orgID string) (*Role, error) {
	query := `SELECT ` + roleColumns + ` FROM roles WHERE id = $2 AND ` + roleScopeClause
	return r.queryOne(ctx, query, orgID, id)
}

// FindByName mencari role berdasarkan nama di dalam satu scope, dengan
// preferensi role scope itu sendiri sebelum jatuh ke role global.
func (r *roleRepository) FindByName(ctx context.Context, name, orgID string) (*Role, error) {
	query := `
		SELECT ` + roleColumns + `
		FROM roles
		WHERE lower(name) = lower($2)
		  AND (org_id IS NOT DISTINCT FROM NULLIF($1, '')::uuid OR org_id IS NULL)
		ORDER BY (org_id IS NULL) ASC
		LIMIT 1`
	return r.queryOne(ctx, query, orgID, name)
}

func (r *roleRepository) List(ctx context.Context, orgID string) ([]*Role, error) {
	query := `SELECT ` + roleColumns + ` FROM roles WHERE ` + roleScopeClause + ` ORDER BY name ASC`
	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list roles: %w", err)
	}
	defer rows.Close()

	var list []*Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, role)
	}
	return list, rows.Err()
}

func (r *roleRepository) AssignPermission(ctx context.Context, roleID, permissionID string) error {
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

func (r *roleRepository) GetPermissionsByRoleID(ctx context.Context, roleID string) ([]Permission, error) {
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

	var list []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan role permission: %w", err)
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// AssignRoleToUser mencatat penugasan beserta scope organisasinya, sehingga
// pemberian role di satu organisasi tidak berlaku di organisasi lain.
func (r *roleRepository) AssignRoleToUser(ctx context.Context, orgID, userID, roleID string) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, org_id)
		VALUES ($1, $2, NULLIF($3, '')::uuid)
		ON CONFLICT (user_id, role_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, strings.TrimSpace(userID), strings.TrimSpace(roleID), strings.TrimSpace(orgID))
	if err != nil {
		return fmt.Errorf("postgres: assign role to user: %w", err)
	}
	return nil
}

func (r *roleRepository) GetRolesByUserID(ctx context.Context, userID string) ([]Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.org_id, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name ASC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: get roles by user: %w", err)
	}
	defer rows.Close()

	var list []Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *role)
	}
	return list, rows.Err()
}

// GetPermissionsByUserID menghitung permission efektif pada satu scope:
// penugasan global (org_id NULL) berlaku di mana saja, penugasan ber-scope
// hanya berlaku di organisasi tersebut.
func (r *roleRepository) GetPermissionsByUserID(ctx context.Context, orgID, userID string) ([]Permission, error) {
	query := `
		SELECT DISTINCT p.id, p.name, p.description, p.created_at, p.updated_at
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = $2
		  AND (ur.org_id IS NULL OR ur.org_id IS NOT DISTINCT FROM NULLIF($1, '')::uuid)
		ORDER BY p.name ASC`

	rows, err := r.db.QueryContext(ctx, query, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: get permissions by user: %w", err)
	}
	defer rows.Close()

	var list []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan user permission: %w", err)
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *roleRepository) queryOne(ctx context.Context, query string, args ...any) (*Role, error) {
	row := r.db.QueryRowContext(ctx, query, args...)
	role, err := scanRole(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("postgres: query role: %w", err)
	}
	return role, nil
}

// rowScanner bekerja untuk *sql.Row maupun *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanRole(row rowScanner) (*Role, error) {
	var role Role
	var orgID sql.NullString

	err := row.Scan(&role.ID, &role.Name, &role.Description, &orgID, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, err
	}

	role.OrgID = orgID.String
	return &role, nil
}
