package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/ramadhanrzq/backend-go/internal/domain/user"
)

// UserRepository mengimplementasikan user.UserRepository di atas PostgreSQL.
type UserRepository struct {
	db *sql.DB
}

// Pastikan kontrak interface domain terpenuhi saat compile.
var _ user.UserRepository = (*UserRepository)(nil)

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

const userColumns = "id, username, name, email, role, password_hash, created_at, updated_at"

// FindByUsername mencari user berdasarkan username (case-insensitive via citext-style lower()).
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*user.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE lower(username) = lower($1) LIMIT 1`

	return r.queryOne(ctx, query, username)
}

// FindByID mencari user berdasarkan ID.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1 LIMIT 1`

	return r.queryOne(ctx, query, id)
}

// List mengembalikan semua user diurutkan berdasarkan ID.
func (r *UserRepository) List(ctx context.Context) ([]*user.User, error) {
	query := `SELECT ` + userColumns + ` FROM users ORDER BY id`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: list users: %w", err)
	}
	defer rows.Close()

	users := []*user.User{}

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
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO users (username, name, email, role, password_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		u.Username, u.Name, u.Email, u.Role, u.PasswordHash,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return mapCreateError(err, u)
	}

	return nil
}

func (r *UserRepository) queryOne(ctx context.Context, query string, args ...any) (*user.User, error) {
	row := r.db.QueryRowContext(ctx, query, args...)

	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: query user: %w", err)
	}

	return u, nil
}

// rowScanner bekerja untuk *sql.Row maupun *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*user.User, error) {
	var u user.User

	err := row.Scan(&u.ID, &u.Username, &u.Name, &u.Email, &u.Role, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// mapCreateError menerjemahkan error dari database ke error domain.
func mapCreateError(err error, u *user.User) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch {
		case strings.Contains(pqErr.Constraint, "username"):
			return user.ErrUsernameTaken
		case strings.Contains(pqErr.Constraint, "email"):
			return user.ErrEmailTaken
		default:
			return user.ErrUsernameTaken
		}
	}

	return fmt.Errorf("postgres: create user: %w", err)
}
