package stock

import (
	"context"
	"errors"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Tipe movement valid.
const (
	TypeIn         = "in"
	TypeOut        = "out"
	TypeAdjustment = "adjustment"
	TypeReturn     = "return"
)

// Error domain module stock.
var (
	ErrInvalidInput    = errors.New("stock: invalid input")
	ErrInvalidType     = errors.New("stock: invalid movement type")
	ErrInsufficient    = errors.New("stock: insufficient stock")
	ErrProductNotFound = errors.New("stock: product not found")
	ErrVariantNotFound = errors.New("stock: variant not found")
	ErrStoreNotFound   = errors.New("stock: store not found")
	ErrNoAccess        = errors.New("stock: no store access")
)

// StockMovement adalah satu baris riwayat mutasi stok milik satu store.
// Stok live tetap di products.stock / product_variants.stock; movement hanya riwayat.
type StockMovement struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	StoreID        string    `json:"store_id"`
	ProductID      string    `json:"product_id"`
	VariantID      *string   `json:"variant_id,omitempty"`
	Type           string    `json:"type"`
	Quantity       int       `json:"quantity"`
	StockBefore    int       `json:"stock_before"`
	StockAfter     int       `json:"stock_after"`
	ReferenceType  string    `json:"reference_type,omitempty"`
	ReferenceID    string    `json:"reference_id,omitempty"`
	Notes          string    `json:"notes,omitempty"`
	CreatedBy      string    `json:"created_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// VariantStock adalah stok live satu varian.
type VariantStock struct {
	VariantID string `json:"variant_id"`
	Stock     int    `json:"stock"`
}

// StockSummary adalah stok live produk + variannya.
type StockSummary struct {
	ProductID    string         `json:"product_id"`
	ProductStock int            `json:"product_stock"`
	Variants     []VariantStock `json:"variants"`
}

// Filter adalah parameter query list movement dalam satu store.
type Filter struct {
	ProductID string
	Type      string
	From      time.Time
	To        time.Time
	Page      int
	Limit     int
}

// Repository adalah kontrak penyimpanan movement dan snapshot stok.
// Semua query ter-scope organization_id+store_id; cross-org ditutupi sebagai NotFound.
type Repository interface {
	// Record menulis satu movement sekaligus update stok live dalam satu tx.
	// Mengisi ID, StockBefore/After, dan CreatedAt ke struct yang diberikan.
	// Di dalam WithTx join tx pemanggil; standalone pakai tx sendiri.
	Record(ctx context.Context, m *StockMovement, allowNegative bool) error
	FindByStore(ctx context.Context, orgID, storeID string, filter Filter) ([]StockMovement, int, error)
	FindByProduct(ctx context.Context, orgID, storeID, productID string) ([]StockMovement, error)
	// FindByReference mengambil semua movement yang menunjuk satu dokumen
	// sumber (mis. seluruh konsumsi bahan satu sale) untuk dibalik saat cancel.
	FindByReference(ctx context.Context, orgID, storeID, referenceType, referenceID string) ([]StockMovement, error)
	GetSummary(ctx context.Context, orgID, storeID, productID string) (*StockSummary, error)
}

// StoreChecker adalah port kepemilikan dan akses store.
// Dipenuhi struktural oleh stores repository (FindByIDInOrg + IsAssigned).
type StoreChecker interface {
	FindByIDInOrg(ctx context.Context, orgID, id string) (*stores.Store, error)
	IsAssigned(ctx context.Context, userID, storeID string) (bool, error)
}

// IsValidType melaporkan apakah typ salah satu dari 4 tipe movement.
func IsValidType(typ string) bool {
	switch typ {
	case TypeIn, TypeOut, TypeAdjustment, TypeReturn:
		return true
	default:
		return false
	}
}
