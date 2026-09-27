package tables

import (
	"context"
	"errors"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Service menangani business rules table dalam satu store.
type Service struct {
	repo   Repository
	stores StoreChecker
}

func NewService(repo Repository, stores StoreChecker) *Service {
	return &Service{repo: repo, stores: stores}
}

func (s *Service) Create(ctx context.Context, orgID, storeID string, in CreateRequest) (*Table, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	name := strings.TrimSpace(in.Name)
	area := strings.TrimSpace(in.Area)
	status := strings.TrimSpace(in.Status)
	if orgID == "" || storeID == "" || name == "" {
		return nil, ErrInvalidInput
	}
	if in.Capacity < 0 {
		return nil, ErrInvalidInput
	}
	if status == "" {
		status = StatusAvailable
	}
	if !validStatus(status) {
		return nil, ErrInvalidStatus
	}
	if err := s.checkStore(ctx, orgID, storeID); err != nil {
		return nil, err
	}

	dup, err := s.repo.ExistsByName(ctx, storeID, name, nil)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, ErrNameExists
	}

	t := &Table{
		OrganizationID: orgID,
		StoreID:        storeID,
		Name:           name,
		Area:           area,
		Capacity:       in.Capacity,
		Status:         status,
		IsActive:       in.IsActive == nil || *in.IsActive,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Get(ctx context.Context, orgID, storeID, id string) (*Table, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	id = strings.TrimSpace(id)
	if orgID == "" || storeID == "" || id == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.FindByID(ctx, orgID, storeID, id)
}

func (s *Service) List(ctx context.Context, orgID, storeID string, filter Filter) ([]Table, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	if orgID == "" || storeID == "" {
		return nil, ErrInvalidInput
	}
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Area = strings.TrimSpace(filter.Area)
	if filter.Status != "" && !validStatus(filter.Status) {
		return nil, ErrInvalidStatus
	}
	if err := s.checkStore(ctx, orgID, storeID); err != nil {
		return nil, err
	}
	list, _, err := s.repo.FindByStore(ctx, orgID, storeID, filter)
	return list, err
}

// Update full replace dalam store yang sama; status tidak diubah di sini.
func (s *Service) Update(ctx context.Context, orgID, storeID, id string, in UpdateRequest) (*Table, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	id = strings.TrimSpace(id)
	name := strings.TrimSpace(in.Name)
	area := strings.TrimSpace(in.Area)
	if orgID == "" || storeID == "" || id == "" || name == "" {
		return nil, ErrInvalidInput
	}
	if in.Capacity < 0 {
		return nil, ErrInvalidInput
	}
	if err := s.checkStore(ctx, orgID, storeID); err != nil {
		return nil, err
	}

	cur, err := s.repo.FindByID(ctx, orgID, storeID, id)
	if err != nil {
		return nil, err
	}

	dup, err := s.repo.ExistsByName(ctx, storeID, name, &id)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, ErrNameExists
	}

	cur.Name = name
	cur.Area = area
	cur.Capacity = in.Capacity
	if in.IsActive != nil {
		cur.IsActive = *in.IsActive
	}
	if err := s.repo.Update(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

// UpdateStatus mengubah occupancy meja; transisi bebas antar status valid
// karena operator yang memutuskan meja terisi/kosong.
func (s *Service) UpdateStatus(ctx context.Context, orgID, storeID, id, status string) (*Table, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	id = strings.TrimSpace(id)
	status = strings.TrimSpace(status)
	if orgID == "" || storeID == "" || id == "" {
		return nil, ErrInvalidInput
	}
	if !validStatus(status) {
		return nil, ErrInvalidStatus
	}
	if err := s.checkStore(ctx, orgID, storeID); err != nil {
		return nil, err
	}
	return s.repo.UpdateStatus(ctx, orgID, storeID, id, status)
}

func (s *Service) Delete(ctx context.Context, orgID, storeID, id string) error {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	id = strings.TrimSpace(id)
	if orgID == "" || storeID == "" || id == "" {
		return ErrInvalidInput
	}
	if err := s.checkStore(ctx, orgID, storeID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, orgID, storeID, id)
}

func (s *Service) checkStore(ctx context.Context, orgID, storeID string) error {
	if _, err := s.stores.FindByIDInOrg(ctx, orgID, storeID); err != nil {
		if errors.Is(err, stores.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func validStatus(s string) bool {
	switch s {
	case StatusAvailable, StatusOccupied, StatusReserved:
		return true
	}
	return false
}
