package tables

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

type postgresRepository struct {
	db *sql.DB
}

var _ Repository = (*postgresRepository)(nil)

func NewRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

// tableColumns COALESCE nullable ke zero-value aman untuk Scan.
const tableColumns = "id, store_id, organization_id, name, COALESCE(area, ''), capacity, status, is_active, created_at, updated_at"

func (r *postgresRepository) Create(ctx context.Context, t *Table) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO tables (store_id, organization_id, name, area, capacity, status, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`,
		t.StoreID, t.OrganizationID, t.Name, t.Area, t.Capacity, t.Status, t.IsActive,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return mapWriteError(err, "create table")
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, orgID, storeID, id string) (*Table, error) {
	var t Table
	err := r.db.QueryRowContext(ctx, `
		SELECT `+tableColumns+`
		FROM tables
		WHERE organization_id = $1 AND store_id = $2 AND id = $3
		LIMIT 1`, orgID, storeID, id).Scan(scanTableDest(&t)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find table: %w", err)
	}
	return &t, nil
}

func (r *postgresRepository) FindByStore(ctx context.Context, orgID, storeID string, filter Filter) ([]Table, int, error) {
	where := `WHERE organization_id = $1 AND store_id = $2`
	args := []any{orgID, storeID}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if filter.Area != "" {
		args = append(args, filter.Area)
		where += fmt.Sprintf(" AND area = $%d", len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tables `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count tables: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT `+tableColumns+`
		FROM tables `+where+`
		ORDER BY area ASC, name ASC`, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list tables: %w", err)
	}
	defer rows.Close()

	list := []Table{}
	for rows.Next() {
		var t Table
		if err := rows.Scan(scanTableDest(&t)...); err != nil {
			return nil, 0, fmt.Errorf("postgres: scan table: %w", err)
		}
		list = append(list, t)
	}
	return list, total, rows.Err()
}

func (r *postgresRepository) Update(ctx context.Context, t *Table) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE tables
		SET name = $4, area = $5, capacity = $6, is_active = $7, updated_at = NOW()
		WHERE organization_id = $1 AND store_id = $2 AND id = $3`,
		t.OrganizationID, t.StoreID, t.ID, t.Name, t.Area, t.Capacity, t.IsActive)
	if err != nil {
		return mapWriteError(err, "update table")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: update table rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	updated, err := r.FindByID(ctx, t.OrganizationID, t.StoreID, t.ID)
	if err != nil {
		return err
	}
	*t = *updated
	return nil
}

func (r *postgresRepository) UpdateStatus(ctx context.Context, orgID, storeID, id, status string) (*Table, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE tables
		SET status = $4, updated_at = NOW()
		WHERE organization_id = $1 AND store_id = $2 AND id = $3`,
		orgID, storeID, id, status)
	if err != nil {
		return nil, mapWriteError(err, "update table status")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("postgres: update table status rows: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.FindByID(ctx, orgID, storeID, id)
}

func (r *postgresRepository) Delete(ctx context.Context, orgID, storeID, id string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM tables
		WHERE organization_id = $1 AND store_id = $2 AND id = $3`,
		orgID, storeID, id)
	if err != nil {
		return fmt.Errorf("postgres: delete table: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: delete table rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) ExistsByName(ctx context.Context, storeID, name string, excludeID *string) (bool, error) {
	var exists bool
	var err error
	if excludeID == nil {
		err = r.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM tables WHERE store_id = $1 AND name = $2)`,
			storeID, name).Scan(&exists)
	} else {
		err = r.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM tables WHERE store_id = $1 AND name = $2 AND id <> $3)`,
			storeID, name, *excludeID).Scan(&exists)
	}
	if err != nil {
		return false, fmt.Errorf("postgres: exists table name: %w", err)
	}
	return exists, nil
}

// mapWriteError menerjemahkan constraint violation ke error domain.
func mapWriteError(err error, op string) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505":
			if pqErr.Constraint == "unique_table_name_per_store" {
				return ErrNameExists
			}
		case "23503", "23514":
			return fmt.Errorf("postgres: %s: %w", op, ErrInvalidInput)
		}
	}
	return fmt.Errorf("postgres: %s: %w", op, err)
}

// scanTableDest memetakan kolom tables ke struct.
func scanTableDest(t *Table) []any {
	return []any{
		&t.ID, &t.StoreID, &t.OrganizationID, &t.Name, &t.Area, &t.Capacity,
		&t.Status, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	}
}
