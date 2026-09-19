package categories

import (
	"context"
	"errors"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Error domain module categories.
var (
	ErrNotFound      = errors.New("category not found")
	ErrSlugExists    = errors.New("category slug already exists in this store")
	ErrInvalidInput  = errors.New("category: invalid input")
	ErrInvalidParent = errors.New("category: invalid parent")
	ErrInUse         = errors.New("category: still used by products")
)

// Category adalah pengelompokan product dalam satu store.
type Category struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	StoreID        string    `json:"store_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    string    `json:"description,omitempty"`
	ParentID       *string   `json:"parent_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Repository adalah kontrak penyimpanan category.
// Semua query ter-scope org+store; cross-org ditutupi sebagai NotFound.
type Repository interface {
	Create(ctx context.Context, c *Category) error
	FindByID(ctx context.Context, orgID, storeID, id string) (*Category, error)
	FindBySlug(ctx context.Context, orgID, storeID, slug string) (*Category, error)
	FindByStore(ctx context.Context, orgID, storeID string) ([]Category, error)
	Update(ctx context.Context, c *Category) error
	Delete(ctx context.Context, orgID, storeID, id string) error
	ExistsBySlug(ctx context.Context, slug, storeID string, excludeID *string) (bool, error)
	CountProducts(ctx context.Context, storeID, categoryID string) (int, error)
}

// StoreChecker adalah port kepemilikan store untuk validasi same-org.
// Dipenuhi struktural oleh stores repository (method FindByIDInOrg).
type StoreChecker interface {
	FindByIDInOrg(ctx context.Context, orgID, id string) (*stores.Store, error)
}
