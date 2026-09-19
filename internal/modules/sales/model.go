package sales

import (
	"context"
	"errors"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Status valid sale.
const (
	StatusPending   = "pending"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
)

// Error domain module sales.
var (
	ErrNotFound        = errors.New("sale: not found")
	ErrInvalidInput    = errors.New("sale: invalid input")
	ErrInvalidStatus   = errors.New("sale: invalid status")
	ErrNoAccess        = errors.New("sale: no store access")
	ErrStoreNotFound   = errors.New("sale: store not found")
	ErrProductNotFound = errors.New("sale: product not found")
)

// Sale adalah transaksi milik satu store dalam satu organization.
// Total dihitung service dari snapshot harga item, bukan dari client.
type Sale struct {
	ID             string     `json:"id"`
	StoreID        string     `json:"store_id"`
	OrganizationID string     `json:"organization_id"`
	UserID         string     `json:"user_id"`
	TotalAmount    int64      `json:"total_amount"`
	DiscountAmount int64      `json:"discount_amount"`
	TaxAmount      int64      `json:"tax_amount"`
	GrandTotal     int64      `json:"grand_total"`
	PaymentMethod  string     `json:"payment_method"`
	Status         string     `json:"status"`
	Notes          string     `json:"notes,omitempty"`
	Items          []SaleItem `json:"items,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// SaleItem adalah satu baris transaksi.
// UnitPrice snapshot dari product.price saat transaksi, bukan FK live.
type SaleItem struct {
	ID        string    `json:"id"`
	SaleID    string    `json:"sale_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	UnitPrice int64     `json:"unit_price"`
	Subtotal  int64     `json:"subtotal"`
	CreatedAt time.Time `json:"created_at"`
}

// SaleLine adalah input satu baris saat buat transaksi.
type SaleLine struct {
	ProductID string
	Quantity  int
}

// Filter adalah parameter query list sale dalam satu store.
type Filter struct {
	Status string
	From   time.Time
	To     time.Time
	Page   int
	Limit  int
}

// Repository adalah kontrak penyimpanan sale dan itemnya.
// Semua query ter-scope store_id; orgID dipakai menutupi cross-org sebagai NotFound.
type Repository interface {
	// Create menyimpan sale beserta Items-nya dalam satu transaksi.
	// Mengisi ID dan timestamp hasil insert ke struct yang diberikan.
	Create(ctx context.Context, s *Sale) error
	FindByID(ctx context.Context, orgID, storeID, id string) (*Sale, error)
	FindItems(ctx context.Context, saleID string) ([]SaleItem, error)
	FindByStore(ctx context.Context, orgID, storeID string, filter Filter) ([]Sale, int, error)
	Cancel(ctx context.Context, orgID, storeID, id string) (*Sale, error)
}

// StoreChecker adalah port kepemilikan dan akses store.
// Dipenuhi struktural oleh stores repository (FindByIDInOrg + IsAssigned).
type StoreChecker interface {
	FindByIDInOrg(ctx context.Context, orgID, id string) (*stores.Store, error)
	IsAssigned(ctx context.Context, userID, storeID string) (bool, error)
}

// ProductChecker adalah port harga produk untuk snapshot unit_price.
// Dipenuhi struktural oleh products repository (method FindByID).
type ProductChecker interface {
	FindByID(ctx context.Context, orgID, storeID, id string) (*products.Product, error)
}
