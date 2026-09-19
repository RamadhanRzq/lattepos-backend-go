package stock

import (
	"context"
	"errors"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Service menangani business rules movement dalam satu store.
type Service struct {
	repo   Repository
	stores StoreChecker
}

func NewService(repo Repository, stores StoreChecker) *Service {
	return &Service{repo: repo, stores: stores}
}

// Record mencatat satu movement; createdBy = userID pencatat (diisi handler dari claims).
func (s *Service) Record(ctx context.Context, orgID, storeID, productID string, variantID *string, typ string, qty int, refType, refID, notes, createdBy string, allowNegative bool) (*StockMovement, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	typ = strings.TrimSpace(typ)
	refType = strings.TrimSpace(refType)
	refID = strings.TrimSpace(refID)
	notes = strings.TrimSpace(notes)
	createdBy = strings.TrimSpace(createdBy)
	var variant *string
	if variantID != nil && strings.TrimSpace(*variantID) != "" {
		v := strings.TrimSpace(*variantID)
		variant = &v
	}
	if orgID == "" || storeID == "" || productID == "" || createdBy == "" {
		return nil, ErrInvalidInput
	}
	if !IsValidType(typ) {
		return nil, ErrInvalidType
	}
	if qty <= 0 {
		return nil, ErrInvalidInput
	}
	if err := s.authorize(ctx, orgID, storeID, createdBy); err != nil {
		return nil, err
	}

	m := &StockMovement{
		OrganizationID: orgID,
		StoreID:        storeID,
		ProductID:      productID,
		VariantID:      variant,
		Type:           typ,
		Quantity:       qty,
		ReferenceType:  refType,
		ReferenceID:    refID,
		Notes:          notes,
		CreatedBy:      createdBy,
	}
	if err := s.repo.Record(ctx, m, allowNegative); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Service) List(ctx context.Context, orgID, storeID, userID string, filter Filter) ([]StockMovement, int, error) {
	if err := s.authorize(ctx, orgID, storeID, userID); err != nil {
		return nil, 0, err
	}
	filter.ProductID = strings.TrimSpace(filter.ProductID)
	filter.Type = strings.TrimSpace(filter.Type)
	if filter.Type != "" && !IsValidType(filter.Type) {
		return nil, 0, ErrInvalidType
	}
	if !filter.From.IsZero() && !filter.To.IsZero() && filter.From.After(filter.To) {
		return nil, 0, ErrInvalidInput
	}
	return s.repo.FindByStore(ctx, orgID, storeID, filter)
}

// StockByProduct mengembalikan ringkasan stok live + riwayat movement produk.
func (s *Service) StockByProduct(ctx context.Context, orgID, storeID, userID, productID string) (*StockSummary, []StockMovement, error) {
	if err := s.authorize(ctx, orgID, storeID, userID); err != nil {
		return nil, nil, err
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, nil, ErrInvalidInput
	}
	sum, err := s.repo.GetSummary(ctx, orgID, storeID, productID)
	if err != nil {
		return nil, nil, err
	}
	history, err := s.repo.FindByProduct(ctx, orgID, storeID, productID)
	if err != nil {
		return nil, nil, err
	}
	if history == nil {
		history = []StockMovement{}
	}
	return sum, history, nil
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
