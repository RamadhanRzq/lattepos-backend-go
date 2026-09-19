package sales

import (
	"context"
	"errors"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/modules/prices"
	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Service menangani business rules sale dalam satu store.
// stores adalah port kepemilikan + akses store; products port keberadaan
// produk; prices port harga berlaku (fallback product.price); sideEffects
// hook post-create (stok + dapur), boleh nil.
type Service struct {
	repo        Repository
	stores      StoreChecker
	products    ProductChecker
	prices      PriceResolver
	sideEffects *SideEffects
}

func NewService(repo Repository, stores StoreChecker, products ProductChecker, prices PriceResolver, sideEffects *SideEffects) *Service {
	return &Service{repo: repo, stores: stores, products: products, prices: prices, sideEffects: sideEffects}
}

func (s *Service) Create(ctx context.Context, orgID, storeID, userID, paymentMethod string, discount, tax int64, notes string, lines []SaleLine) (*Sale, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	userID = strings.TrimSpace(userID)
	paymentMethod = strings.TrimSpace(paymentMethod)
	notes = strings.TrimSpace(notes)
	if orgID == "" || storeID == "" || userID == "" || paymentMethod == "" {
		return nil, ErrInvalidInput
	}
	if discount < 0 || tax < 0 {
		return nil, ErrInvalidInput
	}
	if len(lines) == 0 {
		return nil, ErrInvalidInput
	}
	for i := range lines {
		lines[i].ProductID = strings.TrimSpace(lines[i].ProductID)
		if lines[i].VariantID != nil && strings.TrimSpace(*lines[i].VariantID) == "" {
			lines[i].VariantID = nil
		}
		if lines[i].ProductID == "" || lines[i].Quantity <= 0 {
			return nil, ErrInvalidInput
		}
	}

	if err := s.authorize(ctx, orgID, storeID, userID); err != nil {
		return nil, err
	}

	// Snapshot harga dari price berlaku (fallback product.price) dan cek stok
	// live per line di server, bukan dari client.
	var total int64
	items := make([]SaleItem, len(lines))
	for i, ln := range lines {
		p, err := s.products.FindByID(ctx, orgID, storeID, ln.ProductID)
		if err != nil {
			if errors.Is(err, products.ErrNotFound) {
				return nil, ErrProductNotFound
			}
			return nil, err
		}
		unitPrice, err := s.unitPrice(ctx, orgID, storeID, ln, p)
		if err != nil {
			return nil, err
		}
		if ln.VariantID == nil && p.Stock < ln.Quantity {
			return nil, ErrInsufficientStock
		}
		sub := unitPrice * int64(ln.Quantity)
		total += sub
		items[i] = SaleItem{ProductID: ln.ProductID, VariantID: ln.VariantID, Quantity: ln.Quantity, UnitPrice: unitPrice, Subtotal: sub}
	}
	grand := total - discount + tax
	if grand < 0 {
		return nil, ErrInvalidInput
	}

	sale := &Sale{
		StoreID: storeID, OrganizationID: orgID, UserID: userID,
		TotalAmount: total, DiscountAmount: discount, TaxAmount: tax, GrandTotal: grand,
		PaymentMethod: paymentMethod, Status: StatusPending, Notes: notes, Items: items,
	}
	if err := s.repo.Create(ctx, sale); err != nil {
		return nil, err
	}
	s.fireSideEffects(ctx, sale, lines)
	return sale, nil
}

// unitPrice mengembalikan harga berlaku untuk satu line: prices.EffectivePrice
// bila port tersedia, fallback product.price bila port nil atau tidak ada
// baris harga yang cocok.
func (s *Service) unitPrice(ctx context.Context, orgID, storeID string, ln SaleLine, p *products.Product) (int64, error) {
	if s.prices == nil {
		return p.Price, nil
	}
	price, found, err := s.prices.EffectivePrice(ctx, orgID, storeID, ln.ProductID, ln.VariantID, "retail", ln.Quantity)
	if err != nil {
		// Harga tidak menemukan produk/variant di scope itu → biarkan fallback
		// hanya untuk product-line; error lain diteruskan.
		if ln.VariantID != nil && errors.Is(err, prices.ErrNotFound) {
			return 0, ErrProductNotFound
		}
		return 0, err
	}
	if !found {
		return p.Price, nil
	}
	return price, nil
}

// fireSideEffects menjalankan hook post-create secara best-effort:
// stock out per line (kegagalan stok tidak menggagalkan sale yang sudah
// tersimpan — dibiarkan sebagai riwayat) lalu pembukaan antrian dapur.
func (s *Service) fireSideEffects(ctx context.Context, sale *Sale, lines []SaleLine) {
	if s.sideEffects == nil {
		return
	}
	if s.sideEffects.StockOut != nil {
		for _, ln := range lines {
			_ = s.sideEffects.StockOut(ctx, sale.OrganizationID, sale.StoreID, sale.ID, ln, sale.UserID)
		}
	}
	if s.sideEffects.OpenKitchen != nil {
		_ = s.sideEffects.OpenKitchen(ctx, sale)
	}
}

func (s *Service) Get(ctx context.Context, orgID, storeID, userID, id string) (*Sale, error) {
	if err := s.authorize(ctx, orgID, storeID, userID); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.FindByID(ctx, orgID, storeID, id)
}

func (s *Service) List(ctx context.Context, orgID, storeID, userID string, filter Filter) ([]Sale, int, error) {
	if err := s.authorize(ctx, orgID, storeID, userID); err != nil {
		return nil, 0, err
	}
	filter.Status = strings.TrimSpace(filter.Status)
	switch filter.Status {
	case "", StatusPending, StatusCompleted, StatusCancelled:
	default:
		return nil, 0, ErrInvalidStatus
	}
	if !filter.From.IsZero() && !filter.To.IsZero() && filter.From.After(filter.To) {
		return nil, 0, ErrInvalidInput
	}
	return s.repo.FindByStore(ctx, orgID, storeID, filter)
}

// Cancel membatalkan sale; hanya status pending yang bisa dibatalkan.
// Setelah repo sukses, hook restock + penutupan antrian dapur dijalankan.
func (s *Service) Cancel(ctx context.Context, orgID, storeID, userID, id string) (*Sale, error) {
	if err := s.authorize(ctx, orgID, storeID, userID); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrInvalidInput
	}
	sale, err := s.repo.Cancel(ctx, orgID, storeID, id)
	if err != nil {
		return nil, err
	}
	if s.sideEffects != nil {
		if s.sideEffects.Restock != nil {
			if rerr := s.sideEffects.Restock(ctx, sale); rerr != nil {
				return sale, rerr
			}
		}
		if s.sideEffects.CancelKitchen != nil {
			_ = s.sideEffects.CancelKitchen(ctx, sale)
		}
	}
	return sale, nil
}

// authorize menegakkan tenant + store boundary: store milik org, user punya akses.
func (s *Service) authorize(ctx context.Context, orgID, storeID, userID string) error {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	userID = strings.TrimSpace(userID)
	if orgID == "" || storeID == "" || userID == "" {
		return ErrInvalidInput
	}
	if _, err := s.stores.FindByIDInOrg(ctx, orgID, storeID); err != nil {
		if errors.Is(err, stores.ErrNotFound) {
			return ErrStoreNotFound
		}
		return err
	}
	ok, err := s.stores.IsAssigned(ctx, userID, storeID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoAccess
	}
	return nil
}
