package stock

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

// movementColumns COALESCE nullable ke zero-value aman untuk Scan.
const movementColumns = "id, organization_id, store_id, product_id, COALESCE(variant_id::text, ''), type, quantity, stock_before, stock_after, COALESCE(reference_type, ''), COALESCE(reference_id::text, ''), COALESCE(notes, ''), COALESCE(created_by::text, ''), created_at"

// Record menulis satu movement sekaligus update stok live dalam satu tx.
// Catatan: snapshot stock_before/after yang diminta spec di layer service
// dilipat ke dalam tx ini supaya baca-stok + tulis-movement + update-stok atomik
// (tanpa race antar pencatatan bersamaan); service hanya validasi + delegasi.
func (r *postgresRepository) Record(ctx context.Context, m *StockMovement, allowNegative bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres: begin stock movement: %w", err)
	}
	defer tx.Rollback()

	var before int
	if m.VariantID != nil && strings.TrimSpace(*m.VariantID) != "" {
		err = tx.QueryRowContext(ctx, `
			SELECT stock FROM product_variants
			WHERE id = $1 FOR UPDATE`, *m.VariantID).Scan(&before)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrVariantNotFound
			}
			return fmt.Errorf("postgres: lock variant stock: %w", err)
		}
	} else {
		err = tx.QueryRowContext(ctx, `
			SELECT stock FROM products
			WHERE organization_id = $1 AND store_id = $2 AND id = $3 AND deleted_at IS NULL
			FOR UPDATE`, m.OrganizationID, m.StoreID, m.ProductID).Scan(&before)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrProductNotFound
			}
			return fmt.Errorf("postgres: lock product stock: %w", err)
		}
	}

	var after int
	switch m.Type {
	case TypeIn, TypeReturn:
		after = before + m.Quantity
	case TypeOut:
		after = before - m.Quantity
		if after < 0 {
			return ErrInsufficient
		}
	case TypeAdjustment:
		after = before + m.Quantity
		if after < 0 && !allowNegative {
			return ErrInsufficient
		}
	default:
		return ErrInvalidType
	}
	m.StockBefore = before
	m.StockAfter = after

	err = tx.QueryRowContext(ctx, `
		INSERT INTO stock_movements (organization_id, store_id, product_id, variant_id, type, quantity, stock_before, stock_after, reference_type, reference_id, notes, created_by)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6, $7, $8, NULLIF($9, ''), NULLIF($10, '')::uuid, NULLIF($11, ''), NULLIF($12, '')::uuid)
		RETURNING id, created_at`,
		m.OrganizationID, m.StoreID, m.ProductID, nullableStr(m.VariantID),
		m.Type, m.Quantity, m.StockBefore, m.StockAfter,
		m.ReferenceType, m.ReferenceID, m.Notes, m.CreatedBy,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return ErrInvalidInput
		}
		if errors.As(err, &pqErr) && pqErr.Code == "23514" {
			return ErrInvalidType
		}
		return mapWriteError(err)
	}

	if m.VariantID != nil && strings.TrimSpace(*m.VariantID) != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE product_variants SET stock = $2 WHERE id = $1`, *m.VariantID, after); err != nil {
			return mapWriteError(err)
		}
	} else {
		if _, err := tx.ExecContext(ctx, `UPDATE products SET stock = $2, updated_at = NOW() WHERE id = $1`, m.ProductID, after); err != nil {
			return mapWriteError(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres: commit stock movement: %w", err)
	}
	return nil
}

func (r *postgresRepository) FindByStore(ctx context.Context, orgID, storeID string, filter Filter) ([]StockMovement, int, error) {
	where, args := buildFilter(orgID, storeID, filter)

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM stock_movements `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count stock movements: %w", err)
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
		SELECT `+movementColumns+`
		FROM stock_movements `+where+`
		ORDER BY created_at DESC
		LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list stock movements: %w", err)
	}
	defer rows.Close()

	list := []StockMovement{}
	for rows.Next() {
		var m StockMovement
		if err := scanMovement(rows, &m); err != nil {
			return nil, 0, fmt.Errorf("postgres: scan stock movement: %w", err)
		}
		list = append(list, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres: iterate stock movements: %w", err)
	}
	return list, total, nil
}

