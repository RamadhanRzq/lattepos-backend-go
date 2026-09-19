package categories

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

// categoryColumns COALESCE nullable ke zero-value aman untuk Scan.
const categoryColumns = "id, store_id, organization_id, name, slug, COALESCE(description, ''), parent_id, created_at, updated_at"

func (r *postgresRepository) Create(ctx context.Context, c *Category) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO categories (store_id, organization_id, name, slug, description, parent_id)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, '')::uuid)
		RETURNING id, COALESCE(description, ''), created_at, updated_at`,
		c.StoreID, c.OrganizationID, c.Name, c.Slug, c.Description, nullableStr(c.ParentID),
	).Scan(&c.ID, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" &&
			pqErr.Constraint == "unique_category_slug_per_store" {
			return ErrSlugExists
		}
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return fmt.Errorf("postgres: create category: %w", ErrInvalidInput)
		}
		return fmt.Errorf("postgres: create category: %w", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, orgID, storeID, id string) (*Category, error) {
	var c Category
	err := r.db.QueryRowContext(ctx, `
		SELECT `+categoryColumns+`
		FROM categories
		WHERE organization_id = $1 AND store_id = $2 AND id = $3
		LIMIT 1`, orgID, storeID, id).Scan(scanCategoryDest(&c)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find category: %w", err)
	}
	return &c, nil
}

func (r *postgresRepository) FindBySlug(ctx context.Context, orgID, storeID, slug string) (*Category, error) {
	var c Category
	err := r.db.QueryRowContext(ctx, `
		SELECT `+categoryColumns+`
		FROM categories
		WHERE organization_id = $1 AND store_id = $2 AND slug = $3
		LIMIT 1`, orgID, storeID, slug).Scan(scanCategoryDest(&c)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find category by slug: %w", err)
	}
	return &c, nil
}

func (r *postgresRepository) FindByStore(ctx context.Context, orgID, storeID string) ([]Category, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+categoryColumns+`
		FROM categories
		WHERE organization_id = $1 AND store_id = $2
		ORDER BY name ASC`, orgID, storeID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list categories: %w", err)
	}
	defer rows.Close()

	list := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(scanCategoryDest(&c)...); err != nil {
			return nil, fmt.Errorf("postgres: scan category: %w", err)
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *postgresRepository) Update(ctx context.Context, c *Category) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE categories
		SET name = $4, slug = $5, description = NULLIF($6, ''), parent_id = NULLIF($7, '')::uuid, updated_at = NOW()
		WHERE organization_id = $1 AND store_id = $2 AND id = $3`,
		c.OrganizationID, c.StoreID, c.ID, c.Name, c.Slug, c.Description, nullableStr(c.ParentID))
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" &&
			pqErr.Constraint == "unique_category_slug_per_store" {
			return ErrSlugExists
		}
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return fmt.Errorf("postgres: update category: %w", ErrInvalidInput)
		}
		return fmt.Errorf("postgres: update category: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: update category rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	updated, err := r.FindByID(ctx, c.OrganizationID, c.StoreID, c.ID)
	if err != nil {
		return err
	}
	*c = *updated
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, orgID, storeID, id string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM categories
		WHERE organization_id = $1 AND store_id = $2 AND id = $3`,
		orgID, storeID, id)
	if err != nil {
		return fmt.Errorf("postgres: delete category: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: delete category: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) ExistsBySlug(ctx context.Context, slug, storeID string, excludeID *string) (bool, error) {
	var exists bool
	var err error
	if excludeID == nil {
		err = r.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM categories WHERE store_id = $1 AND slug = $2)`,
			storeID, slug).Scan(&exists)
	} else {
		err = r.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM categories WHERE store_id = $1 AND slug = $2 AND id <> $3)`,
			storeID, slug, *excludeID).Scan(&exists)
	}
	if err != nil {
		return false, fmt.Errorf("postgres: exists category slug: %w", err)
	}
	return exists, nil
}

func (r *postgresRepository) CountProducts(ctx context.Context, storeID, categoryID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM products WHERE store_id = $1 AND category_id = $2 AND deleted_at IS NULL`,
		storeID, categoryID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("postgres: count category products: %w", err)
	}
	return n, nil
}

// scanCategoryDest memetakan kolom categories ke struct.
func scanCategoryDest(c *Category) []any {
	return []any{
		&c.ID, &c.StoreID, &c.OrganizationID, &c.Name, &c.Slug, &c.Description,
		&c.ParentID, &c.CreatedAt, &c.UpdatedAt,
	}
}

func nullableStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
