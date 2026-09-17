package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// postgresRepository mengimplementasikan Repository di atas PostgreSQL.
type postgresRepository struct {
	db *sql.DB
}

// Pastikan kontrak interface terpenuhi saat compile.
var _ Repository = (*postgresRepository)(nil)

func NewRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

const userColumns = "id, username, name, email, role, password_hash, created_at, updated_at"

// userColumnsAliased dipakai pada query yang JOIN ke tabel lain (organization_members)
// supaya kolom id/created_at tidak ambigu.
const userColumnsAliased = "u.id, u.username, u.name, u.email, u.role, u.password_hash, u.created_at, u.updated_at"

// FindByUsername mencari user berdasarkan username (case-insensitive via lower()).
func (r *postgresRepository) FindByUsername(ctx context.Context, username string) (*User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE lower(username) = lower($1) LIMIT 1`

	return r.queryOne(ctx, query, username)
}

// FindByID mencari user berdasarkan ID.
func (r *postgresRepository) FindByID(ctx context.Context, id string) (*User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1 LIMIT 1`

	return r.queryOne(ctx, query, id)
}

// FindByIDInOrg mencari user yang merupakan anggota organisasi tersebut.
func (r *postgresRepository) FindByIDInOrg(ctx context.Context, orgID, id string) (*User, error) {
	query := `SELECT ` + userColumnsAliased + `
		FROM users u
		JOIN organization_members om ON om.user_id = u.id
		WHERE om.org_id = $1 AND u.id = $2
		LIMIT 1`

	return r.queryOne(ctx, query, orgID, id)
}

// ListByOrg mengembalikan anggota organisasi saja, sehingga user dari
// organisasi lain tidak pernah terlihat.
func (r *postgresRepository) ListByOrg(ctx context.Context, orgID string) ([]*User, error) {
	query := `SELECT ` + userColumnsAliased + `
		FROM users u
		JOIN organization_members om ON om.user_id = u.id
		WHERE om.org_id = $1
		ORDER BY u.name ASC`

	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list org users: %w", err)
	}
	defer rows.Close()

	list := []*User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterasi hasil list org users: %w", err)
	}

	return list, nil
}

// List mengembalikan semua user diurutkan berdasarkan ID.
func (r *postgresRepository) List(ctx context.Context) ([]*User, error) {
	query := `SELECT ` + userColumns + ` FROM users ORDER BY id`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: list users: %w", err)
	}
	defer rows.Close()

	users := []*User{}

	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterasi hasil list users: %w", err)
	}

	return users, nil
}

// Create menyimpan user baru.
func (r *postgresRepository) Create(ctx context.Context, u *User) error {
	query := `
		INSERT INTO users (username, name, email, role, password_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		u.Username, u.Name, u.Email, u.Role, u.PasswordHash,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return mapCreateError(err)
	}

	return nil
}

func (r *postgresRepository) queryOne(ctx context.Context, query string, args ...any) (*User, error) {
	row := r.db.QueryRowContext(ctx, query, args...)

	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: query user: %w", err)
	}

	return u, nil
}

// rowScanner bekerja untuk *sql.Row maupun *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*User, error) {
	var u User

	err := row.Scan(&u.ID, &u.Username, &u.Name, &u.Email, &u.Role, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// mapCreateError menerjemahkan error dari database ke error domain.
func mapCreateError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch {
		case strings.Contains(pqErr.Constraint, "username"):
			return ErrUsernameTaken
		case strings.Contains(pqErr.Constraint, "email"):
			return ErrEmailTaken
		default:
			return ErrUsernameTaken
		}
	}

	return fmt.Errorf("postgres: create user: %w", err)
}
