package recipes

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

var _ Repository = (*postgresRepository)(nil)

func NewRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

// getDB mengembalikan tx dari context bila dipanggil di dalam WithTx.
func (r *postgresRepository) getDB(ctx context.Context) dbtx.DB {
	if tx, ok := dbtx.From(ctx); ok {
		return tx
	}
	return r.db
}

// WithTx menjalankan fn dalam satu transaksi dari pool yang sama.
func (r *postgresRepository) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return dbtx.WithTx(r.db, ctx, fn)
}

const recipeColumns = "id, organization_id, store_id, product_id, name, version, yield_quantity, is_active, COALESCE(notes, ''), COALESCE(created_by::text, ''), created_at, updated_at"

const itemColumns = "ri.id, ri.recipe_id, ri.ingredient_product_id, COALESCE(p.name, ''), COALESCE(p.sku, ''), ri.quantity, COALESCE(ri.unit, 'pcs'), ri.wastage_percentage, ri.sequence, COALESCE(ri.notes, '')"

func (r *postgresRepository) Create(ctx context.Context, rec *Recipe) error {
	err := r.getDB(ctx).QueryRowContext(ctx, `
		INSERT INTO recipes (organization_id, store_id, product_id, name, version, yield_quantity, is_active, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, '')::uuid)
		RETURNING id, created_at, updated_at`,
		rec.OrganizationID, rec.StoreID, rec.ProductID, rec.Name, rec.Version,
		rec.YieldQuantity, rec.IsActive, rec.Notes, rec.CreatedBy,
	).Scan(&rec.ID, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch {
			case pqErr.Code == "23505" && pqErr.Constraint == "unique_recipe_version_per_product":
				return ErrVersionExists
			case pqErr.Code == "23505":
				return ErrInvalidInput
			case pqErr.Code == "23503":
				return ErrProductNotFound
			}
		}
		return fmt.Errorf("postgres: create recipe: %w", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, orgID, storeID, productID, id string) (*Recipe, error) {
	var rec Recipe
	err := r.getDB(ctx).QueryRowContext(ctx, `
		SELECT `+recipeColumns+`
		FROM recipes
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3 AND id = $4
		LIMIT 1`, orgID, storeID, productID, id).Scan(scanRecipeDest(&rec)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find recipe: %w", err)
	}
	items, err := r.LoadItems(ctx, rec.ID)
	if err != nil {
		return nil, err
	}
	rec.Items = items
	return &rec, nil
}

func (r *postgresRepository) FindByProduct(ctx context.Context, orgID, storeID, productID string) ([]Recipe, error) {
	return r.queryRecipes(ctx, `
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3
		ORDER BY version DESC`, orgID, storeID, productID)
}

func (r *postgresRepository) FindByStore(ctx context.Context, orgID, storeID string) ([]Recipe, error) {
	return r.queryRecipes(ctx, `
		WHERE organization_id = $1 AND store_id = $2
		ORDER BY created_at DESC`, orgID, storeID)
}

func (r *postgresRepository) FindActiveByProduct(ctx context.Context, orgID, storeID, productID string) (*Recipe, error) {
	var rec Recipe
	err := r.getDB(ctx).QueryRowContext(ctx, `
		SELECT `+recipeColumns+`
		FROM recipes
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3 AND is_active
		LIMIT 1`, orgID, storeID, productID).Scan(scanRecipeDest(&rec)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find active recipe: %w", err)
	}
	items, err := r.LoadItems(ctx, rec.ID)
	if err != nil {
		return nil, err
	}
	rec.Items = items
	return &rec, nil
}

func (r *postgresRepository) queryRecipes(ctx context.Context, where string, args ...any) ([]Recipe, error) {
	rows, err := r.getDB(ctx).QueryContext(ctx, `SELECT `+recipeColumns+` FROM recipes `+where, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: list recipes: %w", err)
	}
	defer rows.Close()

	list := []Recipe{}
	for rows.Next() {
		var rec Recipe
		if err := rows.Scan(scanRecipeDest(&rec)...); err != nil {
			return nil, fmt.Errorf("postgres: scan recipe: %w", err)
		}
		list = append(list, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range list {
		items, err := r.LoadItems(ctx, list[i].ID)
		if err != nil {
			return nil, err
		}
		list[i].Items = items
	}
	return list, nil
}

func (r *postgresRepository) Update(ctx context.Context, rec *Recipe) error {
	res, err := r.getDB(ctx).ExecContext(ctx, `
		UPDATE recipes
		SET name = $5, version = $6, yield_quantity = $7, is_active = $8,
			notes = $9, updated_at = NOW()
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3 AND id = $4`,
		rec.OrganizationID, rec.StoreID, rec.ProductID, rec.ID,
		rec.Name, rec.Version, rec.YieldQuantity, rec.IsActive, rec.Notes)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch {
			case pqErr.Code == "23505" && pqErr.Constraint == "unique_recipe_version_per_product":
				return ErrVersionExists
			case pqErr.Code == "23505":
				return ErrInvalidInput
			}
		}
		return fmt.Errorf("postgres: update recipe: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: update recipe rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, orgID, storeID, productID, id string) error {
	res, err := r.getDB(ctx).ExecContext(ctx, `
		DELETE FROM recipes
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3 AND id = $4`,
		orgID, storeID, productID, id)
	if err != nil {
		return fmt.Errorf("postgres: delete recipe: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: delete recipe rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeactivateOthers mematikan is_active recipe lain milik product yang sama.
// keepID kosong berarti semua versi product itu dinonaktifkan; dipakai sebelum
// insert/update recipe aktif supaya partial unique index tidak bentrok.
func (r *postgresRepository) DeactivateOthers(ctx context.Context, orgID, storeID, productID, keepID string) error {
	_, err := r.getDB(ctx).ExecContext(ctx, `
		UPDATE recipes
		SET is_active = FALSE, updated_at = NOW()
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3
			AND is_active
			AND ($4 = '' OR id <> NULLIF($4, '')::uuid)`,
		orgID, storeID, productID, keepID)
	if err != nil {
		return fmt.Errorf("postgres: deactivate other recipes: %w", err)
	}
	return nil
}

func (r *postgresRepository) SetActive(ctx context.Context, orgID, storeID, productID, id string, active bool) error {
	res, err := r.getDB(ctx).ExecContext(ctx, `
		UPDATE recipes
		SET is_active = $5, updated_at = NOW()
		WHERE organization_id = $1 AND store_id = $2 AND product_id = $3 AND id = $4`,
		orgID, storeID, productID, id, active)
	if err != nil {
		return fmt.Errorf("postgres: set recipe active: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: set recipe active rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) LoadItems(ctx context.Context, recipeID string) ([]RecipeItem, error) {
	rows, err := r.getDB(ctx).QueryContext(ctx, `
		SELECT `+itemColumns+`
		FROM recipe_items ri
		LEFT JOIN products p ON p.id = ri.ingredient_product_id
		WHERE ri.recipe_id = $1
		ORDER BY ri.sequence ASC, ri.created_at ASC`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("postgres: load recipe items: %w", err)
	}
	defer rows.Close()

	list := []RecipeItem{}
	for rows.Next() {
		var it RecipeItem
		if err := rows.Scan(&it.ID, &it.RecipeID, &it.IngredientProductID, &it.IngredientName,
			&it.IngredientSKU, &it.Quantity, &it.Unit, &it.WastagePercentage, &it.Sequence, &it.Notes); err != nil {
			return nil, fmt.Errorf("postgres: scan recipe item: %w", err)
		}
		list = append(list, it)
	}
	return list, rows.Err()
}

// ReplaceItems menghapus lalu menulis ulang seluruh item recipe (full replace).
func (r *postgresRepository) ReplaceItems(ctx context.Context, recipeID string, items []RecipeItem) error {
	if _, err := r.getDB(ctx).ExecContext(ctx, `DELETE FROM recipe_items WHERE recipe_id = $1`, recipeID); err != nil {
		return fmt.Errorf("postgres: clear recipe items: %w", err)
	}
	for _, it := range items {
		_, err := r.getDB(ctx).ExecContext(ctx, `
			INSERT INTO recipe_items (recipe_id, organization_id, store_id, ingredient_product_id, quantity, unit, wastage_percentage, sequence, notes)
			SELECT $1, r.organization_id, r.store_id, $2, $3, $4, $5, $6, $7
			FROM recipes r WHERE r.id = $1`,
			recipeID, it.IngredientProductID, it.Quantity, it.Unit,
			it.WastagePercentage, it.Sequence, it.Notes)
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) {
				switch {
				case pqErr.Code == "23505" && strings.Contains(pqErr.Constraint, "unique_recipe_item_per_ingredient"):
					return ErrIngredientDuplicate
				case pqErr.Code == "23503":
					return ErrIngredientNotFound
				case pqErr.Code == "23514":
					return ErrInvalidInput
				}
			}
			return fmt.Errorf("postgres: insert recipe item: %w", err)
		}
	}
	return nil
}

func scanRecipeDest(rec *Recipe) []any {
	return []any{
		&rec.ID, &rec.OrganizationID, &rec.StoreID, &rec.ProductID, &rec.Name,
		&rec.Version, &rec.YieldQuantity, &rec.IsActive, &rec.Notes, &rec.CreatedBy,
		&rec.CreatedAt, &rec.UpdatedAt,
	}
}
