package products

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

type postgresRepository struct {
	db *sql.DB
}

var _ Repository = (*postgresRepository)(nil)

func NewRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

// productColumns COALESCE nullable ke zero-value aman untuk Scan.
const productColumns = "id, store_id, organization_id, name, sku, COALESCE(description, ''), price, stock, COALESCE(unit, 'pcs'), category_id, COALESCE(image_url, ''), is_active, COALESCE(created_by::text, ''), created_at, updated_at, deleted_at"

func (r *postgresRepository) Create(ctx context.Context, p *Product) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO products (store_id, organization_id, name, sku, description, price, stock, unit, category_id, image_url, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, '')::uuid, $10, $11, NULLIF($12, '')::uuid)
		RETURNING id, COALESCE(unit, 'pcs'), is_active, created_at, updated_at`,
		p.StoreID, p.OrganizationID, p.Name, p.SKU, p.Description, p.Price, p.Stock, p.Unit,
		nullableStr(p.CategoryID), p.ImageURL, p.IsActive, p.CreatedBy,
	).Scan(&p.ID, &p.Unit, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" &&
			pqErr.Constraint == "unique_product_sku_per_store" {
			return ErrSKUDuplicate
		}
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return fmt.Errorf("postgres: create product: %w", ErrInvalidInput)
		}
		return fmt.Errorf("postgres: create product: %w", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, orgID, storeID, id string) (*Product, error) {
	var p Product
	err := r.db.QueryRowContext(ctx, `
		SELECT `+productColumns+`
		FROM products
		WHERE organization_id = $1 AND store_id = $2 AND id = $3 AND deleted_at IS NULL
		LIMIT 1`, orgID, storeID, id).Scan(scanProductDest(&p)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find product: %w", err)
	}
	return &p, nil
}

func (r *postgresRepository) FindByStore(ctx context.Context, orgID, storeID string, filter Filter) ([]Product, int, error) {
	where, args := buildFilter(orgID, storeID, filter)

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM products `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count products: %w", err)
	}

	page, limit := filter.Page, filter.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	args = append(args, limit, (page-1)*limit)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+productColumns+`
		FROM products `+where+`
		ORDER BY name ASC
		LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list products: %w", err)
	}
	defer rows.Close()

	list := []Product{}
	for rows.Next() {
		var p Product
		if err := rows.Scan(scanProductDest(&p)...); err != nil {
			return nil, 0, fmt.Errorf("postgres: scan product: %w", err)
		}
		list = append(list, p)
	}
	return list, total, rows.Err()
}

func (r *postgresRepository) Update(ctx context.Context, p *Product) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE products
		SET name = $4, sku = $5, description = $6, price = $7, stock = $8,
			unit = $9, category_id = NULLIF($10, '')::uuid,
			image_url = $11, is_active = $12, updated_at = NOW()
		WHERE organization_id = $1 AND store_id = $2 AND id = $3 AND deleted_at IS NULL`,
		p.OrganizationID, p.StoreID, p.ID, p.Name, p.SKU, p.Description, p.Price, p.Stock,
		p.Unit, nullableStr(p.CategoryID), p.ImageURL, p.IsActive)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" &&
			pqErr.Constraint == "unique_product_sku_per_store" {
			return ErrSKUDuplicate
		}
		return fmt.Errorf("postgres: update product: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: update product rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	updated, err := r.FindByID(ctx, p.OrganizationID, p.StoreID, p.ID)
	if err != nil {
		return err
	}
	*p = *updated
	return nil
}

func (r *postgresRepository) SoftDelete(ctx context.Context, orgID, storeID, id string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE products
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE organization_id = $1 AND store_id = $2 AND id = $3 AND deleted_at IS NULL`,
		orgID, storeID, id)
	if err != nil {
		return fmt.Errorf("postgres: delete product: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: delete product rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) ExistsBySKU(ctx context.Context, sku, storeID string, excludeID *string) (bool, error) {
	q := `SELECT 1 FROM products WHERE store_id = $1 AND sku = $2 AND deleted_at IS NULL`
	args := []any{storeID, sku}
	if excludeID != nil && *excludeID != "" {
		q += ` AND id <> $3`
		args = append(args, *excludeID)
	}
	q += ` LIMIT 1`
	var one int
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&one)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("postgres: exists product sku: %w", err)
	}
	return true, nil
}

// buildFilter menyusun WHERE ter-scope org+store; search ILIKE name/sku.
func buildFilter(orgID, storeID string, filter Filter) (string, []any) {
	where := `WHERE organization_id = $1 AND store_id = $2 AND deleted_at IS NULL`
	args := []any{orgID, storeID}
	if s := strings.TrimSpace(filter.Search); s != "" {
		args = append(args, "%"+s+"%")
		where += ` AND (name ILIKE $` + itoa(len(args)) + ` OR sku ILIKE $` + itoa(len(args)) + `)`
	}
	if filter.CategoryID != nil && *filter.CategoryID != "" {
		args = append(args, *filter.CategoryID)
		where += ` AND category_id = $` + itoa(len(args)) + `::uuid`
	}
	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		where += ` AND is_active = $` + itoa(len(args))
	}
	return where, args
}

// scanProductDest memetakan kolom products ke struct.
func scanProductDest(p *Product) []any {
	return []any{
		&p.ID, &p.StoreID, &p.OrganizationID, &p.Name, &p.SKU,
		&p.Description, &p.Price, &p.Stock, &p.Unit, &p.CategoryID,
		&p.ImageURL, &p.IsActive, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
	}
}

func nullableStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func itoa(n int) string {
	return fmt.Sprint(n)
}
