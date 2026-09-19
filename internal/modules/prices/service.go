package prices

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	"github.com/ramadhanrzq/backend-go/internal/modules/variants"
)

// Service menangani business rules price dalam satu product+store.
type Service struct {
	repo     Repository
	stores   StoreChecker
	products ProductChecker
	variants VariantChecker
}

func NewService(repo Repository, stores StoreChecker, products ProductChecker, variants VariantChecker) *Service {
	return &Service{repo: repo, stores: stores, products: products, variants: variants}
}

func (s *Service) Create(ctx context.Context, orgID, storeID, productID string, variantID *string, priceType string, price int64, minQty int, validFrom, validUntil *time.Time) (*ProductPrice, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	priceType = strings.TrimSpace(priceType)
	variantID = trimOptID(variantID)
	if orgID == "" || storeID == "" || productID == "" {
		return nil, ErrInvalidInput
	}
	if priceType == "" {
		priceType = "retail"
	}
	if price < 0 {
		return nil, ErrInvalidInput
	}
	if minQty < 1 {
		return nil, ErrInvalidInput
	}
	if validFrom != nil && validUntil != nil && validUntil.Before(*validFrom) {
		return nil, ErrInvalidInput
	}

	if err := s.checkProduct(ctx, orgID, storeID, productID); err != nil {
		return nil, err
	}
	if err := s.checkVariant(ctx, orgID, storeID, productID, variantID); err != nil {
		return nil, err
	}

	p := &ProductPrice{
		OrganizationID: orgID,
		StoreID:        storeID,
		ProductID:      productID,
		VariantID:      variantID,
		PriceType:      priceType,
		Price:          price,
		MinQuantity:    minQty,
		IsActive:       true,
		ValidFrom:      validFrom,
		ValidUntil:     validUntil,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) List(ctx context.Context, orgID, storeID, productID string, onlyActive bool) ([]ProductPrice, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	if orgID == "" || storeID == "" || productID == "" {
		return nil, ErrInvalidInput
	}
	if err := s.checkProduct(ctx, orgID, storeID, productID); err != nil {
		return nil, err
	}
	return s.repo.FindByProduct(ctx, orgID, storeID, productID, onlyActive)
}

func (s *Service) Update(ctx context.Context, orgID, storeID, productID, priceID string, variantID *string, priceType string, price int64, minQty int, isActive bool, validFrom, validUntil *time.Time) (*ProductPrice, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	priceID = strings.TrimSpace(priceID)
	priceType = strings.TrimSpace(priceType)
	variantID = trimOptID(variantID)
	if orgID == "" || storeID == "" || productID == "" || priceID == "" {
		return nil, ErrInvalidInput
	}
	if priceType == "" {
		priceType = "retail"
	}
	if price < 0 {
		return nil, ErrInvalidInput
	}
	if minQty < 1 {
		return nil, ErrInvalidInput
	}
	if validFrom != nil && validUntil != nil && validUntil.Before(*validFrom) {
		return nil, ErrInvalidInput
	}

	if err := s.checkProduct(ctx, orgID, storeID, productID); err != nil {
		return nil, err
	}
	if err := s.checkVariant(ctx, orgID, storeID, productID, variantID); err != nil {
		return nil, err
	}

	p := &ProductPrice{
		ID:             priceID,
		OrganizationID: orgID,
		StoreID:        storeID,
		ProductID:      productID,
		VariantID:      variantID,
		PriceType:      priceType,
		Price:          price,
		MinQuantity:    minQty,
		IsActive:       isActive,
		ValidFrom:      validFrom,
		ValidUntil:     validUntil,
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Delete(ctx context.Context, orgID, storeID, productID, priceID string) error {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	priceID = strings.TrimSpace(priceID)
	if orgID == "" || storeID == "" || productID == "" || priceID == "" {
		return ErrInvalidInput
	}
	if err := s.checkProduct(ctx, orgID, storeID, productID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, orgID, storeID, productID, priceID)
}

// EffectivePrice mengembalikan harga berlaku; found=false bila tidak ada baris cocok.
func (s *Service) EffectivePrice(ctx context.Context, orgID, storeID, productID string, variantID *string, priceType string, qty int) (int64, bool, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	priceType = strings.TrimSpace(priceType)
	variantID = trimOptID(variantID)
	if orgID == "" || storeID == "" || productID == "" || priceType == "" || qty < 1 {
		return 0, false, ErrInvalidInput
	}
	p, err := s.repo.FindEffective(ctx, orgID, storeID, productID, variantID, priceType, qty, time.Now())
	if err != nil {
		return 0, false, err
	}
	if p == nil {
		return 0, false, nil
	}
	return p.Price, true, nil
}

// checkProduct menegakkan tenant boundary: store milik org, product milik store.
func (s *Service) checkProduct(ctx context.Context, orgID, storeID, productID string) error {
	if _, err := s.stores.FindByIDInOrg(ctx, orgID, storeID); err != nil {
		if errors.Is(err, stores.ErrNotFound) {
			return ErrStoreNotFound
		}
		return err
	}
	if _, err := s.products.FindByID(ctx, orgID, storeID, productID); err != nil {
		if errors.Is(err, products.ErrNotFound) {
			return ErrProductNotFound
		}
		return err
	}
	return nil
}

// checkVariant memastikan variant milik product yang sama.
func (s *Service) checkVariant(ctx context.Context, orgID, storeID, productID string, variantID *string) error {
	if variantID == nil || *variantID == "" {
		return nil
	}
	v, err := s.variants.FindByID(ctx, orgID, storeID, productID, *variantID)
	if err != nil {
		if errors.Is(err, variants.ErrNotFound) {
			return ErrVariantNotFound
		}
		return err
	}
	if v.ProductID != productID {
		return ErrInvalidInput
	}
	return nil
}

func trimOptID(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}
