package variants

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

// variantColumns COALESCE nullable ke zero-value aman untuk Scan.
const variantColumns = "id, organization_id, store_id, product_id, name, COALESCE(sku, ''), stock, is_active, created_at, updated_at"

func (r *postgresRepository) Create(ctx context.Context, v *ProductVariant) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO product_variants (organization_id, store_id, product_id, name, sku, stock, is_active)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, COALESCE($7, TRUE))
		RETURNING id, COALESCE(sku, ''), created_at, updated_at`,
		v.OrganizationID, v.StoreID, v.ProductID, v.Name, v.SKU, v.Stock, v.IsActive,
	).Scan(&v.ID, &v.SKU, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch {
			case pqErr.Code == "23505" && pqErr.Constraint == "unique_variant_sku_per_store":
				return ErrSKUExists
			case pqErr.Code == "23505" || pqErr.Code == "23503" || pqErr.Code == "23514":
				return fmt.Errorf("postgres: create variant: %w", ErrInvalidInput)
			}
		}
		return fmt.Errorf("postgres: create variant: %w", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, orgID, storeID, productID, id string) (*ProductVariant, error) {
	var v ProductVariant
	err := r.db.QueryRowContext(ctx, `
		SELECT `+variantColumns+`
		FROM product_variants
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3 AND id = $4
		LIMIT 1`, orgID, storeID, productID, id).Scan(scanVariantDest(&v)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find variant: %w", err)
	}
	return &v, nil
}

func (r *postgresRepository) FindByProduct(ctx context.Context, orgID, storeID, productID string) ([]ProductVariant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+variantColumns+`
		FROM product_variants
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3
		ORDER BY name ASC`, orgID, storeID, productID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list variants: %w", err)
	}
	defer rows.Close()

	list := []ProductVariant{}
	for rows.Next() {
		var v ProductVariant
		if err := rows.Scan(scanVariantDest(&v)...); err != nil {
			return nil, fmt.Errorf("postgres: scan variant: %w", err)
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

func (r *postgresRepository) Update(ctx context.Context, v *ProductVariant) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE product_variants
		SET name = $5, sku = NULLIF($6, ''), stock = $7, is_active = $8, updated_at = NOW()
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3 AND id = $4`,
		v.OrganizationID, v.StoreID, v.ProductID, v.ID, v.Name, v.SKU, v.Stock, v.IsActive)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch {
			case pqErr.Code == "23505" && pqErr.Constraint == "unique_variant_sku_per_store":
				return ErrSKUExists
			case pqErr.Code == "23505" || pqErr.Code == "23503" || pqErr.Code == "23514":
				return fmt.Errorf("postgres: update variant: %w", ErrInvalidInput)
			}
		}
		return fmt.Errorf("postgres: update variant: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: update variant rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	updated, err := r.FindByID(ctx, v.OrganizationID, v.StoreID, v.ProductID, v.ID)
	if err != nil {
		return err
	}
	*v = *updated
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, orgID, storeID, productID, id string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM product_variants
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3 AND id = $4`,
		orgID, storeID, productID, id)
	if err != nil {
		return fmt.Errorf("postgres: delete variant: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: delete variant rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) ExistsBySKU(ctx context.Context, storeID, sku, excludeID string) (bool, error) {
	if sku == "" {
		return false, nil
	}
	q := `SELECT 1 FROM product_variants WHERE store_id = $1 AND sku = $2`
	args := []any{storeID, sku}
	if excludeID != "" {
		q += ` AND id <> $3`
		args = append(args, excludeID)
	}
	q += ` LIMIT 1`
	var one int
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&one)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("postgres: exists variant sku: %w", err)
	}
	return true, nil
}

// scanVariantDest memetakan kolom product_variants ke struct.
func scanVariantDest(v *ProductVariant) []any {
	return []any{
		&v.ID, &v.OrganizationID, &v.StoreID, &v.ProductID, &v.Name,
		&v.SKU, &v.Stock, &v.IsActive, &v.CreatedAt, &v.UpdatedAt,
	}
}
