package prices

import (
	"context"
	"errors"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	"github.com/ramadhanrzq/backend-go/internal/modules/variants"
)

// Error domain module prices.
var (
	ErrNotFound        = errors.New("price not found")
	ErrInvalidInput    = errors.New("price: invalid input")
	ErrStoreNotFound   = errors.New("price: store not found")
	ErrProductNotFound = errors.New("price: product not found")
	ErrVariantNotFound = errors.New("price: variant not found")
)

// ProductPrice adalah harga per product/variant dalam satu store.
// Price int64 rupiah; NUMERIC DB selalu bilangan bulat (sen tidak dipakai).
type ProductPrice struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	StoreID        string     `json:"store_id"`
	ProductID      string     `json:"product_id"`
	VariantID      *string    `json:"variant_id,omitempty"`
	PriceType      string     `json:"price_type"`
	Price          int64      `json:"price"`
	MinQuantity    int        `json:"min_quantity"`
	IsActive       bool       `json:"is_active"`
	ValidFrom      *time.Time `json:"valid_from,omitempty"`
	ValidUntil     *time.Time `json:"valid_until,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Repository adalah kontrak penyimpanan price.
// Semua query ter-scope org+store+product; cross-org ditutup sebagai NotFound.
type Repository interface {
	Create(ctx context.Context, p *ProductPrice) error
	FindByID(ctx context.Context, orgID, storeID, productID, priceID string) (*ProductPrice, error)
	FindByProduct(ctx context.Context, orgID, storeID, productID string, onlyActive bool) ([]ProductPrice, error)
	Update(ctx context.Context, p *ProductPrice) error
	Delete(ctx context.Context, orgID, storeID, productID, priceID string) error
	FindEffective(ctx context.Context, orgID, storeID, productID string, variantID *string, priceType string, qty int, now time.Time) (*ProductPrice, error)
}

// StoreChecker adalah port kepemilikan store untuk validasi same-org.
type StoreChecker interface {
	FindByIDInOrg(ctx context.Context, orgID, id string) (*stores.Store, error)
}

// ProductChecker adalah port kepemilikan product untuk validasi same-store.
type ProductChecker interface {
	FindByID(ctx context.Context, orgID, storeID, id string) (*products.Product, error)
}

// VariantChecker adalah port kepemilikan variant untuk validasi same-product.
// Dipenuhi struktural oleh variants repository (method FindByID).
type VariantChecker interface {
	FindByID(ctx context.Context, orgID, storeID, productID, id string) (*variants.ProductVariant, error)
}
