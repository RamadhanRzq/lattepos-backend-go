package sales

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/ramadhanrzq/backend-go/internal/dbtx"
)

type postgresRepository struct {
	db *sql.DB
}

// getDB mengembalikan tx dari context bila Create/StockOut berbagi WithTx,
// kalau tidak ya koneksi pool biasa.
func (r *postgresRepository) getDB(ctx context.Context) dbtx.DB {
	if tx, ok := dbtx.From(ctx); ok {
		return tx
	}
	return r.db
}

// WithTx menjalankan fn dalam satu transaksi; nested call join tx yang sama.
func (r *postgresRepository) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return dbtx.WithTx(r.db, ctx, fn)
}

var _ Repository = (*postgresRepository)(nil)

func NewRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

// saleColumns COALESCE nullable ke zero-value aman untuk Scan.
const saleColumns = "id, store_id, organization_id, user_id, total_amount, discount_amount, tax_amount, grand_total, payment_method, status, COALESCE(notes, ''), created_at, updated_at"

func (r *postgresRepository) Create(ctx context.Context, s *Sale) error {
	// Join tx pemanggil bila dipanggil dalam WithTx (atomic sale+stok),
	// standalone begin/commit sendiri seperti sebelumnya.
	if _, ok := dbtx.From(ctx); ok {
		return r.insertSale(ctx, r.getDB(ctx), s)
	}
	return r.WithTx(ctx, func(txCtx context.Context) error {
		return r.insertSale(txCtx, r.getDB(txCtx), s)
	})
}

