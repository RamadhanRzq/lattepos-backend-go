package categories_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/modules/categories"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// stubRepo mengimplementasikan categories.Repository di memori per store.
type stubRepo struct {
	categories.Repository
	items       map[string]*categories.Category
	productUses map[string]int
}

func newStubRepo() *stubRepo {
	return &stubRepo{items: map[string]*categories.Category{}, productUses: map[string]int{}}
}

func (s *stubRepo) Create(_ context.Context, c *categories.Category) error {
	for _, cur := range s.items {
		if cur.StoreID == c.StoreID && cur.Slug == c.Slug {
			return categories.ErrSlugExists
		}
	}
	c.ID = "cat-" + c.Slug
	cp := *c
	s.items[c.ID] = &cp
	return nil
}

func (s *stubRepo) FindByID(_ context.Context, orgID, storeID, id string) (*categories.Category, error) {
	c, ok := s.items[id]
	if !ok || c.OrganizationID != orgID || c.StoreID != storeID {
		return nil, categories.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (s *stubRepo) FindBySlug(_ context.Context, orgID, storeID, slug string) (*categories.Category, error) {
	for _, c := range s.items {
		if c.OrganizationID == orgID && c.StoreID == storeID && c.Slug == slug {
			cp := *c
			return &cp, nil
		}
	}
	return nil, categories.ErrNotFound
}

func (s *stubRepo) FindByStore(_ context.Context, orgID, storeID string) ([]categories.Category, error) {
	var out []categories.Category
	for _, c := range s.items {
		if c.OrganizationID == orgID && c.StoreID == storeID {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (s *stubRepo) Update(_ context.Context, c *categories.Category) error {
	cur, ok := s.items[c.ID]
	if !ok || cur.OrganizationID != c.OrganizationID || cur.StoreID != c.StoreID {
		return categories.ErrNotFound
	}
	*cur = *c
	return nil
}

func (s *stubRepo) Delete(_ context.Context, orgID, storeID, id string) error {
	c, ok := s.items[id]
	if !ok || c.OrganizationID != orgID || c.StoreID != storeID {
		return categories.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *stubRepo) ExistsBySlug(_ context.Context, slug, storeID string, excludeID *string) (bool, error) {
	for _, c := range s.items {
		if c.StoreID != storeID || c.Slug != slug {
			continue
		}
		if excludeID != nil && c.ID == *excludeID {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (s *stubRepo) CountProducts(_ context.Context, storeID, categoryID string) (int, error) {
	return s.productUses[storeID+"/"+categoryID], nil
}

// stubStores: hanya store-a milik org-a.
type stubStores struct{}

func (stubStores) FindByIDInOrg(_ context.Context, orgID, id string) (*stores.Store, error) {
	if orgID == "org-a" && id == "store-a" {
		return &stores.Store{ID: id, OrganizationID: orgID}, nil
	}
	return nil, stores.ErrNotFound
}

func svc() (*categories.Service, *stubRepo) {
	repo := newStubRepo()
	return categories.NewService(repo, stubStores{}), repo
}

func TestService_CreateOK(t *testing.T) {
	s, _ := svc()

	got, err := s.Create(context.Background(), "org-a", "store-a", "Minuman", "minuman", "Aneka minuman", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Name != "Minuman" || got.Slug != "minuman" {
		t.Fatalf("field tak sesuai: %+v", got)
	}
}

func TestService_CreateAutoSlug(t *testing.T) {
	s, _ := svc()

	got, err := s.Create(context.Background(), "org-a", "store-a", "Kopi Susu Gula Aren", "", "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Slug != "kopi-susu-gula-aren" {
		t.Fatalf("auto-slug salah: %q", got.Slug)
	}
}

func TestService_CreateInvalidInput(t *testing.T) {
	s, _ := svc()

	if _, err := s.Create(context.Background(), "org-a", "store-a", "   ", "", "", nil); !errors.Is(err, categories.ErrInvalidInput) {
		t.Fatalf("nama kosong harus ErrInvalidInput, got %v", err)
	}
	if _, err := s.Create(context.Background(), "", "store-a", "Minuman", "", "", nil); !errors.Is(err, categories.ErrInvalidInput) {
		t.Fatalf("org kosong harus ErrInvalidInput, got %v", err)
	}
}

func TestService_CreateDuplicateSlug(t *testing.T) {
	s, _ := svc()
	ctx := context.Background()

	if _, err := s.Create(ctx, "org-a", "store-a", "Minuman", "minuman", "", nil); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.Create(ctx, "org-a", "store-a", "Minuman Lain", "minuman", "", nil); !errors.Is(err, categories.ErrSlugExists) {
		t.Fatalf("slug duplikat harus ErrSlugExists, got %v", err)
	}
}

func TestService_ParentMustBeSameStore(t *testing.T) {
	s, repo := svc()
	ctx := context.Background()

	parent, err := s.Create(ctx, "org-a", "store-a", "Makanan", "", "", nil)
	if err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	// Parent beda store: tanam langsung ke repo meniru store lain.
	other := &categories.Category{ID: "cat-other", OrganizationID: "org-a", StoreID: "store-b", Name: "Lain", Slug: "lain"}
	repo.items[other.ID] = other

	if _, err := s.Create(ctx, "org-a", "store-a", "Anak", "", "", &other.ID); !errors.Is(err, categories.ErrInvalidParent) {
		t.Fatalf("parent beda store harus ErrInvalidParent, got %v", err)
	}

	got, err := s.Create(ctx, "org-a", "store-a", "Anak", "", "", &parent.ID)
	if err != nil {
		t.Fatalf("parent satu store harus lolos: %v", err)
	}
	if got.ParentID == nil || *got.ParentID != parent.ID {
		t.Fatalf("parentID tak tersimpan: %+v", got)
	}
}

func TestService_DeleteBlockedWhenInUse(t *testing.T) {
	s, repo := svc()
	ctx := context.Background()

	c, err := s.Create(ctx, "org-a", "store-a", "Minuman", "", "", nil)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	repo.productUses["store-a/"+c.ID] = 2

	if err := s.Delete(ctx, "org-a", "store-a", c.ID); !errors.Is(err, categories.ErrInUse) {
		t.Fatalf("kategori terpakai harus ErrInUse, got %v", err)
	}
}

func TestService_CrossOrgNotFound(t *testing.T) {
	s, _ := svc()
	ctx := context.Background()

	c, err := s.Create(ctx, "org-a", "store-a", "Minuman", "", "", nil)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.Get(ctx, "org-b", "store-a", c.ID); !errors.Is(err, categories.ErrNotFound) {
		t.Fatalf("org lain harus NotFound, got %v", err)
	}
	if err := s.Delete(ctx, "org-b", "store-a", c.ID); !errors.Is(err, categories.ErrNotFound) {
		t.Fatalf("delete org lain harus NotFound, got %v", err)
	}
}
