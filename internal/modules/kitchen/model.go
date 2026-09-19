package kitchen

import (
	"context"
	"errors"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/sales"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Status valid kitchen sale.
const (
	StatusPending   = "pending"
	StatusPreparing = "preparing"
	StatusReady     = "ready"
	StatusServed    = "served"
	StatusCancelled = "cancelled"
)

// Error domain module kitchen.
var (
	ErrNotFound          = errors.New("kitchen: not found")
	ErrInvalidInput      = errors.New("kitchen: invalid input")
	ErrInvalidStatus     = errors.New("kitchen: invalid status")
	ErrInvalidTransition = errors.New("kitchen: invalid status transition")
	ErrSaleNotFound      = errors.New("kitchen: sale not found")
	ErrSaleCancelled     = errors.New("kitchen: sale cancelled")
	ErrStoreNotFound     = errors.New("kitchen: store not found")
	ErrNoAccess          = errors.New("kitchen: no store access")
)

// KitchenSale adalah antrian dapur untuk satu sale dalam satu store.
// Satu sale hanya punya satu baris (unique_kitchen_sale).
type KitchenSale struct {
	ID             string            `json:"id"`
	SaleID         string            `json:"sale_id"`
	OrganizationID string            `json:"organization_id"`
	StoreID        string            `json:"store_id"`
	Status         string            `json:"status"`
	Priority       int               `json:"priority"`
	Notes          string            `json:"notes,omitempty"`
	StartedAt      *time.Time        `json:"started_at,omitempty"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty"`
	Items          []KitchenSaleItem `json:"items,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// KitchenSaleItem adalah satu baris antrian dapur, merujuk satu sale_items.
type KitchenSaleItem struct {
	ID            string    `json:"id"`
	KitchenSaleID string    `json:"kitchen_sale_id"`
	SaleItemID    string    `json:"sale_item_id"`
	ProductID     string    `json:"product_id"`
	VariantID     *string   `json:"variant_id,omitempty"`
	Quantity      int       `json:"quantity"`
	Status        string    `json:"status"`
	Notes         string    `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SaleItemRef adalah input satu baris saat buat antrian dari sale.
type SaleItemRef struct {
	SaleItemID string
	ProductID  string
	VariantID  *string
	Quantity   int
	Notes      string
}

// Repository adalah kontrak penyimpanan kitchen sale dan itemnya.
// Semua query ter-scope store_id; orgID dipakai menutupi cross-org sebagai NotFound.
type Repository interface {
	// Create menyimpan kitchen sale beserta Items-nya dalam satu transaksi.
	// Mengisi ID dan timestamp hasil insert ke struct yang diberikan.
	Create(ctx context.Context, ks *KitchenSale) error
	FindByID(ctx context.Context, orgID, storeID, id string) (*KitchenSale, error)
	FindQueue(ctx context.Context, orgID, storeID string) ([]KitchenSale, error)
	FindBySaleID(ctx context.Context, saleID string) (*KitchenSale, error)
	// CancelBySale menutup antrian aktif milik satu sale: antrian dan item
	// yang masih pending/preparing jadi cancelled. Mengembalikan jumlah
	// antrian yang ter-update (0 = tidak ada antrian aktif).
	CancelBySale(ctx context.Context, orgID, storeID, saleID string) (int64, error)
	UpdateStatus(ctx context.Context, orgID, storeID, id, status string, started, completed *time.Time) (*KitchenSale, error)
	UpdateItemStatus(ctx context.Context, orgID, storeID, kitchenID, itemID, status string) (*KitchenSaleItem, error)
	FindItems(ctx context.Context, kitchenID string) ([]KitchenSaleItem, error)
}

// StoreChecker adalah port kepemilikan dan akses store.
// Dipenuhi struktural oleh stores repository (FindByIDInOrg + IsAssigned).
type StoreChecker interface {
	FindByIDInOrg(ctx context.Context, orgID, id string) (*stores.Store, error)
	IsAssigned(ctx context.Context, userID, storeID string) (bool, error)
}

// SaleChecker adalah port baca sale induk antrian.
// Dipenuhi struktural oleh sales repository (method FindByID).
type SaleChecker interface {
	FindByID(ctx context.Context, orgID, storeID, id string) (*sales.Sale, error)
}