// insertSale menulis header + items ke q (tx atau pool); tanpa begin/commit.
func (r *postgresRepository) insertSale(ctx context.Context, q dbtx.DB, s *Sale) error {
	err := q.QueryRowContext(ctx, `
		INSERT INTO sales (store_id, organization_id, user_id, total_amount, discount_amount, tax_amount, grand_total, payment_method, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, ''))
		RETURNING id, created_at, updated_at`,
		s.StoreID, s.OrganizationID, s.UserID, s.TotalAmount, s.DiscountAmount,
		s.TaxAmount, s.GrandTotal, s.PaymentMethod, StatusPending, s.Notes,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return mapCreateError(err)
	}

	s.Status = StatusPending
	for i := range s.Items {
		it := &s.Items[i]
		it.SaleID = s.ID
		err := q.QueryRowContext(ctx, `
			INSERT INTO sale_items (sale_id, product_id, variant_id, quantity, unit_price, subtotal)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, created_at`,
			s.ID, it.ProductID, it.VariantID, it.Quantity, it.UnitPrice, it.Subtotal,
		).Scan(&it.ID, &it.CreatedAt)
		if err != nil {
			return mapCreateError(err)
		}
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, orgID, storeID, id string) (*Sale, error) {
	var s Sale
	err := r.getDB(ctx).QueryRowContext(ctx, `
		SELECT `+saleColumns+`
		FROM sales
		WHERE organization_id = $1 AND store_id = $2 AND id = $3
		LIMIT 1`, orgID, storeID, id).Scan(scanSaleDest(&s)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find sale: %w", err)
	}
	items, err := scanTxItems(ctx, r.getDB(ctx), s.ID)
	if err != nil {
		return nil, err
	}
	s.Items = items
	return &s, nil
}

func (r *postgresRepository) FindItems(ctx context.Context, saleID string) ([]SaleItem, error) {
	return scanTxItems(ctx, r.getDB(ctx), saleID)
}

func (r *postgresRepository) FindByStore(ctx context.Context, orgID, storeID string, filter Filter) ([]Sale, int, error) {
	where, args := buildFilter(orgID, storeID, filter)

	var total int
	if err := r.getDB(ctx).QueryRowContext(ctx, `SELECT COUNT(*) FROM sales `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count sales: %w", err)
	}

	page, limit := filter.Page, filter.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	args = append(args, limit, (page-1)*limit)
	rows, err := r.getDB(ctx).QueryContext(ctx, `
		SELECT `+saleColumns+`
		FROM sales `+where+`
		ORDER BY created_at DESC
		LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list sales: %w", err)
	}
	defer rows.Close()

	list := []Sale{}
	for rows.Next() {
		var s Sale
		if err := rows.Scan(scanSaleDest(&s)...); err != nil {
			return nil, 0, fmt.Errorf("postgres: scan sale: %w", err)
		}
		list = append(list, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres: iterate sales: %w", err)
	}
	return list, total, nil
}

// Cancel menulis status cancelled hanya dari pending: UPDATE bersyarat status
// sekaligus enforcement; 0 baris = NotFound bila missing/sudah final.
func (r *postgresRepository) Cancel(ctx context.Context, orgID, storeID, id string) (*Sale, error) {
	var s Sale
	err := r.getDB(ctx).QueryRowContext(ctx, `
		UPDATE sales
		SET status = $4, updated_at = NOW()
		WHERE organization_id = $1 AND store_id = $2 AND id = $3 AND status = $5
		RETURNING `+saleColumns, orgID, storeID, id, StatusCancelled, StatusPending,
	).Scan(scanSaleDest(&s)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			cur, ferr := r.FindByID(ctx, orgID, storeID, id)
			if ferr != nil {
				return nil, ferr
			}
			if cur.Status != StatusPending {
				return nil, ErrInvalidStatus
			}
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: cancel sale: %w", err)
	}
	items, err := scanTxItems(ctx, r.getDB(ctx), s.ID)
	if err != nil {
		return nil, err
	}
	s.Items = items
	return &s, nil
}

// buildFilter menyusun WHERE ter-scope org+store; status dan rentang tanggal opsional.
func buildFilter(orgID, storeID string, filter Filter) (string, []any) {
	where := `WHERE organization_id = $1 AND store_id = $2`
	args := []any{orgID, storeID}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where += ` AND status = $` + itoa(len(args))
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

// scanSaleDest memetakan kolom sales ke struct; NUMERIC dibaca via Numeric wrapper.
func scanSaleDest(s *Sale) []any {
	return []any{
		&s.ID, &s.StoreID, &s.OrganizationID, &s.UserID,
		numeric(&s.TotalAmount), numeric(&s.DiscountAmount),
		numeric(&s.TaxAmount), numeric(&s.GrandTotal),
		&s.PaymentMethod, &s.Status, &s.Notes, &s.CreatedAt, &s.UpdatedAt,
	}
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func scanTxItems(ctx context.Context, q querier, saleID string) ([]SaleItem, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, sale_id, product_id, variant_id, quantity, unit_price, subtotal, created_at
		FROM sale_items
		WHERE sale_id = $1
		ORDER BY created_at ASC`, saleID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list sale items: %w", err)
	}
	defer rows.Close()

	items := []SaleItem{}
	for rows.Next() {
		var it SaleItem
		var unit, sub numericValue
		if err := rows.Scan(&it.ID, &it.SaleID, &it.ProductID, &it.VariantID, &it.Quantity, &unit, &sub, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan sale item: %w", err)
		}
		it.UnitPrice, it.Subtotal = int64(unit), int64(sub)
		items = append(items, it)
	}
	if items == nil {
		items = []SaleItem{}
	}
	return items, rows.Err()
}

// mapCreateError memetakan pelanggaran FK ke domain error; unique sale tidak ada.
func mapCreateError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23503" {
		switch pqErr.Constraint {
		case "fk_sales_store", "fk_sales_org", "fk_sales_user":
			return fmt.Errorf("postgres: create sale: %w", ErrInvalidInput)
		case "fk_sale_items_sale", "fk_sale_items_product":
			return fmt.Errorf("postgres: create sale item: %w", ErrProductNotFound)
		}
		return fmt.Errorf("postgres: create sale: %w", ErrInvalidInput)
	}
	return fmt.Errorf("postgres: create sale: %w", err)
}

func itoa(n int) string {
	return fmt.Sprint(n)
}

// numericValue dan numeric membaca NUMERIC sebagai int64 rupiah tanpa float.
// Nilai transaksi selalu bilangan bulat (sen tidak dipakai); parse string
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
