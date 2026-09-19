package prices_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/prices"
	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	"github.com/ramadhanrzq/backend-go/internal/modules/variants"
)

// fakeRepo mengimplementasikan prices.Repository di memori.
type fakeRepo struct {
	items map[string]*prices.ProductPrice
	seq   int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: map[string]*prices.ProductPrice{}}
}

func (f *fakeRepo) Create(_ context.Context, p *prices.ProductPrice) error {
	f.seq++
	if p.ID == "" {
		p.ID = "pr-" + strconv.Itoa(f.seq)
	}
	cp := *p
	f.items[cp.ID] = &cp
	*p = cp
	return nil
}

func (f *fakeRepo) FindByID(_ context.Context, orgID, storeID, productID, priceID string) (*prices.ProductPrice, error) {
	it, ok := f.items[priceID]
	if !ok || it.OrganizationID != orgID || it.StoreID != storeID || it.ProductID != productID {
		return nil, prices.ErrNotFound
	}
	cp := *it
	return &cp, nil
}

func (f *fakeRepo) FindByProduct(_ context.Context, orgID, storeID, productID string, onlyActive bool) ([]prices.ProductPrice, error) {
	var out []prices.ProductPrice
	for _, it := range f.items {
		if it.OrganizationID != orgID || it.StoreID != storeID || it.ProductID != productID {
			continue
		}
		if onlyActive && !it.IsActive {
			continue
		}
		out = append(out, *it)
	}
	if out == nil {
		out = []prices.ProductPrice{}
	}
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, p *prices.ProductPrice) error {
	cur, ok := f.items[p.ID]
	if !ok || cur.OrganizationID != p.OrganizationID || cur.StoreID != p.StoreID || cur.ProductID != p.ProductID {
		return prices.ErrNotFound
	}
	*cur = *p
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, orgID, storeID, productID, priceID string) error {
	it, ok := f.items[priceID]
	if !ok || it.OrganizationID != orgID || it.StoreID != storeID || it.ProductID != productID {
		return prices.ErrNotFound
	}
	delete(f.items, priceID)
	return nil
}

// FindEffective meniru semantik SQL: filter aktif+tipe+jendela+minQty,
// kecocokan variant (nil = hanya base; non-nil = variant itu atau base),
// variant-specific didahulukan lalu harga termurah.
func (f *fakeRepo) FindEffective(_ context.Context, orgID, storeID, productID string, variantID *string, priceType string, qty int, now time.Time) (*prices.ProductPrice, error) {
	var best *prices.ProductPrice
	for _, it := range f.items {
		if it.OrganizationID != orgID || it.StoreID != storeID || it.ProductID != productID {
			continue
		}
		if !it.IsActive || it.PriceType != priceType || it.MinQuantity > qty {
			continue
		}
		if it.ValidFrom != nil && it.ValidFrom.After(now) {
			continue
		}
		if it.ValidUntil != nil && it.ValidUntil.Before(now) {
			continue
		}
		if variantID != nil && *variantID != "" {
			if it.VariantID != nil && *it.VariantID != *variantID {
				continue
			}
		} else if it.VariantID != nil {
			continue
		}
		if best == nil || rank(it) < rank(best) || (rank(it) == rank(best) && it.Price < best.Price) {
			cp := *it
			best = &cp
		}
	}
	return best, nil
}

func rank(p *prices.ProductPrice) int {
	if p.VariantID == nil {
		return 1
	}
	return 0
}

// fakeStores: hanya store-a milik org-a.
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

// fakeVariants: v-1 milik p-1, v-9 milik p-9 (untuk uji mismatch).
type fakeVariants struct{}

func (fakeVariants) FindByID(_ context.Context, orgID, storeID, productID, id string) (*variants.ProductVariant, error) {
	if orgID != "org-a" || storeID != "store-a" {
		return nil, variants.ErrNotFound
	}
	switch id {
	case "v-1":
		return &variants.ProductVariant{ID: id, ProductID: "p-1", StoreID: storeID}, nil
	case "v-9":
		return &variants.ProductVariant{ID: id, ProductID: "p-9", StoreID: storeID}, nil
	}
	return nil, variants.ErrNotFound
}

func svc() (*prices.Service, *fakeRepo) {
	r := newFakeRepo()
	return prices.NewService(r, fakeStores{}, fakeProducts{}, fakeVariants{}), r
}

func strp(s string) *string { return &s }

