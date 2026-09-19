package sales

import (
	"context"
	"errors"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Service menangani business rules sale dalam satu store.
// stores adalah port kepemilikan + akses store; prices port snapshot harga produk.
type Service struct {
	repo     Repository
	stores   StoreChecker
	products ProductChecker
}

func NewService(repo Repository, stores StoreChecker, products ProductChecker) *Service {
	return &Service{repo: repo, stores: stores, products: products}
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
		if lines[i].ProductID == "" || lines[i].Quantity <= 0 {
			return nil, ErrInvalidInput
		}
	}

	if err := s.authorize(ctx, orgID, storeID, userID); err != nil {
		return nil, err
	}

	// Snapshot harga dari product dan hitung total di server, bukan dari client.
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
		sub := p.Price * int64(ln.Quantity)
		total += sub
		items[i] = SaleItem{ProductID: ln.ProductID, Quantity: ln.Quantity, UnitPrice: p.Price, Subtotal: sub}
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
	return sale, nil
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
func (s *Service) Cancel(ctx context.Context, orgID, storeID, userID, id string) (*Sale, error) {
	if err := s.authorize(ctx, orgID, storeID, userID); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.Cancel(ctx, orgID, storeID, id)
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
