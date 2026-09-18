package products

import (
	"context"
	"errors"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Service menangani business rules product dalam satu store.
// storeChecker adalah port kepemilikan store untuk validasi same-org.
type Service struct {
	repo   Repository
	stores StoreChecker
}

func NewService(repo Repository, stores StoreChecker) *Service {
	return &Service{repo: repo, stores: stores}
}

func (s *Service) Create(ctx context.Context, orgID, storeID, createdBy, name, sku, description string, price int64, stock int, unit string, categoryID *string, imageURL string) (*Product, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	name = strings.TrimSpace(name)
	sku = strings.TrimSpace(sku)
	description = strings.TrimSpace(description)
	unit = strings.TrimSpace(unit)
	imageURL = strings.TrimSpace(imageURL)
	if orgID == "" || storeID == "" {
		return nil, ErrInvalidInput
	}
	if name == "" || sku == "" {
		return nil, ErrInvalidInput
	}
	if price < 0 {
		return nil, ErrInvalidPrice
	}
	if stock < 0 {
		return nil, ErrInvalidStock
	}
	if unit == "" {
		unit = "pcs"
	}

	if _, err := s.stores.FindByIDInOrg(ctx, orgID, storeID); err != nil {
		if errors.Is(err, stores.ErrNotFound) {
			return nil, ErrStoreNotFound
		}
		return nil, err
	}

	dup, err := s.repo.ExistsBySKU(ctx, sku, storeID, nil)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, ErrSKUDuplicate
	}

	p := &Product{
		StoreID:        storeID,
		OrganizationID: orgID,
		Name:           name,
		SKU:            sku,
		Description:    description,
		Price:          price,
		Stock:          stock,
		Unit:           unit,
		CategoryID:     categoryID,
		ImageURL:       imageURL,
		IsActive:       true,
		CreatedBy:      strings.TrimSpace(createdBy),
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Get(ctx context.Context, orgID, storeID, id string) (*Product, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	id = strings.TrimSpace(id)
	if orgID == "" || storeID == "" || id == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.FindByID(ctx, orgID, storeID, id)
}

func (s *Service) List(ctx context.Context, orgID, storeID string, filter Filter) ([]Product, int, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	if orgID == "" || storeID == "" {
		return nil, 0, ErrInvalidInput
	}
	if _, err := s.stores.FindByIDInOrg(ctx, orgID, storeID); err != nil {
		if errors.Is(err, stores.ErrNotFound) {
			return nil, 0, ErrStoreNotFound
		}
		return nil, 0, err
	}
	return s.repo.FindByStore(ctx, orgID, storeID, filter)
}

// Update full replace dalam store yang sama.
// store_id/organization_id tidak pernah berubah: kolom itu tidak ditulis repository.
func (s *Service) Update(ctx context.Context, orgID, storeID, id, name, sku, description string, price int64, stock int, unit string, categoryID *string, imageURL string, isActive bool) (*Product, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	sku = strings.TrimSpace(sku)
	description = strings.TrimSpace(description)
	unit = strings.TrimSpace(unit)
	imageURL = strings.TrimSpace(imageURL)
	if orgID == "" || storeID == "" || id == "" {
		return nil, ErrInvalidInput
	}
	if name == "" || sku == "" {
		return nil, ErrInvalidInput
	}
	if price < 0 {
		return nil, ErrInvalidPrice
	}
	if stock < 0 {
		return nil, ErrInvalidStock
	}
	if unit == "" {
		unit = "pcs"
	}

	dup, err := s.repo.ExistsBySKU(ctx, sku, storeID, &id)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, ErrSKUDuplicate
	}

	p := &Product{
		ID:             id,
		StoreID:        storeID,
		OrganizationID: orgID,
		Name:           name,
		SKU:            sku,
		Description:    description,
		Price:          price,
		Stock:          stock,
		Unit:           unit,
		CategoryID:     categoryID,
		ImageURL:       imageURL,
		IsActive:       isActive,
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Delete(ctx context.Context, orgID, storeID, id string) error {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	id = strings.TrimSpace(id)
	if orgID == "" || storeID == "" || id == "" {
		return ErrInvalidInput
	}
	return s.repo.SoftDelete(ctx, orgID, storeID, id)
}
