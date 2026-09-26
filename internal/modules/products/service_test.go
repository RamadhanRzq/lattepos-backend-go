package products_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// stubRepo mengimplementasikan products.Repository di memori per store.
type stubRepo struct {
	products.Repository
	items map[string]*products.Product
}

func newStubRepo() *stubRepo {
	return &stubRepo{items: map[string]*products.Product{}}
}

func (s *stubRepo) Create(_ context.Context, p *products.Product) error {
	if p.SKU == "TAKEN" {
		return products.ErrSKUDuplicate
	}
	p.ID = "prod-" + p.SKU
	p.IsActive = true
	cp := *p
	s.items[p.ID] = &cp
	return nil
}

func (s *stubRepo) FindByID(_ context.Context, orgID, storeID, id string) (*products.Product, error) {
	p, ok := s.items[id]
	if !ok || p.OrganizationID != orgID || p.StoreID != storeID {
		return nil, products.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (s *stubRepo) FindByStore(_ context.Context, orgID, storeID string, _ products.Filter) ([]products.Product, int, error) {
	var out []products.Product
	for _, p := range s.items {
		if p.OrganizationID == orgID && p.StoreID == storeID {
			out = append(out, *p)
		}
	}
	return out, len(out), nil
}

func (s *stubRepo) Update(_ context.Context, p *products.Product) error {
	cur, ok := s.items[p.ID]
	if !ok || cur.OrganizationID != p.OrganizationID || cur.StoreID != p.StoreID {
		return products.ErrNotFound
	}
	*cur = *p
	return nil
}

func (s *stubRepo) SoftDelete(_ context.Context, orgID, storeID, id string) error {
	p, ok := s.items[id]
	if !ok || p.OrganizationID != orgID || p.StoreID != storeID {
		return products.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *stubRepo) ExistsBySKU(_ context.Context, sku, storeID string, excludeID *string) (bool, error) {
	for _, p := range s.items {
		if p.StoreID != storeID || p.SKU != sku {
			continue
		}
		if excludeID != nil && p.ID == *excludeID {
			continue
		}
		return true, nil
	}
	return sku == "TAKEN", nil
}

// stubStores: hanya store-a milik org-a.
type stubStores struct{}

// stubDetail: DetailSources kosong; detail product tetap terisi dari entity.
type stubDetail struct{}

func (stubDetail) ListVariants(context.Context, string, string, string) ([]products.VariantView, error) {
	return []products.VariantView{}, nil
}

func (stubDetail) ListPrices(context.Context, string, string, string) ([]products.PriceView, error) {
	return []products.PriceView{}, nil
}

func (stubStores) FindByIDInOrg(_ context.Context, orgID, id string) (*stores.Store, error) {
	if orgID == "org-a" && id == "store-a" {
		return &stores.Store{ID: id, OrganizationID: orgID}, nil
	}
	return nil, stores.ErrNotFound
}

func svc() (*products.Service, *stubRepo) {
	repo := newStubRepo()
	return products.NewService(repo, stubStores{}, stubDetail{}), repo
}

func TestService_Create(t *testing.T) {
	s, _ := svc()

	got, err := s.Create(context.Background(), "org-a", "store-a", "user-a", "Kopi Susu", "KOPI-01", "", "", 15000, 10, "pcs", nil, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.OrganizationID != "org-a" || got.StoreID != "store-a" || !got.IsActive {
		t.Fatalf("Create boundary/active: %+v", got)
	}

	if _, err := s.Create(context.Background(), "org-a", "store-a", "user-a", "", "X", "", "", 0, 0, "", nil, ""); !errors.Is(err, products.ErrInvalidInput) {
		t.Fatalf("name kosong harus ErrInvalidInput, got %v", err)
	}
	if _, err := s.Create(context.Background(), "org-a", "store-a", "user-a", "N", "TAKEN", "", "", 0, 0, "", nil, ""); !errors.Is(err, products.ErrSKUDuplicate) {
		t.Fatalf("sku duplikat harus ErrSKUDuplicate, got %v", err)
	}
	if _, err := s.Create(context.Background(), "org-a", "store-a", "user-a", "N", "X", "", "", -1, 0, "", nil, ""); !errors.Is(err, products.ErrInvalidPrice) {
		t.Fatalf("price negatif harus ErrInvalidPrice, got %v", err)
	}
	if _, err := s.Create(context.Background(), "org-a", "store-a", "user-a", "N", "X", "", "", 0, -1, "", nil, ""); !errors.Is(err, products.ErrInvalidStock) {
		t.Fatalf("stock negatif harus ErrInvalidStock, got %v", err)
	}
	if _, err := s.Create(context.Background(), "org-a", "no-store", "user-a", "N", "X", "", "", 0, 0, "", nil, ""); !errors.Is(err, products.ErrStoreNotFound) {
		t.Fatalf("store hilang harus ErrStoreNotFound, got %v", err)
	}
}

func TestService_CreateProductType(t *testing.T) {
	s, _ := svc()

	raw, err := s.Create(context.Background(), "org-a", "store-a", "user-a", "Susu Segar", "BB-01", "", "raw_material", 0, 500, "ml", nil, "")
	if err != nil {
		t.Fatalf("Create raw material: %v", err)
	}
	if raw.ProductType != products.TypeRawMaterial {
		t.Fatalf("product_type harus RAW_MATERIAL, got %q", raw.ProductType)
	}

	def, err := s.Create(context.Background(), "org-a", "store-a", "user-a", "Kopi Tubruk", "MN-01", "", "", 15000, 10, "pcs", nil, "")
	if err != nil {
		t.Fatalf("Create tanpa product_type: %v", err)
	}
	if def.ProductType != products.TypeMenu {
		t.Fatalf("product_type default harus MENU, got %q", def.ProductType)
	}

	if _, err := s.Create(context.Background(), "org-a", "store-a", "user-a", "X", "X-01", "", "BAHAN", 0, 0, "", nil, ""); !errors.Is(err, products.ErrInvalidProductType) {
		t.Fatalf("tipe tak dikenal harus ErrInvalidProductType, got %v", err)
	}

	if _, err := s.Update(context.Background(), "org-a", "store-a", raw.ID, "Susu Segar", "BB-01", "", "PACKAGING", 0, 500, "ml", nil, "", true); err != nil {
		t.Fatalf("Update ke PACKAGING: %v", err)
	} else if got, _ := s.Get(context.Background(), "org-a", "store-a", raw.ID); got.ProductType != products.TypePackaging {
		t.Fatalf("product_type setelah update harus PACKAGING, got %q", got.ProductType)
	}
}

func TestService_TenantIsolation(t *testing.T) {
	s, _ := svc()
	ctx := context.Background()

	st, err := s.Create(ctx, "org-a", "store-a", "user-a", "Kopi", "KOPI-01", "", "", 1000, 1, "", nil, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if st.Unit != "pcs" {
		t.Fatalf("unit default harus pcs, got %q", st.Unit)
	}

	if _, err := s.Get(ctx, "org-b", "store-a", st.ID); !errors.Is(err, products.ErrNotFound) {
		t.Fatalf("product org lain harus NotFound, got %v", err)
	}
	if _, err := s.Get(ctx, "org-a", "store-b", st.ID); !errors.Is(err, products.ErrNotFound) {
		t.Fatalf("product store lain harus NotFound, got %v", err)
	}
	list, total, err := s.List(ctx, "org-a", "store-a", products.Filter{})
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("list store sendiri harus 1, got %v %d %+v", err, total, list)
	}
	if _, _, err := s.List(ctx, "org-a", "store-b", products.Filter{}); !errors.Is(err, products.ErrStoreNotFound) {
		t.Fatalf("list store hilang harus ErrStoreNotFound, got %v", err)
	}
	if _, _, err := s.List(ctx, "org-b", "store-a", products.Filter{}); !errors.Is(err, products.ErrStoreNotFound) {
		t.Fatalf("list store org lain harus ErrStoreNotFound, got %v", err)
	}
}

func TestService_UpdateDelete(t *testing.T) {
	s, _ := svc()
	ctx := context.Background()

	st, _ := s.Create(ctx, "org-a", "store-a", "user-a", "Kopi", "KOPI-01", "", "", 1000, 1, "", nil, "")
	active := true

	// SKU milik product lain di store yang sama ditolak.
	other, _ := s.Create(ctx, "org-a", "store-a", "user-a", "Teh", "TEH-01", "", "", 500, 2, "", nil, "")
	if _, err := s.Update(ctx, "org-a", "store-a", st.ID, "Kopi Baru", other.SKU, "", "", 1200, 3, "pcs", nil, "", active); !errors.Is(err, products.ErrSKUDuplicate) {
		t.Fatalf("sku duplikat harus ErrSKUDuplicate, got %v", err)
	}

	got, err := s.Update(ctx, "org-a", "store-a", st.ID, "Kopi Baru", "KOPI-02", "", "", 1200, 3, "pcs", nil, "", active)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Name != "Kopi Baru" || got.StoreID != "store-a" || got.OrganizationID != "org-a" {
		t.Fatalf("Update pindah scope: %+v", got)
	}
	if _, err := s.Update(ctx, "org-b", "store-a", st.ID, "X", "X", "", "", 0, 0, "", nil, "", active); !errors.Is(err, products.ErrNotFound) {
		t.Fatalf("update org lain harus NotFound, got %v", err)
	}

	if err := s.Delete(ctx, "org-a", "store-a", st.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, "org-a", "store-a", st.ID); !errors.Is(err, products.ErrNotFound) {
		t.Fatalf("get setelah delete harus NotFound, got %v", err)
	}
}
