package variants_test

import (
	"context"
	"errors"
	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	"github.com/ramadhanrzq/backend-go/internal/modules/variants"
	"strconv"
	"testing"
)

// fakeRepo mengimplementasikan variants.Repository di memori per product.
type fakeRepo struct {
	items map[string]*variants.ProductVariant
	seq   int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: map[string]*variants.ProductVariant{}}
}

func keyOf(v *variants.ProductVariant) string {
	return v.OrganizationID + "|" + v.StoreID + "|" + v.ProductID + "|" + v.ID
}

func (f *fakeRepo) Create(_ context.Context, v *variants.ProductVariant) error {
	for _, it := range f.items {
		if it.StoreID == v.StoreID && it.ProductID == v.ProductID && it.Name == v.Name {
			return variants.ErrInvalidInput
		}
		if v.SKU != "" && it.StoreID == v.StoreID && it.SKU == v.SKU {
			return variants.ErrSKUExists
		}
	}
	f.seq++
	if v.ID == "" {
		v.ID = "v-" + strconv.Itoa(f.seq)
	}
	cp := *v
	f.items[keyOf(&cp)] = &cp
	*v = cp
	return nil
}

func (f *fakeRepo) FindByID(_ context.Context, orgID, storeID, productID, id string) (*variants.ProductVariant, error) {
	for _, it := range f.items {
		if it.ID == id {
			if it.OrganizationID != orgID || it.StoreID != storeID || it.ProductID != productID {
				return nil, variants.ErrNotFound
			}
			cp := *it
			return &cp, nil
		}
	}
	return nil, variants.ErrNotFound
}

func (f *fakeRepo) FindByProduct(_ context.Context, orgID, storeID, productID string) ([]variants.ProductVariant, error) {
	var out []variants.ProductVariant
	for _, it := range f.items {
		if it.OrganizationID == orgID && it.StoreID == storeID && it.ProductID == productID {
			out = append(out, *it)
		}
	}
	if out == nil {
		out = []variants.ProductVariant{}
	}
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, v *variants.ProductVariant) error {
	k := keyOf(v)
	cur, ok := f.items[k]
	if !ok {
		// Cari by ID untuk bedakan cross-org sebagai NotFound.
		for _, it := range f.items {
			if it.ID == v.ID {
				return variants.ErrNotFound
			}
		}
		return variants.ErrNotFound
	}
	for _, it := range f.items {
		if it.ID == v.ID {
			continue
		}
		if v.SKU != "" && it.StoreID == v.StoreID && it.SKU == v.SKU {
			return variants.ErrSKUExists
		}
	}
	*cur = *v
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, orgID, storeID, productID, id string) error {
	for k, it := range f.items {
		if it.ID == id {
			if it.OrganizationID != orgID || it.StoreID != storeID || it.ProductID != productID {
				return variants.ErrNotFound
			}
			delete(f.items, k)
			return nil
		}
	}
	return variants.ErrNotFound
}

func (f *fakeRepo) ExistsBySKU(_ context.Context, storeID, sku, excludeID string) (bool, error) {
	if sku == "" {
		return false, nil
	}
	for _, it := range f.items {
		if it.StoreID == storeID && it.SKU == sku && it.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

// fakeStores: hanya store-a milik org-a dengan product p-1.
type fakeStores struct{}

func (fakeStores) FindByIDInOrg(_ context.Context, orgID, id string) (*stores.Store, error) {
	if orgID == "org-a" && id == "store-a" {
		return &stores.Store{ID: id, OrganizationID: orgID}, nil
	}
	return nil, stores.ErrNotFound
}

type fakeProducts struct{}

func (fakeProducts) FindByID(_ context.Context, orgID, storeID, id string) (*products.Product, error) {
	if orgID == "org-a" && storeID == "store-a" && id == "p-1" {
		return &products.Product{ID: id, StoreID: storeID, OrganizationID: orgID}, nil
	}
	return nil, products.ErrNotFound
}

func svc() (*variants.Service, *fakeRepo) {
	r := newFakeRepo()
	return variants.NewService(r, fakeStores{}, fakeProducts{}), r
}

func TestService_CreateOK(t *testing.T) {
	s, _ := svc()
	v, err := s.Create(context.Background(), "org-a", "store-a", "p-1", "Large", "SKU-1", 5)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v.ID == "" || !v.IsActive {
		t.Fatalf("variant harus punya ID dan aktif: %+v", v)
	}
}

func TestService_InvalidInput(t *testing.T) {
	s, _ := svc()
	for _, tc := range []struct {
		name    string
		org     string
		store   string
		product string
		vname   string
		stock   int
	}{
		{"empty name", "org-a", "store-a", "p-1", "", 1},
		{"negative stock", "org-a", "store-a", "p-1", "Large", -1},
		{"empty org", "", "store-a", "p-1", "Large", 1},
	} {
		if _, err := s.Create(context.Background(), tc.org, tc.store, tc.product, tc.vname, "", tc.stock); !errors.Is(err, variants.ErrInvalidInput) {
			t.Fatalf("%s: harus ErrInvalidInput, got %v", tc.name, err)
		}
	}
}

func TestService_DuplicateSKU(t *testing.T) {
	s, _ := svc()
	if _, err := s.Create(context.Background(), "org-a", "store-a", "p-1", "Large", "SKU-DUP", 1); err != nil {
		t.Fatalf("create pertama: %v", err)
	}
	if _, err := s.Create(context.Background(), "org-a", "store-a", "p-1", "Small", "SKU-DUP", 1); !errors.Is(err, variants.ErrSKUExists) {
		t.Fatalf("SKU duplikat harus ErrSKUExists, got %v", err)
	}
}

func TestService_CrossOrgNotFound(t *testing.T) {
	s, r := svc()
	v, err := s.Create(context.Background(), "org-a", "store-a", "p-1", "Large", "", 1)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := r.FindByID(context.Background(), "org-b", "store-a", "p-1", v.ID); !errors.Is(err, variants.ErrNotFound) {
		t.Fatalf("cross-org harus NotFound, got %v", err)
	}
	if err := s.Delete(context.Background(), "org-b", "store-a", "p-1", v.ID); !errors.Is(err, variants.ErrStoreNotFound) {
		t.Fatalf("cross-org delete harus ErrStoreNotFound, got %v", err)
	}
}
