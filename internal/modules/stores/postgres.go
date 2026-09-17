package stores

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/ramadhanrzq/backend-go/internal/modules/users"
)

type postgresRepository struct {
	db *sql.DB
}

var _ Repository = (*postgresRepository)(nil)

func NewRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

// storeColumns COALESCE nullable address/phone ke string kosong agar Scan ke string aman.
const storeColumns = "id, organization_id, name, code, COALESCE(address, ''), COALESCE(phone, ''), is_active, created_at, updated_at"

const storeColumnsAliased = "s.id, s.organization_id, s.name, s.code, COALESCE(s.address, ''), COALESCE(s.phone, ''), s.is_active, s.created_at, s.updated_at"

func (r *postgresRepository) Create(ctx context.Context, s *Store) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO stores (organization_id, name, code, address, phone)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''))
		RETURNING id, is_active, created_at, updated_at`,
		s.OrganizationID, s.Name, s.Code, s.Address, s.Phone,
	).Scan(&s.ID, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" &&
			pqErr.Constraint == "unique_store_code_per_org" {
			return ErrCodeTaken
		}
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return fmt.Errorf("postgres: create store: %w", ErrInvalidInput)
		}
		return fmt.Errorf("postgres: create store: %w", err)
	}
	return nil
}

func (r *postgresRepository) FindByIDInOrg(ctx context.Context, orgID, id string) (*Store, error) {
	var s Store
	err := r.db.QueryRowContext(ctx, `
		SELECT `+storeColumns+`
		FROM stores
		WHERE organization_id = $1 AND id = $2
		LIMIT 1`, orgID, id).Scan(scanStoreDest(&s)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find store: %w", err)
	}
	return &s, nil
}

func (r *postgresRepository) ListByOrg(ctx context.Context, orgID string) ([]Store, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+storeColumns+`
		FROM stores
		WHERE organization_id = $1
		ORDER BY name ASC`, orgID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list stores: %w", err)
	}
	defer rows.Close()

	list := []Store{}
	for rows.Next() {
		var s Store
		if err := rows.Scan(scanStoreDest(&s)...); err != nil {
			return nil, fmt.Errorf("postgres: scan store: %w", err)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *postgresRepository) Update(ctx context.Context, s *Store) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE stores
		SET name = $3, code = $4,
			address = NULLIF($5, ''), phone = NULLIF($6, ''),
			updated_at = NOW()
		WHERE organization_id = $1 AND id = $2`,
		s.OrganizationID, s.ID, s.Name, s.Code, s.Address, s.Phone)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" &&
			pqErr.Constraint == "unique_store_code_per_org" {
			return ErrCodeTaken
		}
		return fmt.Errorf("postgres: update store: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: update store rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	updated, err := r.FindByIDInOrg(ctx, s.OrganizationID, s.ID)
	if err != nil {
		return err
	}
	*s = *updated
	return nil
}

func (r *postgresRepository) SetActive(ctx context.Context, orgID, id string, active bool) (*Store, error) {
	var s Store
	err := r.db.QueryRowContext(ctx, `
		UPDATE stores
		SET is_active = $3, updated_at = NOW()
		WHERE organization_id = $1 AND id = $2
		RETURNING `+storeColumns, orgID, id, active).Scan(scanStoreDest(&s)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: set store active: %w", err)
	}
	return &s, nil
}

func (r *postgresRepository) AssignUser(ctx context.Context, userID, storeID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_stores (user_id, store_id)
		VALUES ($1, $2)`, userID, storeID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" &&
			pqErr.Constraint == "unique_user_store" {
			return ErrAlreadyAssigned
		}
		return fmt.Errorf("postgres: assign user store: %w", err)
	}
	return nil
}

func (r *postgresRepository) IsAssigned(ctx context.Context, userID, storeID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_stores
			WHERE user_id = $1 AND store_id = $2
		)`, userID, storeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("postgres: is assigned check: %w", err)
	}
	return exists, nil
}

func (r *postgresRepository) RemoveUser(ctx context.Context, userID, storeID string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM user_stores
		WHERE user_id = $1 AND store_id = $2`, userID, storeID)
	if err != nil {
		return fmt.Errorf("postgres: remove user store: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: remove user store rows: %w", err)
	}
	if n == 0 {
		return ErrNotAssigned
	}
	return nil
}

// ListUserStores hanya mengembalikan store dalam orgID: JOIN ke stores
// sekaligus enforcement tenant boundary, bukan sekadar filter store_id.
func (r *postgresRepository) ListUserStores(ctx context.Context, orgID, userID string) ([]Store, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+storeColumnsAliased+`
		FROM stores s
		JOIN user_stores us ON us.store_id = s.id
		WHERE s.organization_id = $1 AND us.user_id = $2
		ORDER BY s.name ASC`, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list user stores: %w", err)
	}
	defer rows.Close()

	list := []Store{}
	for rows.Next() {
		var s Store
		if err := rows.Scan(scanStoreDest(&s)...); err != nil {
			return nil, fmt.Errorf("postgres: scan user store: %w", err)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// ListStoreUsers hanya mengembalikan user yang assignment-nya milik orgID:
// same-org dienforce di JOIN stores, menolak user lintas organisasi.
func (r *postgresRepository) ListStoreUsers(ctx context.Context, orgID, storeID string) ([]users.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.username, u.name, u.email, u.role, u.created_at, u.updated_at
		FROM users u
		JOIN user_stores us ON us.user_id = u.id
		JOIN stores s ON s.id = us.store_id AND s.organization_id = $1
		WHERE us.store_id = $2
		ORDER BY u.name ASC`, orgID, storeID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list store users: %w", err)
	}
	defer rows.Close()

	members := []users.User{}
	for rows.Next() {
		var u users.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Name, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan store user: %w", err)
		}
		members = append(members, u)
	}
	return members, rows.Err()
}

// scanStoreDest memetakan kolom stores ke struct.
func scanStoreDest(s *Store) []any {
	return []any{
		&s.ID, &s.OrganizationID, &s.Name, &s.Code,
		&s.Address, &s.Phone, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	}
}
