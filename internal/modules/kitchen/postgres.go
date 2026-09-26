package kitchen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type postgresRepository struct {
	db *sql.DB
}

var _ Repository = (*postgresRepository)(nil)

func NewRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

// Kolom dengan COALESCE nullable ke zero-value aman untuk Scan.
const kitchenColumns = `id, sale_id, organization_id, store_id, status, priority,
	COALESCE(notes, ''), started_at, completed_at, created_at, updated_at`

const kitchenItemColumns = `id, kitchen_sale_id, sale_item_id, product_id, variant_id,
	quantity, status, COALESCE(notes, ''), created_at, updated_at`

func (r *postgresRepository) Create(ctx context.Context, ks *KitchenSale) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres: begin kitchen: %w", err)
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx, `
		INSERT INTO kitchen_sales (sale_id, organization_id, store_id, status, priority, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, started_at, completed_at, created_at, updated_at`,
		ks.SaleID, ks.OrganizationID, ks.StoreID, StatusPending, ks.Priority, ks.Notes,
	).Scan(&ks.ID, nullTime(&ks.StartedAt), nullTime(&ks.CompletedAt), &ks.CreatedAt, &ks.UpdatedAt)
	if err != nil {
		return mapCreateError(err)
	}

	ks.Status = StatusPending
	for i := range ks.Items {
		it := &ks.Items[i]
		it.KitchenSaleID = ks.ID
		it.Status = StatusPending
		err := tx.QueryRowContext(ctx, `
			INSERT INTO kitchen_sale_items
				(kitchen_sale_id, sale_item_id, product_id, variant_id, quantity, status, notes)
			VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6, $7)
			RETURNING id, created_at, updated_at`,
			ks.ID, it.SaleItemID, it.ProductID, nullableStr(it.VariantID),
			it.Quantity, StatusPending, it.Notes,
		).Scan(&it.ID, &it.CreatedAt, &it.UpdatedAt)
		if err != nil {
			return mapCreateError(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres: commit kitchen: %w", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, orgID, storeID, id string) (*KitchenSale, error) {
	var ks KitchenSale
	err := r.db.QueryRowContext(ctx, `
		SELECT `+kitchenColumns+`
		FROM kitchen_sales
		WHERE organization_id = $1 AND store_id = $2 AND id = $3
		LIMIT 1`, orgID, storeID, id).Scan(scanKitchenDest(&ks)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find kitchen: %w", err)
	}
	items, err := scanItems(ctx, r.db, ks.ID)
	if err != nil {
		return nil, err
	}
	ks.Items = items
	return &ks, nil
}

func (r *postgresRepository) FindQueue(ctx context.Context, orgID, storeID string) ([]KitchenSale, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+kitchenColumns+`
		FROM kitchen_sales
		WHERE organization_id = $1 AND store_id = $2
		  AND status IN ('pending', 'preparing')
		ORDER BY priority DESC, created_at ASC`, orgID, storeID)
	if err != nil {
		return nil, fmt.Errorf("postgres: queue kitchen: %w", err)
	}
	defer rows.Close()

	var out []KitchenSale
	for rows.Next() {
		var ks KitchenSale
		if err := rows.Scan(scanKitchenDest(&ks)...); err != nil {
			return nil, fmt.Errorf("postgres: scan kitchen: %w", err)
		}
		items, err := scanItems(ctx, r.db, ks.ID)
		if err != nil {
			return nil, err
		}
		ks.Items = items
		out = append(out, ks)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: queue kitchen: %w", err)
	}
	if out == nil {
		out = []KitchenSale{}
	}
	return out, nil
}

func (r *postgresRepository) FindBySaleID(ctx context.Context, saleID string) (*KitchenSale, error) {
	var ks KitchenSale
	err := r.db.QueryRowContext(ctx, `
		SELECT `+kitchenColumns+`
		FROM kitchen_sales
		WHERE sale_id = $1
		LIMIT 1`, saleID).Scan(scanKitchenDest(&ks)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find kitchen by sale: %w", err)
	}
	items, err := scanItems(ctx, r.db, ks.ID)
	if err != nil {
		return nil, err
	}
	ks.Items = items
	return &ks, nil
}

// UpdateStatus menulis status baru bila berbeda: UPDATE bersyarat sekaligus
// enforcement; 0 baris = NotFound (transisi sudah dicek service).
func (r *postgresRepository) UpdateStatus(ctx context.Context, orgID, storeID, id, status string, started, completed *time.Time) (*KitchenSale, error) {
	var ks KitchenSale
	err := r.db.QueryRowContext(ctx, `
		UPDATE kitchen_sales
		SET status = $4,
			started_at = COALESCE($5, started_at),
			completed_at = COALESCE($6, completed_at),
			updated_at = NOW()
		WHERE organization_id = $1 AND store_id = $2 AND id = $3 AND status <> $4
		RETURNING `+kitchenColumns,
		orgID, storeID, id, status, toNullTime(started), toNullTime(completed),
	).Scan(scanKitchenDest(&ks)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: update kitchen status: %w", err)
	}
	items, err := scanItems(ctx, r.db, ks.ID)
	if err != nil {
		return nil, err
	}
	ks.Items = items
	return &ks, nil
}

// mapCreateError memetakan pelanggaran unique satu-sale-satu-antrian dan FK
// ke domain error; duplikat sale jadi invalid input (satu sale satu antrian).
func mapCreateError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505":
			if pqErr.Constraint == "unique_kitchen_sale" {
				return fmt.Errorf("postgres: create kitchen: %w", ErrInvalidInput)
			}
		case "23503":
			return fmt.Errorf("postgres: create kitchen: %w", ErrInvalidInput)
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return fmt.Errorf("postgres: create kitchen: %w", err)
}

