package variants

import (
	"context"
	"errors"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Error domain module variants.
var (
	ErrNotFound        = errors.New("variant not found")
	ErrSKUExists       = errors.New("variant SKU already exists in this store")
	ErrInvalidInput    = errors.New("variant: invalid input")
	ErrStoreNotFound   = errors.New("variant: store not found")
	ErrProductNotFound = errors.New("variant: product not found")
)

// ProductVariant adalah varian barang dalam satu product+store.
type ProductVariant struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	StoreID        string    `json:"store_id"`
	ProductID      string    `json:"product_id"`
	Name           string    `json:"name"`
	SKU            string    `json:"sku"`
	Stock          int       `json:"stock"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Repository adalah kontrak penyimpanan variant.
// Semua query ter-scope org+store+product; cross-org ditutup sebagai NotFound.
type Repository interface {
	Create(ctx context.Context, v *ProductVariant) error
	FindByID(ctx context.Context, orgID, storeID, productID, id string) (*ProductVariant, error)
	FindByProduct(ctx context.Context, orgID, storeID, productID string) ([]ProductVariant, error)
	Update(ctx context.Context, v *ProductVariant) error
	Delete(ctx context.Context, orgID, storeID, productID, id string) error
	ExistsBySKU(ctx context.Context, storeID, sku, excludeID string) (bool, error)
}

// StoreChecker adalah port kepemilikan store untuk validasi same-org.
type StoreChecker interface {
	FindByIDInOrg(ctx context.Context, orgID, id string) (*stores.Store, error)
}

// ProductChecker adalah port kepemilikan product untuk validasi same-store.
type ProductChecker interface {
	FindByID(ctx context.Context, orgID, storeID, id string) (*products.Product, error)
}
