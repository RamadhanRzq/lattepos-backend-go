package recipes

import (
	"context"
	"errors"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stock"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Error domain module recipes.
var (
	ErrNotFound            = errors.New("recipe not found")
	ErrInvalidInput        = errors.New("recipe: invalid input")
	ErrStoreNotFound       = errors.New("recipe: store not found")
	ErrProductNotFound     = errors.New("recipe: product not found")
	ErrIngredientNotFound  = errors.New("recipe: ingredient product not found")
	ErrIngredientSelf      = errors.New("recipe: ingredient tidak boleh product sendiri")
	ErrIngredientDuplicate = errors.New("recipe: ingredient duplikat")
	ErrVersionExists       = errors.New("recipe: version sudah dipakai product ini")
)

// ReferenceType dipakai di stock_movements saat resep dikonsumsi.
const ReferenceType = "recipe"

// Recipe adalah komposisi bahan baku milik satu product dalam satu store.
// Satu product boleh punya beberapa versi; hanya satu yang is_active.
type Recipe struct {
	ID             string       `json:"id"`
	OrganizationID string       `json:"organization_id"`
	StoreID        string       `json:"store_id"`
	ProductID      string       `json:"product_id"`
	Name           string       `json:"name"`
	Version        int          `json:"version"`
	YieldQuantity  float64      `json:"yield_quantity"`
	IsActive       bool         `json:"is_active"`
	Notes          string       `json:"notes,omitempty"`
	CreatedBy      string       `json:"created_by,omitempty"`
	Items          []RecipeItem `json:"items"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// RecipeItem adalah satu bahan dalam recipe. Quantity memakai satuan bahan itu
// sendiri (ml, gram, pcs); stok bahan dikurangi sebanyak quantity x qty jual.
type RecipeItem struct {
	ID                  string  `json:"id"`
	RecipeID            string  `json:"recipe_id"`
	IngredientProductID string  `json:"ingredient_product_id"`
	IngredientName      string  `json:"ingredient_name,omitempty"`
	IngredientSKU       string  `json:"ingredient_sku,omitempty"`
	Quantity            float64 `json:"quantity"`
	Unit                string  `json:"unit"`
	WastagePercentage   float64 `json:"wastage_percentage"`
	Sequence            int     `json:"sequence"`
	Notes               string  `json:"notes,omitempty"`
}

// ItemInput adalah satu baris bahan saat create/update recipe (full replace).
type ItemInput struct {
	IngredientProductID string  `json:"ingredient_product_id"`
	Quantity            float64 `json:"quantity"`
	Unit                string  `json:"unit"`
	WastagePercentage   float64 `json:"wastage_percentage"`
	Notes               string  `json:"notes"`
}

// Repository adalah kontrak penyimpanan recipe beserta itemnya.
// Semua query ter-scope org+store+product; cross-org ditutup sebagai NotFound.
type Repository interface {
	// WithTx menjalankan fn dalam satu transaksi; dipakai saat menggeser
	// recipe aktif (deaktivasi lain + aktivasi target harus atomik).
	WithTx(ctx context.Context, fn func(context.Context) error) error
	Create(ctx context.Context, r *Recipe) error
	FindByID(ctx context.Context, orgID, storeID, productID, id string) (*Recipe, error)
	FindByProduct(ctx context.Context, orgID, storeID, productID string) ([]Recipe, error)
	FindByStore(ctx context.Context, orgID, storeID string) ([]Recipe, error)
	FindActiveByProduct(ctx context.Context, orgID, storeID, productID string) (*Recipe, error)
	Update(ctx context.Context, r *Recipe) error
	Delete(ctx context.Context, orgID, storeID, productID, id string) error
	DeactivateOthers(ctx context.Context, orgID, storeID, productID, keepID string) error
	SetActive(ctx context.Context, orgID, storeID, productID, id string, active bool) error
	LoadItems(ctx context.Context, recipeID string) ([]RecipeItem, error)
	ReplaceItems(ctx context.Context, recipeID string, items []RecipeItem) error
}

// StoreChecker adalah port kepemilikan store untuk validasi same-org.
type StoreChecker interface {
	FindByIDInOrg(ctx context.Context, orgID, id string) (*stores.Store, error)
}

// ProductChecker adalah port kepemilikan product untuk validasi same-store.
type ProductChecker interface {
	FindByID(ctx context.Context, orgID, storeID, id string) (*products.Product, error)
}

// Recorder adalah port pencatatan mutasi stok untuk konsumsi bahan baku.
// Dipenuhi langsung oleh *stock.Service; saat dipanggil di dalam WithTx sale,
// movement ikut transaksi yang sama.
type Recorder interface {
	Record(ctx context.Context, orgID, storeID, productID string, variantID *string, typ string, qty int, refType, refID, notes, createdBy string, allowNegative bool) (*stock.StockMovement, error)
}