func (r *postgresRepository) UpdateItemStatus(ctx context.Context, orgID, storeID, kitchenID, itemID, status string) (*KitchenSaleItem, error) {
	var it KitchenSaleItem
	err := r.db.QueryRowContext(ctx, `
		UPDATE kitchen_sale_items ki
		SET status = $5, updated_at = NOW()
		FROM kitchen_sales ks
		WHERE ki.id = $4 AND ki.kitchen_sale_id = $3
		  AND ks.id = ki.kitchen_sale_id
		  AND ks.organization_id = $1 AND ks.store_id = $2
		  AND ki.status <> $5
		RETURNING `+`ki.id, ki.kitchen_sale_id, ki.sale_item_id, ki.product_id, ki.variant_id,
			ki.quantity, ki.status, COALESCE(ki.notes, ''), ki.created_at, ki.updated_at`,
		orgID, storeID, kitchenID, itemID, status,
	).Scan(scanItemDest(&it)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: update kitchen item: %w", err)
	}
	return &it, nil
}

func (r *postgresRepository) FindItems(ctx context.Context, kitchenID string) ([]KitchenSaleItem, error) {
	return scanItems(ctx, r.db, kitchenID)
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func scanItems(ctx context.Context, q querier, kitchenID string) ([]KitchenSaleItem, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT `+kitchenItemColumns+`
		FROM kitchen_sale_items
		WHERE kitchen_sale_id = $1
		ORDER BY created_at ASC`, kitchenID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list kitchen items: %w", err)
	}
	defer rows.Close()

	var out []KitchenSaleItem
	for rows.Next() {
		var it KitchenSaleItem
		if err := rows.Scan(scanItemDest(&it)...); err != nil {
			return nil, fmt.Errorf("postgres: scan kitchen item: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list kitchen items: %w", err)
	}
	if out == nil {
		out = []KitchenSaleItem{}
	}
	return out, nil
}

func scanKitchenDest(ks *KitchenSale) []any {
	return []any{
		&ks.ID, &ks.SaleID, &ks.OrganizationID, &ks.StoreID,
		&ks.Status, &ks.Priority, &ks.Notes,
		nullTime(&ks.StartedAt), nullTime(&ks.CompletedAt),
		&ks.CreatedAt, &ks.UpdatedAt,
	}
}

func scanItemDest(it *KitchenSaleItem) []any {
	return []any{
		&it.ID, &it.KitchenSaleID, &it.SaleItemID, &it.ProductID,
		nullStr(&it.VariantID), &it.Quantity, &it.Status, &it.Notes,
		&it.CreatedAt, &it.UpdatedAt,
	}
}

func nullableStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// nullTime/nullStr/toNullTime menjembatani *time.Time/*string dengan
// kolom NULLABLE tanpa sql.Null* di struct domain.
type nullTimeT struct{ dst **time.Time }

func nullTime(dst **time.Time) *nullTimeT { return &nullTimeT{dst} }

func (n *nullTimeT) Scan(src any) error {
	if src == nil {
		*n.dst = nil
		return nil
	}
	var t time.Time
	switch v := src.(type) {
	case time.Time:
		t = v
	case []byte:
		parsed, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", string(v))
		if err != nil {
			parsed, err = time.Parse(time.RFC3339, string(v))
			if err != nil {
				return fmt.Errorf("postgres: scan time: %w", err)
			}
		}
		t = parsed
	default:
		return fmt.Errorf("postgres: scan time: unexpected %T", src)
	}
	*n.dst = &t
	return nil
}

func toNullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

type nullStrT struct{ dst **string }

func nullStr(dst **string) *nullStrT { return &nullStrT{dst} }

func (n *nullStrT) Scan(src any) error {
	if src == nil {
		*n.dst = nil
		return nil
	}
	switch v := src.(type) {
	case string:
		*n.dst = &v
	case []byte:
		s := string(v)
		*n.dst = &s
	default:
		return fmt.Errorf("postgres: scan nullable string: unexpected %T", src)
	}
	return nil
}

// CancelBySale membatalkan antrian aktif (pending/preparing) milik satu sale
// beserta itemnya yang masih pending/preparing. Idempoten: 0 baris = tidak
// ada antrian aktif untuk sale itu.
func (r *postgresRepository) CancelBySale(ctx context.Context, orgID, storeID, saleID string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE kitchen_sales
		SET status = $4, completed_at = NOW(), updated_at = NOW()
		WHERE organization_id = $1 AND store_id = $2 AND sale_id = $3
		  AND status IN ('pending', 'preparing')`,
		orgID, storeID, saleID, StatusCancelled,
	)
	if err != nil {
		return 0, fmt.Errorf("postgres: cancel kitchen by sale: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("postgres: cancel kitchen by sale: %w", err)
	}
	if n > 0 {
		if _, err := r.db.ExecContext(ctx, `
			UPDATE kitchen_sale_items ki
			SET status = $2, updated_at = NOW()
			FROM kitchen_sales ks
			WHERE ki.kitchen_sale_id = ks.id AND ks.sale_id = $1
			  AND ki.status IN ('pending', 'preparing')`,
			saleID, StatusCancelled,
		); err != nil {
			return n, fmt.Errorf("postgres: cancel kitchen items: %w", err)
		}
	}
	return n, nil
}
