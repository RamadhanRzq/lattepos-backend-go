package prices

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// postgresRepository menyimpan price di tabel product_prices.
type postgresRepository struct {
	db *sql.DB
}

var _ Repository = (*postgresRepository)(nil)

func NewRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

// priceColumns memetakan kolom product_prices; price dibaca via numeric helper.
const priceColumns = "id, organization_id, store_id, product_id, variant_id, COALESCE(price_type, 'retail'), price, min_quantity, is_active, valid_from, valid_until, created_at, updated_at"

func (r *postgresRepository) Create(ctx context.Context, p *ProductPrice) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO product_prices (organization_id, store_id, product_id, variant_id, price_type, price, min_quantity, is_active, valid_from, valid_until)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, NULLIF($5, ''), $6, $7, COALESCE($8, TRUE), $9, $10)
		RETURNING id, COALESCE(price_type, 'retail'), created_at, updated_at`,
		p.OrganizationID, p.StoreID, p.ProductID, nullableStr(p.VariantID), p.PriceType,
		p.Price, p.MinQuantity, p.IsActive, p.ValidFrom, p.ValidUntil,
	).Scan(&p.ID, &p.PriceType, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres: create price: %w", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, orgID, storeID, productID, priceID string) (*ProductPrice, error) {
	var p ProductPrice
	err := r.db.QueryRowContext(ctx, `
		SELECT `+priceColumns+`
		FROM product_prices
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3 AND id = $4
		LIMIT 1`, orgID, storeID, productID, priceID).Scan(scanPriceDest(&p)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find price: %w", err)
	}
	return &p, nil
}

func (r *postgresRepository) FindByProduct(ctx context.Context, orgID, storeID, productID string, onlyActive bool) ([]ProductPrice, error) {
	q := `
		SELECT ` + priceColumns + `
		FROM product_prices
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3`
	if onlyActive {
		q += ` AND is_active = TRUE`
	}
	q += ` ORDER BY min_quantity ASC, price ASC`
	rows, err := r.db.QueryContext(ctx, q, orgID, storeID, productID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list prices: %w", err)
	}
	defer rows.Close()

	list := []ProductPrice{}
	for rows.Next() {
		var p ProductPrice
		if err := rows.Scan(scanPriceDest(&p)...); err != nil {
			return nil, fmt.Errorf("postgres: scan price: %w", err)
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *postgresRepository) Update(ctx context.Context, p *ProductPrice) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE product_prices
		SET variant_id = NULLIF($5, '')::uuid, price_type = NULLIF($6, ''),
			price = $7, min_quantity = $8, is_active = $9,
			valid_from = $10, valid_until = $11, updated_at = NOW()
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3 AND id = $4`,
		p.OrganizationID, p.StoreID, p.ProductID, p.ID,
		nullableStr(p.VariantID), p.PriceType, p.Price, p.MinQuantity,
		p.IsActive, p.ValidFrom, p.ValidUntil)
	if err != nil {
		return fmt.Errorf("postgres: update price: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: update price rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	updated, err := r.FindByID(ctx, p.OrganizationID, p.StoreID, p.ProductID, p.ID)
	if err != nil {
		return err
	}
	*p = *updated
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, orgID, storeID, productID, priceID string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM product_prices
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3 AND id = $4`,
		orgID, storeID, productID, priceID)
	if err != nil {
		return fmt.Errorf("postgres: delete price: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: delete price rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// FindEffective mencari harga berlaku: aktif, tipe cocok, jendela valid,
// min_quantity <= qty; variant-specific didahulukan lalu harga termurah.
// sql.ErrNoRows dipetakan ke (nil, nil): tidak ada harga berlaku.
func (r *postgresRepository) FindEffective(ctx context.Context, orgID, storeID, productID string, variantID *string, priceType string, qty int, now time.Time) (*ProductPrice, error) {
	var variantParam any
	if variantID != nil && *variantID != "" {
		variantParam = *variantID
	} else {
		variantParam = nil
	}
	var p ProductPrice
	err := r.db.QueryRowContext(ctx, `
		SELECT `+priceColumns+`
		FROM product_prices
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3
			AND is_active = TRUE
			AND price_type = $4
			AND min_quantity <= $5
			AND (valid_from IS NULL OR valid_from <= $6)
			AND (valid_until IS NULL OR valid_until >= $6)
			AND (variant_id IS NULL OR variant_id = $7::uuid OR $7 IS NULL)
		ORDER BY CASE WHEN variant_id IS NULL THEN 1 ELSE 0 END, price ASC
		LIMIT 1`,
		orgID, storeID, productID, priceType, qty, now, variantParam).Scan(scanPriceDest(&p)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres: effective price: %w", err)
	}
	return &p, nil
}

// scanPriceDest memetakan kolom product_prices ke struct.
// variant_id nullable: scan via NullString lalu petakan ke *string.
func scanPriceDest(p *ProductPrice) []any {
	ns := &sql.NullString{}
	return scanPriceDestWithVariant(p, &nullableVariantID{ns: ns, dst: &p.VariantID})
}

// nullableVariantID memetakan variant_id NULL ke *string nil.
type nullableVariantID struct {
	ns  *sql.NullString
	dst **string
}

func (n *nullableVariantID) Scan(src any) error {
	if err := n.ns.Scan(src); err != nil {
		return err
	}
	if !n.ns.Valid || n.ns.String == "" {
		*n.dst = nil
		return nil
	}
	v := n.ns.String
	*n.dst = &v
	return nil
}

func scanPriceDestWithVariant(p *ProductPrice, vid *nullableVariantID) []any {
	return []any{
		&p.ID, &p.OrganizationID, &p.StoreID, &p.ProductID,
		vid, &p.PriceType, numeric(&p.Price),
		&p.MinQuantity, &p.IsActive, &p.ValidFrom, &p.ValidUntil,
		&p.CreatedAt, &p.UpdatedAt,
	}
}

func nullableStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// numericValue dan numeric membaca NUMERIC sebagai int64 rupiah tanpa float.
// Nilai harga selalu bilangan bulat (sen tidak dipakai); parse string
// desimal dan tolak pecahan supaya tidak ada rounding diam-diam.
type numericValue int64

func numeric(dst *int64) *numericValue {
	return (*numericValue)(dst)
}

func (n *numericValue) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	case int64:
		*n = numericValue(v)
		return nil
	default:
		return fmt.Errorf("postgres: decode numeric: %T", src)
	}
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '.'); i >= 0 {
		frac := strings.TrimRight(s[i+1:], "0")
		if frac != "" {
			return fmt.Errorf("postgres: decode numeric %q: pecahan tidak didukung", s)
		}
		s = s[:i]
	}
	var v int64
	if _, err := fmt.Sscan(s, &v); err != nil {
		return fmt.Errorf("postgres: decode numeric %q: %w", s, err)
	}
	*n = numericValue(v)
	return nil
}
