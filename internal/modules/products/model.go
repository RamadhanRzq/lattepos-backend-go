package products

import (
	"context"
	"errors"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Tipe product: MENU barang jual, RAW_MATERIAL bahan baku, PACKAGING, OTHER.
const (
	TypeMenu        = "MENU"
	TypeRawMaterial = "RAW_MATERIAL"
	TypePackaging   = "PACKAGING"
	TypeOther       = "OTHER"
)

// IsValidType melaporkan apakah typ salah satu tipe product yang dikenal.
func IsValidType(typ string) bool {
	switch typ {
	case TypeMenu, TypeRawMaterial, TypePackaging, TypeOther:
		return true
	default:
		return false
	}
}

// Error domain module products.
var (
	ErrNotFound           = errors.New("product not found")
	ErrSKUDuplicate       = errors.New("product SKU already exists in this store")
	ErrUnauthorized       = errors.New("unauthorized to access this product")
	ErrInvalidInput       = errors.New("product: invalid input")
	ErrInvalidPrice       = errors.New("product: price must be >= 0")
	ErrInvalidStock       = errors.New("product: stock must be >= 0")
	ErrInvalidProductType = errors.New("product: invalid product type")
	ErrStoreNotFound      = errors.New("product: store not found")
)

// Product adalah barang dagangan milik satu store dalam satu organization.
type Product struct {
	ID             string     `json:"id"`
	StoreID        string     `json:"store_id"`
	OrganizationID string     `json:"organization_id"`
	Name           string     `json:"name"`
	SKU            string     `json:"sku"`
	Description    string     `json:"description,omitempty"`
	ProductType    string     `json:"product_type"`
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
	Search      string
	CategoryID  *string
	ProductType string
	IsActive    *bool
	Page        int
	Limit       int
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

// VariantView adalah varian ringkas untuk detail product. DTO lokal supaya
// products tidak mengimpor module variants (variants mengimpor products).
type VariantView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	SKU      string `json:"sku"`
	Stock    int    `json:"stock"`
	IsActive bool   `json:"is_active"`
}

// PriceView adalah satu baris harga untuk detail product.
type PriceView struct {
	ID          string     `json:"id"`
	VariantID   *string    `json:"variant_id,omitempty"`
	PriceType   string     `json:"price_type"`
	Price       int64      `json:"price"`
	MinQuantity int        `json:"min_quantity"`
	IsActive    bool       `json:"is_active"`
	ValidFrom   *time.Time `json:"valid_from,omitempty"`
	ValidUntil  *time.Time `json:"valid_until,omitempty"`
}

// VariantStockView adalah stok live satu varian.
type VariantStockView struct {
	VariantID string `json:"variant_id"`
	Stock     int    `json:"stock"`
}

// StockView adalah ringkasan stok live product (kolom stock entity; riwayat
// movement tetap milik module stock).
type StockView struct {
	ProductStock int                `json:"product_stock"`
	Variants     []VariantStockView `json:"variants"`
}

// ProductDetail adalah response GET products/{id}: product + varian + harga +
// ringkasan stok live.
type ProductDetail struct {
	Product
	Variants []VariantView `json:"variants"`
	Prices   []PriceView   `json:"prices"`
	Stock    StockView     `json:"stock"`
}

// DetailSources adalah port pengaya detail product (varian + harga).
// Dipenuhi adapter composition root di atas variants/prices service;
// products tidak mengimpor module itu karena keduanya mengimpor products.
type DetailSources interface {
	ListVariants(ctx context.Context, orgID, storeID, productID string) ([]VariantView, error)
	ListPrices(ctx context.Context, orgID, storeID, productID string) ([]PriceView, error)
}
