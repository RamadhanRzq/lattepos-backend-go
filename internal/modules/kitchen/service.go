package kitchen

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/sales"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Service menangani business rules antrian dapur dalam satu store.
type Service struct {
	repo   Repository
	stores StoreChecker
	sales  SaleChecker
}

func NewService(repo Repository, stores StoreChecker, sales SaleChecker) *Service {
	return &Service{repo: repo, stores: stores, sales: sales}
}

// Transisi legal per status; served/cancelled terminal.
var saleTransitions = map[string][]string{
	StatusPending:   {StatusPreparing, StatusCancelled},
	StatusPreparing: {StatusReady, StatusCancelled},
	StatusReady:     {StatusServed, StatusCancelled},
	StatusServed:    {},
	StatusCancelled: {},
}

var itemTransitions = map[string][]string{
	StatusPending:   {StatusPreparing, StatusCancelled},
	StatusPreparing: {StatusReady, StatusCancelled},
	StatusReady:     {},
	StatusCancelled: {},
}

func validStatus(s string, table map[string][]string) bool {
	_, ok := table[s]
	return ok
}

func allowedTransition(cur, next string, table map[string][]string) bool {
	for _, n := range table[cur] {
		if n == next {
			return true
		}
	}
	return false
}

// CreateForSale membuka antrian dapur untuk satu sale; dipanggil parent
// (sales) setelah sale dibuat. Semua item mulai pending.
func (s *Service) CreateForSale(ctx context.Context, orgID, storeID, saleID, userID, notes string, items []SaleItemRef) (*KitchenSale, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	saleID = strings.TrimSpace(saleID)
	userID = strings.TrimSpace(userID)
	notes = strings.TrimSpace(notes)
	if orgID == "" || storeID == "" || saleID == "" || userID == "" {
		return nil, ErrInvalidInput
	}
	if len(items) == 0 {
		return nil, ErrInvalidInput
	}
	if err := s.authorize(ctx, orgID, storeID, userID); err != nil {
		return nil, err
	}

	sale, err := s.sales.FindByID(ctx, orgID, storeID, saleID)
	if err != nil {
		if errors.Is(err, sales.ErrNotFound) {
			return nil, ErrSaleNotFound
		}
		return nil, err
	}
	if sale.Status == sales.StatusCancelled {
		return nil, ErrSaleCancelled
	}

	ks := &KitchenSale{
		SaleID:         saleID,
		OrganizationID: orgID,
		StoreID:        storeID,
		Status:         StatusPending,
		Notes:          notes,
		Items:          make([]KitchenSaleItem, len(items)),
	}
	for i, ref := range items {
		ref.SaleItemID = strings.TrimSpace(ref.SaleItemID)
		ref.ProductID = strings.TrimSpace(ref.ProductID)
		if ref.SaleItemID == "" || ref.ProductID == "" || ref.Quantity <= 0 {
			return nil, ErrInvalidInput
		}
		ks.Items[i] = KitchenSaleItem{
			SaleItemID: ref.SaleItemID,
			ProductID:  ref.ProductID,
			VariantID:  ref.VariantID,
			Quantity:   ref.Quantity,
			Status:     StatusPending,
			Notes:      strings.TrimSpace(ref.Notes),
		}
	}
	if err := s.repo.Create(ctx, ks); err != nil {
		return nil, err
	}
	return ks, nil
}

// Queue mengembalikan antrian aktif (pending, preparing) terurut prioritas.
func (s *Service) Queue(ctx context.Context, orgID, storeID, userID string) ([]KitchenSale, error) {
	if err := s.authorize(ctx, orgID, storeID, userID); err != nil {
		return nil, err
	}
	return s.repo.FindQueue(ctx, strings.TrimSpace(orgID), strings.TrimSpace(storeID))
}

// Get mengambil satu antrian beserta itemnya.
func (s *Service) Get(ctx context.Context, orgID, storeID, userID, id string) (*KitchenSale, error) {
	if err := s.authorize(ctx, orgID, storeID, userID); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.FindByID(ctx, strings.TrimSpace(orgID), strings.TrimSpace(storeID), id)
}

