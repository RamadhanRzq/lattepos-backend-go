package products

import (
	"context"
	"errors"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Error domain module products.
var (
	ErrNotFound      = errors.New("product not found")
	ErrSKUDuplicate  = errors.New("product SKU already exists in this store")
	ErrUnauthorized  = errors.New("unauthorized to access this product")
	ErrInvalidInput  = errors.New("product: invalid input")
	ErrInvalidPrice  = errors.New("product: price must be >= 0")
	ErrInvalidStock  = errors.New("product: stock must be >= 0")
	ErrStoreNotFound = errors.New("product: store not found")
)

// Product adalah barang dagangan milik satu store dalam satu organization.
type Product struct {
	ID             string     `json:"id"`
	StoreID        string     `json:"store_id"`
	OrganizationID string     `json:"organization_id"`
	Name           string     `json:"name"`
	SKU            string     `json:"sku"`
	Description    string     `json:"description,omitempty"`
	Price          int64      `json:"price"`
	Stock          int        `json:"stock"`
	Unit           string     `json:"unit"`
	CategoryID     *string    `json:"category_id,omitempty"`
	ImageURL       string     `json:"image_url,omitempty"`
	IsActive       bool       `json:"is_active"`
	CreatedBy      string     `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// Filter adalah parameter query list product dalam satu store.
type Filter struct {
	Search     string
	CategoryID *string
	IsActive   *bool
	Page       int
	Limit      int
}

// Repository adalah kontrak penyimpanan product.
// Semua query ter-scope store_id; orgID dipakai menutupi cross-org sebagai NotFound.
type Repository interface {
	Create(ctx context.Context, p *Product) error
	FindByID(ctx context.Context, orgID, storeID, id string) (*Product, error)
	FindByStore(ctx context.Context, orgID, storeID string, filter Filter) ([]Product, int, error)
	Update(ctx context.Context, p *Product) error
	SoftDelete(ctx context.Context, orgID, storeID, id string) error
	ExistsBySKU(ctx context.Context, sku, storeID string, excludeID *string) (bool, error)
}

// StoreChecker adalah port kepemilikan store untuk validasi same-org.
// Dipenuhi struktural oleh stores repository (method FindByIDInOrg).
type StoreChecker interface {
	FindByIDInOrg(ctx context.Context, orgID, id string) (*stores.Store, error)
}
