package categories

import (
	"context"
	"errors"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Service menangani business rules category dalam satu store.
type Service struct {
	repo   Repository
	stores StoreChecker
}

func NewService(repo Repository, stores StoreChecker) *Service {
	return &Service{repo: repo, stores: stores}
}

func (s *Service) Create(ctx context.Context, orgID, storeID, name, slug, desc string, parentID *string) (*Category, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	name = strings.TrimSpace(name)
	slug = strings.TrimSpace(slug)
	desc = strings.TrimSpace(desc)
	if orgID == "" || storeID == "" {
		return nil, ErrInvalidInput
	}
	if name == "" {
		return nil, ErrInvalidInput
	}
	if slug == "" {
		slug = slugify(name)
	}
	if slug == "" {
		return nil, ErrInvalidInput
	}
	if err := s.checkStore(ctx, orgID, storeID); err != nil {
		return nil, err
	}

	// Normalisasi parent: trim, string kosong dianggap nil.
	parentID = normalizeParentID(parentID)
	if parentID != nil {
		if _, err := s.repo.FindByID(ctx, orgID, storeID, *parentID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrInvalidParent
			}
			return nil, err
		}
	}

	dup, err := s.repo.ExistsBySlug(ctx, slug, storeID, nil)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, ErrSlugExists
	}

	c := &Category{
		StoreID:        storeID,
		OrganizationID: orgID,
		Name:           name,
		Slug:           slug,
		Description:    desc,
		ParentID:       parentID,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) Get(ctx context.Context, orgID, storeID, id string) (*Category, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	id = strings.TrimSpace(id)
	if orgID == "" || storeID == "" || id == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.FindByID(ctx, orgID, storeID, id)
}

func (s *Service) List(ctx context.Context, orgID, storeID string) ([]Category, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	if orgID == "" || storeID == "" {
		return nil, ErrInvalidInput
	}
	if err := s.checkStore(ctx, orgID, storeID); err != nil {
		return nil, err
	}
	return s.repo.FindByStore(ctx, orgID, storeID)
}

// Update full replace dalam store yang sama.
// store_id/organization_id tidak pernah berubah: kolom itu tidak ditulis repository.
func (s *Service) Update(ctx context.Context, orgID, storeID, id, name, slug, desc string, parentID *string) (*Category, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	slug = strings.TrimSpace(slug)
	desc = strings.TrimSpace(desc)
	if orgID == "" || storeID == "" || id == "" {
		return nil, ErrInvalidInput
	}
	if name == "" {
		return nil, ErrInvalidInput
	}
	if slug == "" {
		slug = slugify(name)
	}
	if slug == "" {
		return nil, ErrInvalidInput
	}
	if err := s.checkStore(ctx, orgID, storeID); err != nil {
		return nil, err
	}

	cur, err := s.repo.FindByID(ctx, orgID, storeID, id)
	if err != nil {
		return nil, err
	}

	parentID = normalizeParentID(parentID)
	if parentID != nil {
		if *parentID == id {
			return nil, ErrInvalidParent
		}
		if _, err := s.repo.FindByID(ctx, orgID, storeID, *parentID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrInvalidParent
			}
			return nil, err
		}
	}

	dup, err := s.repo.ExistsBySlug(ctx, slug, storeID, &id)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, ErrSlugExists
	}

	cur.Name = name
	cur.Slug = slug
	cur.Description = desc
	cur.ParentID = parentID
	if err := s.repo.Update(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
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
	if _, err := s.repo.FindByID(ctx, orgID, storeID, id); err != nil {
		return err
	}
	n, err := s.repo.CountProducts(ctx, storeID, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrInUse
	}
	return s.repo.Delete(ctx, orgID, storeID, id)
}

// checkStore menutupi cross-org sebagai NotFound (bukan StoreNotFound).
func (s *Service) checkStore(ctx context.Context, orgID, storeID string) error {
	if _, err := s.stores.FindByIDInOrg(ctx, orgID, storeID); err != nil {
		if errors.Is(err, stores.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func normalizeParentID(parentID *string) *string {
	if parentID == nil {
		return nil
	}
	v := strings.TrimSpace(*parentID)
	if v == "" {
		return nil
	}
	return &v
}

// slugify: lower, spasi jadi -, hanya [a-z0-9-].
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '_' || r == '-':
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