// UpdateStatus memajukan status antrian sesuai mesin transisi.
func (s *Service) UpdateStatus(ctx context.Context, orgID, storeID, userID, id, status string) (*KitchenSale, error) {
	if err := s.authorize(ctx, orgID, storeID, userID); err != nil {
		return nil, err
	}
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	id = strings.TrimSpace(id)
	status = strings.TrimSpace(status)
	if id == "" {
		return nil, ErrInvalidInput
	}
	if !validStatus(status, saleTransitions) {
		return nil, ErrInvalidStatus
	}

	cur, err := s.repo.FindByID(ctx, orgID, storeID, id)
	if err != nil {
		return nil, err
	}
	if !allowedTransition(cur.Status, status, saleTransitions) {
		return nil, ErrInvalidTransition
	}
	if err := s.guardSaleOpen(ctx, orgID, storeID, cur.SaleID); err != nil {
		return nil, err
	}

	var started, completed *time.Time
	now := time.Now()
	switch status {
	case StatusPreparing:
		if cur.StartedAt == nil {
			started = &now
		}
	case StatusServed, StatusCancelled:
		completed = &now
	}
	return s.repo.UpdateStatus(ctx, orgID, storeID, id, status, started, completed)
}

// UpdateItemStatus memajukan satu item; bila semua item ready, antrian
// ikut naik ke ready otomatis.
func (s *Service) UpdateItemStatus(ctx context.Context, orgID, storeID, userID, kitchenID, itemID, status string) (*KitchenSaleItem, error) {
	if err := s.authorize(ctx, orgID, storeID, userID); err != nil {
		return nil, err
	}
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	kitchenID = strings.TrimSpace(kitchenID)
	itemID = strings.TrimSpace(itemID)
	status = strings.TrimSpace(status)
	if kitchenID == "" || itemID == "" {
		return nil, ErrInvalidInput
	}
	if !validStatus(status, itemTransitions) {
		return nil, ErrInvalidStatus
	}

	cur, err := s.repo.FindByID(ctx, orgID, storeID, kitchenID)
	if err != nil {
		return nil, err
	}
	var curItem *KitchenSaleItem
	for i := range cur.Items {
		if cur.Items[i].ID == itemID {
			curItem = &cur.Items[i]
			break
		}
	}
	if curItem == nil {
		return nil, ErrNotFound
	}
	if !allowedTransition(curItem.Status, status, itemTransitions) {
		return nil, ErrInvalidTransition
	}
	if err := s.guardSaleOpen(ctx, orgID, storeID, cur.SaleID); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateItemStatus(ctx, orgID, storeID, kitchenID, itemID, status)
	if err != nil {
		return nil, err
	}

	// Auto-fire: semua item ready dan antrian masih preparing → naik ke ready.
	items, err := s.repo.FindItems(ctx, kitchenID)
	if err != nil {
		return updated, nil
	}
	allReady := len(items) > 0
	for _, it := range items {
		if it.Status != StatusReady {
			allReady = false
			break
		}
	}
	if allReady && cur.Status == StatusPreparing {
		if _, err := s.repo.UpdateStatus(ctx, orgID, storeID, kitchenID, StatusReady, nil, nil); err != nil {
			if !errors.Is(err, ErrNotFound) {
				return nil, err
			}
		}
	}
	return updated, nil
}

// guardSaleOpen menolak mutasi bila sale induk sudah cancelled.
func (s *Service) guardSaleOpen(ctx context.Context, orgID, storeID, saleID string) error {
	sale, err := s.sales.FindByID(ctx, orgID, storeID, saleID)
	if err != nil {
		if errors.Is(err, sales.ErrNotFound) {
			return ErrSaleNotFound
		}
		return err
	}
	if sale.Status == sales.StatusCancelled {
		return ErrSaleCancelled
	}
	return nil
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

// CancelForSale menutup antrian dapur milik sale yang dibatalkan parent
// (dipanggil composition root dari hook pembatalan sale). Idempoten:
// antrian tidak ada → nil; antrian served tetap served (tidak ditimpa).
func (s *Service) CancelForSale(ctx context.Context, orgID, storeID, saleID string) error {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	saleID = strings.TrimSpace(saleID)
	if orgID == "" || storeID == "" || saleID == "" {
		return ErrInvalidInput
	}
	_, err := s.repo.FindBySaleID(ctx, saleID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	if _, err := s.repo.CancelBySale(ctx, orgID, storeID, saleID); err != nil {
		return err
	}
	return nil
}