func TestService_CreateOK(t *testing.T) {
	s, _ := svc()
	p, err := s.Create(context.Background(), "org-a", "store-a", "p-1", strp("v-1"), "retail", 10000, 1, nil, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.ID == "" || !p.IsActive || p.Price != 10000 {
		t.Fatalf("price cacat: %+v", p)
	}
}

func TestService_InvalidInput(t *testing.T) {
	s, _ := svc()
	if _, err := s.Create(context.Background(), "org-a", "store-a", "p-1", nil, "retail", -1, 1, nil, nil); !errors.Is(err, prices.ErrInvalidInput) {
		t.Fatalf("harga negatif harus ErrInvalidInput, got %v", err)
	}
	if _, err := s.Create(context.Background(), "org-a", "store-a", "p-1", nil, "retail", 100, 0, nil, nil); !errors.Is(err, prices.ErrInvalidInput) {
		t.Fatalf("minQty 0 harus ErrInvalidInput, got %v", err)
	}
	from := time.Now()
	until := from.Add(-time.Hour)
	if _, err := s.Create(context.Background(), "org-a", "store-a", "p-1", nil, "retail", 100, 1, &from, &until); !errors.Is(err, prices.ErrInvalidInput) {
		t.Fatalf("rentang valid rusak harus ErrInvalidInput, got %v", err)
	}
}

func TestService_VariantProductMismatch(t *testing.T) {
	s, _ := svc()
	// v-9 milik p-9, dipasang ke p-1 → tolak.
	if _, err := s.Create(context.Background(), "org-a", "store-a", "p-1", strp("v-9"), "retail", 100, 1, nil, nil); !errors.Is(err, prices.ErrInvalidInput) {
		t.Fatalf("variant beda product harus ErrInvalidInput, got %v", err)
	}
	// variant tak ada → ErrVariantNotFound.
	if _, err := s.Create(context.Background(), "org-a", "store-a", "p-1", strp("ghost"), "retail", 100, 1, nil, nil); !errors.Is(err, prices.ErrVariantNotFound) {
		t.Fatalf("variant hilang harus ErrVariantNotFound, got %v", err)
	}
}

func TestService_CrossOrgNotFound(t *testing.T) {
	s, r := svc()
	p, err := s.Create(context.Background(), "org-a", "store-a", "p-1", nil, "retail", 5000, 1, nil, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := r.FindByID(context.Background(), "org-b", "store-a", "p-1", p.ID); !errors.Is(err, prices.ErrNotFound) {
		t.Fatalf("cross-org harus NotFound, got %v", err)
	}
}

func TestService_EffectivePrefersVariantSpecific(t *testing.T) {
	s, r := svc()
	ctx := context.Background()
	if _, err := s.Create(ctx, "org-a", "store-a", "p-1", nil, "retail", 9000, 1, nil, nil); err != nil {
		t.Fatalf("create base: %v", err)
	}
	if _, err := s.Create(ctx, "org-a", "store-a", "p-1", strp("v-1"), "retail", 12000, 1, nil, nil); err != nil {
		t.Fatalf("create variant: %v", err)
	}
	got, found, err := s.EffectivePrice(ctx, "org-a", "store-a", "p-1", strp("v-1"), "retail", 1)
	if err != nil || !found || got != 12000 {
		t.Fatalf("variant-specific harus menang: got=%d found=%v err=%v", got, found, err)
	}
	// Tanpa variantID, base dipakai.
	got, found, err = s.EffectivePrice(ctx, "org-a", "store-a", "p-1", nil, "retail", 1)
	if err != nil || !found || got != 9000 {
		t.Fatalf("base harus dipakai: got=%d found=%v err=%v", got, found, err)
	}
	_ = r
}

func TestService_EffectiveIgnoresExpired(t *testing.T) {
	s, _ := svc()
	ctx := context.Background()
	past := time.Now().Add(-2 * time.Hour)
	expired := time.Now().Add(-time.Hour)
	if _, err := s.Create(ctx, "org-a", "store-a", "p-1", nil, "retail", 7000, 1, &past, &expired); err != nil {
		t.Fatalf("create expired: %v", err)
	}
	if _, found, err := s.EffectivePrice(ctx, "org-a", "store-a", "p-1", nil, "retail", 1); err != nil || found {
		t.Fatalf("expired harus diabaikan: found=%v err=%v", found, err)
	}
}