func (r *postgresRepository) FindByProduct(ctx context.Context, orgID, storeID, productID string) ([]StockMovement, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+movementColumns+`
		FROM stock_movements
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3
		ORDER BY created_at DESC`, orgID, storeID, productID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list product movements: %w", err)
	}
	defer rows.Close()

	list := []StockMovement{}
	for rows.Next() {
		var m StockMovement
		if err := scanMovement(rows, &m); err != nil {
			return nil, fmt.Errorf("postgres: scan stock movement: %w", err)
		}
		list = append(list, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate stock movements: %w", err)
	}
	return list, nil
}

func (r *postgresRepository) GetSummary(ctx context.Context, orgID, storeID, productID string) (*StockSummary, error) {
	var stock int
	err := r.db.QueryRowContext(ctx, `
		SELECT stock FROM products
		WHERE organization_id = $1 AND store_id = $2 AND id = $3 AND deleted_at IS NULL`,
		orgID, storeID, productID).Scan(&stock)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("postgres: product stock: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, stock FROM product_variants
		WHERE product_id = $1`, productID)
	if err != nil {
		return nil, fmt.Errorf("postgres: variant stocks: %w", err)
	}
	defer rows.Close()

	sum := &StockSummary{ProductID: productID, ProductStock: stock, Variants: []VariantStock{}}
	for rows.Next() {
		var v VariantStock
		if err := rows.Scan(&v.VariantID, &v.Stock); err != nil {
			return nil, fmt.Errorf("postgres: scan variant stock: %w", err)
		}
		sum.Variants = append(sum.Variants, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate variant stocks: %w", err)
	}
	return sum, nil
}

// buildFilter menyusun WHERE ter-scope org+store; product/type/rentang tanggal opsional.
func buildFilter(orgID, storeID string, filter Filter) (string, []any) {
	where := `WHERE organization_id = $1 AND store_id = $2`
	args := []any{orgID, storeID}
	if strings.TrimSpace(filter.ProductID) != "" {
		args = append(args, strings.TrimSpace(filter.ProductID))
		where += ` AND product_id = $` + itoa(len(args))
	}
	if strings.TrimSpace(filter.Type) != "" {
		args = append(args, strings.TrimSpace(filter.Type))
		where += ` AND type = $` + itoa(len(args))
	}
	if !filter.From.IsZero() {
		args = append(args, filter.From)
		where += ` AND created_at >= $` + itoa(len(args))
	}
	if !filter.To.IsZero() {
		args = append(args, filter.To)
		where += ` AND created_at <= $` + itoa(len(args))
	}
	return where, args
}

// rowScanner mencakup *sql.Row dan *sql.Rows untuk scan movement.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanMovement memetakan satu baris stock_movements ke struct.
func scanMovement(s rowScanner, m *StockMovement) error {
	var variantID sql.NullString
	if err := s.Scan(
		&m.ID, &m.OrganizationID, &m.StoreID, &m.ProductID, &variantID,
		&m.Type, &m.Quantity, &m.StockBefore, &m.StockAfter,
		&m.ReferenceType, &m.ReferenceID, &m.Notes, &m.CreatedBy, &m.CreatedAt,
	); err != nil {
		return err
	}
	if variantID.Valid && variantID.String != "" {
		v := variantID.String
		m.VariantID = &v
	}
	return nil
}

func nullableStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func mapWriteError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23503" {
		return ErrInvalidInput
	}
	return fmt.Errorf("postgres: write stock movement: %w", err)
}

func itoa(n int) string {
	return fmt.Sprint(n)
}
