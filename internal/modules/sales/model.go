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
// ErrInsufficientStock = stok tidak cukup untuk transaksi (diterjemahkan
// adapter composition root dari stock.ErrInsufficient).
var (
	ErrNotFound          = errors.New("sale: not found")
	ErrInvalidInput      = errors.New("sale: invalid input")
	ErrInvalidStatus     = errors.New("sale: invalid status")
	ErrNoAccess          = errors.New("sale: no store access")
	ErrStoreNotFound     = errors.New("sale: store not found")
	ErrProductNotFound   = errors.New("sale: product not found")
	ErrInsufficientStock = errors.New("sale: insufficient stock")
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
// UnitPrice snapshot dari price berlaku (product/variant) saat transaksi,
// bukan FK live. VariantID opsional: nullable FK yang ditulis repository.
type SaleItem struct {
	ID        string    `json:"id"`
	SaleID    string    `json:"sale_id"`
	ProductID string    `json:"product_id"`
	VariantID *string   `json:"variant_id,omitempty"`
	Quantity  int       `json:"quantity"`
	UnitPrice int64     `json:"unit_price"`
	Subtotal  int64     `json:"subtotal"`
	CreatedAt time.Time `json:"created_at"`
}

// SaleLine adalah input satu baris saat buat transaksi.
type SaleLine struct {
	ProductID string
	VariantID *string
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

// ProductChecker adalah port baca produk untuk validasi keberadaan + stok live.
// Dipenuhi struktural oleh products repository (method FindByID).
type ProductChecker interface {
	FindByID(ctx context.Context, orgID, storeID, id string) (*products.Product, error)
}

// PriceResolver adalah port harga berlaku; dipenuhi struktural oleh
// prices.Service (method EffectivePrice). Fallback product.price dipakai
// bila nil (port opsional) atau tidak ada baris harga yang cocok.
type PriceResolver interface {
	EffectivePrice(ctx context.Context, orgID, storeID, productID string, variantID *string, priceType string, qty int) (int64, bool, error)
}

// SideEffects adalah hook post-create dan post-cancel sale: mutasi stok
// otomatis + pembukaan/penutupan antrian dapur. Dipenuhi adapter composition
// root di atas module stock dan kitchen supaya sales tidak mengimpor keduanya
// (patOK stores.OrgChecker); nil berarti efek samping dimatikan.
type SideEffects struct {
	// StockOut dipanggil per line saat sale dibuat; return err diterjemahkan
	// ke ErrInsufficientStock bila sesuai (lihat adapter di main.go).
	StockOut func(ctx context.Context, orgID, storeID, saleID string, line SaleLine, createdBy string) error
	// OpenKitchen dipanggil sekali setelah sale dibuat; gagal tidak
	// menggagalkan sale (log adapter, best-effort).
	OpenKitchen func(ctx context.Context, sale *Sale) error
	// Restock dipanggil sekali setelah sale dibatalkan: kembalikan stok semua
	// line (item sudah dimuat repo.Cancel). Gagal → Cancel tetap sukses,
	// error diteruskan ke caller untuk ditindaklanjuti.
	Restock func(ctx context.Context, sale *Sale) error
	// CancelKitchen menutup antrian dapur milik sale yang dibatalkan.
	// Idempoten di adapter: antrian tidak ada → nil.
	CancelKitchen func(ctx context.Context, sale *Sale) error
}
