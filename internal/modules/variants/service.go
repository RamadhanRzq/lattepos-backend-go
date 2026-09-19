package variants

import (
	"context"
	"errors"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Service menangani business rules variant dalam satu product+store.
type Service struct {
	repo     Repository
	stores   StoreChecker
	products ProductChecker
}

func NewService(repo Repository, stores StoreChecker, products ProductChecker) *Service {
	return &Service{repo: repo, stores: stores, products: products}
}

func (s *Service) Create(ctx context.Context, orgID, storeID, productID, name, sku string, stock int) (*ProductVariant, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	name = strings.TrimSpace(name)
	sku = strings.TrimSpace(sku)
	if orgID == "" || storeID == "" || productID == "" {
		return nil, ErrInvalidInput
	}
	if name == "" {
		return nil, ErrInvalidInput
	}
	if stock < 0 {
		return nil, ErrInvalidInput
	}

	if err := s.checkProduct(ctx, orgID, storeID, productID); err != nil {
		return nil, err
	}

	if sku != "" {
		dup, err := s.repo.ExistsBySKU(ctx, storeID, sku, "")
		if err != nil {
			return nil, err
		}
		if dup {
			return nil, ErrSKUExists
		}
	}

	v := &ProductVariant{
		OrganizationID: orgID,
		StoreID:        storeID,
		ProductID:      productID,
		Name:           name,
		SKU:            sku,
		Stock:          stock,
		IsActive:       true,
	}
	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) List(ctx context.Context, orgID, storeID, productID string) ([]ProductVariant, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	if orgID == "" || storeID == "" || productID == "" {
		return nil, ErrInvalidInput
	}
	if err := s.checkProduct(ctx, orgID, storeID, productID); err != nil {
		return nil, err
	}
	return s.repo.FindByProduct(ctx, orgID, storeID, productID)
}

func (s *Service) Update(ctx context.Context, orgID, storeID, productID, id, name, sku string, stock int, isActive bool) (*ProductVariant, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	sku = strings.TrimSpace(sku)
	if orgID == "" || storeID == "" || productID == "" || id == "" {
		return nil, ErrInvalidInput
	}
	if name == "" {
		return nil, ErrInvalidInput
	}
	if stock < 0 {
		return nil, ErrInvalidInput
	}

	if err := s.checkProduct(ctx, orgID, storeID, productID); err != nil {
		return nil, err
	}

	if sku != "" {
		dup, err := s.repo.ExistsBySKU(ctx, storeID, sku, id)
		if err != nil {
			return nil, err
		}
		if dup {
			return nil, ErrSKUExists
		}
	}

	v := &ProductVariant{
		ID:             id,
		OrganizationID: orgID,
		StoreID:        storeID,
		ProductID:      productID,
		Name:           name,
		SKU:            sku,
		Stock:          stock,
		IsActive:       isActive,
	}
	if err := s.repo.Update(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) Delete(ctx context.Context, orgID, storeID, productID, id string) error {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	id = strings.TrimSpace(id)
	if orgID == "" || storeID == "" || productID == "" || id == "" {
		return ErrInvalidInput
	}
	if err := s.checkProduct(ctx, orgID, storeID, productID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, orgID, storeID, productID, id)
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
