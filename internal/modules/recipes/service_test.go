package recipes_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/recipes"
	"github.com/ramadhanrzq/backend-go/internal/modules/stock"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// stubRepo menyimpan recipe + item di memori per store.
type stubRepo struct {
	items map[string]*recipes.Recipe
	order []string
	seq   int
}

func newStubRepo() *stubRepo {
	return &stubRepo{items: map[string]*recipes.Recipe{}}
}

func (s *stubRepo) WithTx(_ context.Context, fn func(context.Context) error) error {
	return fn(context.Background())
}

func (s *stubRepo) Create(_ context.Context, r *recipes.Recipe) error {
	for _, cur := range s.items {
		if cur.ProductID == r.ProductID && cur.Version == r.Version {
			return recipes.ErrVersionExists
		}
		// Meniru partial unique index: hanya satu resep aktif per product,
		// jadi insert resep aktif harus didahului DeactivateOthers.
		if r.IsActive && cur.ProductID == r.ProductID && cur.IsActive {
			return recipes.ErrInvalidInput
		}
	}
	s.seq++
	r.ID = "rec-" + string(rune('0'+s.seq))
	cp := *r
	s.items[r.ID] = &cp
	s.order = append(s.order, r.ID)
	return nil
}

func (s *stubRepo) FindByID(_ context.Context, orgID, storeID, productID, id string) (*recipes.Recipe, error) {
	r, ok := s.items[id]
	if !ok || r.OrganizationID != orgID || r.StoreID != storeID || r.ProductID != productID {
		return nil, recipes.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (s *stubRepo) FindByProduct(_ context.Context, orgID, storeID, productID string) ([]recipes.Recipe, error) {
	out := []recipes.Recipe{}
	for _, r := range s.items {
		if r.OrganizationID == orgID && r.StoreID == storeID && r.ProductID == productID {
			out = append(out, *r)
		}
	}
	return out, nil
}

func (s *stubRepo) FindByStore(_ context.Context, orgID, storeID string) ([]recipes.Recipe, error) {
	out := []recipes.Recipe{}
	for _, r := range s.items {
		if r.OrganizationID == orgID && r.StoreID == storeID {
			out = append(out, *r)
		}
	}
	return out, nil
}

func (s *stubRepo) FindActiveByProduct(_ context.Context, orgID, storeID, productID string) (*recipes.Recipe, error) {
	for _, r := range s.items {
		if r.OrganizationID == orgID && r.StoreID == storeID && r.ProductID == productID && r.IsActive {
			cp := *r
			return &cp, nil
		}
	}
	return nil, recipes.ErrNotFound
}

func (s *stubRepo) Update(_ context.Context, r *recipes.Recipe) error {
	cur, ok := s.items[r.ID]
	if !ok {
		return recipes.ErrNotFound
	}
	for _, other := range s.items {
		if other.ID == r.ID {
			continue
		}
		if other.ProductID == r.ProductID && other.Version == r.Version {
			return recipes.ErrVersionExists
		}
		if r.IsActive && other.ProductID == r.ProductID && other.IsActive {
			return recipes.ErrInvalidInput
		}
	}
	*cur = *r
	return nil
}

func (s *stubRepo) Delete(_ context.Context, orgID, storeID, productID, id string) error {
	r, ok := s.items[id]
	if !ok || r.OrganizationID != orgID || r.StoreID != storeID || r.ProductID != productID {
		return recipes.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *stubRepo) DeactivateOthers(_ context.Context, orgID, storeID, productID, keepID string) error {
	for _, r := range s.items {
		if r.OrganizationID == orgID && r.StoreID == storeID && r.ProductID == productID && r.ID != keepID {
			r.IsActive = false
		}
	}
	return nil
}

func (s *stubRepo) SetActive(_ context.Context, _, _, _, id string, active bool) error {
	r, ok := s.items[id]
	if !ok {
		return recipes.ErrNotFound
	}
	if active {
		for _, other := range s.items {
			if other.ID != id && other.ProductID == r.ProductID && other.IsActive {
				return recipes.ErrInvalidInput
			}
		}
	}
	r.IsActive = active
	return nil
}

func (s *stubRepo) LoadItems(_ context.Context, recipeID string) ([]recipes.RecipeItem, error) {
	r, ok := s.items[recipeID]
	if !ok {
		return nil, recipes.ErrNotFound
	}
	return r.Items, nil
}

func (s *stubRepo) ReplaceItems(_ context.Context, recipeID string, items []recipes.RecipeItem) error {
	r, ok := s.items[recipeID]
	if !ok {
		return recipes.ErrNotFound
	}
	r.Items = items
	return nil
}

type stubStores struct{}

func (stubStores) FindByIDInOrg(_ context.Context, orgID, id string) (*stores.Store, error) {
	if orgID == "org-a" && id == "store-a" {
		return &stores.Store{ID: id, OrganizationID: orgID}, nil
	}
	return nil, stores.ErrNotFound
}

// stubProducts: hanya SKU yang terdaftar ada di store-a.
type stubProducts struct{ skus map[string]bool }

func (s stubProducts) FindByID(_ context.Context, orgID, storeID, id string) (*products.Product, error) {
	if orgID != "org-a" || storeID != "store-a" || !s.skus[id] {
		return nil, products.ErrNotFound
	}
	return &products.Product{ID: id, Name: "Bahan " + id, SKU: id, StoreID: storeID, OrganizationID: orgID}, nil
}

// stubRecorder mencatat pemanggilan Record untuk verifikasi konsumsi bahan.
type stubRecorder struct {
	calls []stock.StockMovement
	err   error
}

func (s *stubRecorder) Record(_ context.Context, orgID, storeID, productID string, variantID *string, typ string, qty int, refType, refID, notes, createdBy string, _ bool) (*stock.StockMovement, error) {
	if s.err != nil {
		return nil, s.err
	}
	m := stock.StockMovement{
		OrganizationID: orgID, StoreID: storeID, ProductID: productID, VariantID: variantID,
		Type: typ, Quantity: qty, ReferenceType: refType, ReferenceID: refID,
		Notes: notes, CreatedBy: createdBy,
	}
	s.calls = append(s.calls, m)
	return &m, nil
}

func newSvc() (*recipes.Service, *stubRepo, *stubRecorder) {
	repo := newStubRepo()
	rec := &stubRecorder{}
	prods := stubProducts{skus: map[string]bool{"menu-1": true, "susu": true, "espresso": true, "gula": true}}
	return recipes.NewService(repo, stubStores{}, prods, rec), repo, rec
}

func items() []recipes.ItemInput {
	return []recipes.ItemInput{
		{IngredientProductID: "susu", Quantity: 250, Unit: "ml"},
		{IngredientProductID: "espresso", Quantity: 20, Unit: "ml"},
		{IngredientProductID: "gula", Quantity: 15, Unit: "ml", WastagePercentage: 10},
	}
}

func TestService_CreateWithItems(t *testing.T) {
	s, _, _ := newSvc()

	rec, err := s.Create(context.Background(), "org-a", "store-a", "menu-1", "user-a", "Kopi Susu", 1, 1, true, "", items())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !rec.IsActive || len(rec.Items) != 3 {
		t.Fatalf("recipe tidak lengkap: %+v", rec)
	}
	if rec.Items[0].Quantity != 250 || rec.Items[0].Unit != "ml" {
		t.Fatalf("item pertama salah: %+v", rec.Items[0])
	}
	if rec.Items[0].IngredientName == "" {
		t.Fatal("nama bahan harus diisi dari product checker")
	}
	if rec.Items[2].WastagePercentage != 10 {
		t.Fatalf("wastage hilang: %+v", rec.Items[2])
	}
}

func TestService_CreateRejectsBadItems(t *testing.T) {
	s, _, _ := newSvc()
	ctx := context.Background()

	cases := []struct {
		name  string
		items []recipes.ItemInput
		want  error
	}{
		{"kosong", nil, recipes.ErrInvalidInput},
		{"quantity nol", []recipes.ItemInput{{IngredientProductID: "susu", Quantity: 0}}, recipes.ErrInvalidInput},
		{"bahan sendiri", []recipes.ItemInput{{IngredientProductID: "menu-1", Quantity: 1}}, recipes.ErrIngredientSelf},
		{"duplikat", []recipes.ItemInput{
			{IngredientProductID: "susu", Quantity: 1},
			{IngredientProductID: "susu", Quantity: 2},
		}, recipes.ErrIngredientDuplicate},
		{"bahan store lain", []recipes.ItemInput{{IngredientProductID: "tidak-ada", Quantity: 1}}, recipes.ErrIngredientNotFound},
		{"wastage > 100", []recipes.ItemInput{{IngredientProductID: "susu", Quantity: 1, WastagePercentage: 150}}, recipes.ErrInvalidInput},
	}
	for _, c := range cases {
		if _, err := s.Create(ctx, "org-a", "store-a", "menu-1", "user-a", "X", 1, 1, false, "", c.items); !errors.Is(err, c.want) {
			t.Errorf("%s: want %v, got %v", c.name, c.want, err)
		}
	}

	if _, err := s.Create(ctx, "org-a", "store-a", "ghost", "user-a", "X", 1, 1, false, "", items()); !errors.Is(err, recipes.ErrProductNotFound) {
		t.Fatalf("product hilang harus ErrProductNotFound, got %v", err)
	}
}

// Mengaktifkan versi baru harus menonaktifkan versi lama: hanya satu resep aktif.
func TestService_ActivateKeepsSingleActive(t *testing.T) {
	s, _, _ := newSvc()
	ctx := context.Background()

	v1, err := s.Create(ctx, "org-a", "store-a", "menu-1", "user-a", "Kopi Susu v1", 1, 1, true, "", items())
	if err != nil {
		t.Fatalf("Create v1: %v", err)
	}
	v2, err := s.Create(ctx, "org-a", "store-a", "menu-1", "user-a", "Kopi Susu v2", 2, 1, true, "", items())
	if err != nil {
		t.Fatalf("Create v2: %v", err)
	}

	old, err := s.Get(ctx, "org-a", "store-a", "menu-1", v1.ID)
	if err != nil {
		t.Fatalf("Get v1: %v", err)
	}
	if old.IsActive {
		t.Fatal("v1 harus nonaktif setelah v2 aktif")
	}
	if !v2.IsActive {
		t.Fatal("v2 harus aktif")
	}

	if _, err := s.Activate(ctx, "org-a", "store-a", "menu-1", v1.ID, true); err != nil {
		t.Fatalf("Activate v1: %v", err)
	}
	list, err := s.List(ctx, "org-a", "store-a", "menu-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	active := 0
	for _, r := range list {
		if r.IsActive {
			active++
			if r.ID != v1.ID {
				t.Fatalf("resep aktif harus v1, got %s", r.ID)
			}
		}
	}
	if active != 1 {
		t.Fatalf("harus tepat satu resep aktif, got %d", active)
	}
}

func TestService_VersionConflict(t *testing.T) {
	s, _, _ := newSvc()
	ctx := context.Background()

	if _, err := s.Create(ctx, "org-a", "store-a", "menu-1", "user-a", "v1", 1, 1, false, "", items()); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := s.Create(ctx, "org-a", "store-a", "menu-1", "user-a", "v1 lagi", 1, 1, false, "", items()); !errors.Is(err, recipes.ErrVersionExists) {
		t.Fatalf("version duplikat harus ErrVersionExists, got %v", err)
	}
}

// Konsumsi bahan: 2 gelas Kopi Susu → susu 500ml, espresso 40ml, gula 33ml
// (15 x 2 = 30, +10% wastage = 33 dibulatkan ke atas).
func TestService_ConsumeScalesWithQuantity(t *testing.T) {
	s, _, rec := newSvc()
	ctx := context.Background()

	if _, err := s.Create(ctx, "org-a", "store-a", "menu-1", "user-a", "Kopi Susu", 1, 1, true, "", items()); err != nil {
		t.Fatalf("Create: %v", err)
	}

	res, err := s.Consume(ctx, "org-a", "store-a", "menu-1", 2, "sale-1", "user-a")
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if res == nil || len(res.Items) != 3 {
		t.Fatalf("konsumsi tidak lengkap: %+v", res)
	}
	want := map[string]int{"susu": 500, "espresso": 40, "gula": 33}
	for _, it := range res.Items {
		if want[it.ProductID] != it.QuantityUsed {
			t.Errorf("%s: want %d, got %d", it.ProductID, want[it.ProductID], it.QuantityUsed)
		}
	}
	for _, m := range rec.calls {
		if m.Type != stock.TypeOut || m.ReferenceType != recipes.ReferenceType || m.ReferenceID != "sale-1" {
			t.Errorf("movement salah: %+v", m)
		}
	}
}

// yield_quantity > 1 berarti satu batch resep menghasilkan beberapa porsi.
func TestService_ConsumeHonoursYield(t *testing.T) {
	s, _, _ := newSvc()
	ctx := context.Background()

	// 1 batch = 4 gelas; jual 2 gelas → separuh resep.
	if _, err := s.Create(ctx, "org-a", "store-a", "menu-1", "user-a", "Batch 4 gelas", 1, 4, true, "", []recipes.ItemInput{
		{IngredientProductID: "susu", Quantity: 1000, Unit: "ml"},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	res, err := s.Consume(ctx, "org-a", "store-a", "menu-1", 2, "sale-2", "user-a")
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if res.Items[0].QuantityUsed != 500 {
		t.Fatalf("want 500ml, got %d", res.Items[0].QuantityUsed)
	}
}

// Product tanpa resep aktif tetap bisa dijual: tidak ada bahan yang dikurangi.
func TestService_ConsumeWithoutRecipeIsNoop(t *testing.T) {
	s, _, rec := newSvc()

	res, err := s.Consume(context.Background(), "org-a", "store-a", "menu-1", 1, "sale-3", "user-a")
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if res != nil {
		t.Fatalf("tanpa resep harus nil, got %+v", res)
	}
	if len(rec.calls) != 0 {
		t.Fatalf("tanpa resep tidak boleh mencatat movement, got %d", len(rec.calls))
	}
}

// Stok bahan kurang diteruskan apa adanya supaya sale di-rollback caller.
func TestService_ConsumePropagatesStockError(t *testing.T) {
	s, _, rec := newSvc()
	rec.err = stock.ErrInsufficient
	ctx := context.Background()

	if _, err := s.Create(ctx, "org-a", "store-a", "menu-1", "user-a", "Kopi Susu", 1, 1, true, "", items()); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := s.Consume(ctx, "org-a", "store-a", "menu-1", 1, "sale-4", "user-a"); !errors.Is(err, stock.ErrInsufficient) {
		t.Fatalf("want ErrInsufficient, got %v", err)
	}
}

func TestService_UpdateReplacesItems(t *testing.T) {
	s, _, _ := newSvc()
	ctx := context.Background()

	rec, err := s.Create(ctx, "org-a", "store-a", "menu-1", "user-a", "Kopi Susu", 1, 1, true, "", items())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Update(ctx, "org-a", "store-a", "menu-1", rec.ID, "Kopi Susu Gula Aren", 1, 1, true, "",
		[]recipes.ItemInput{{IngredientProductID: "gula", Quantity: 30, Unit: "ml"}})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Name != "Kopi Susu Gula Aren" || len(got.Items) != 1 || got.Items[0].Quantity != 30 {
		t.Fatalf("update tidak full replace: %+v", got)
	}

	if _, err := s.Update(ctx, "org-b", "store-a", "menu-1", rec.ID, "X", 1, 1, true, "", items()); !errors.Is(err, recipes.ErrNotFound) {
		t.Fatalf("cross-org harus NotFound, got %v", err)
	}
	if err := s.Delete(ctx, "org-b", "store-a", "menu-1", rec.ID); !errors.Is(err, recipes.ErrNotFound) {
		t.Fatalf("delete cross-org harus NotFound, got %v", err)
	}
	if err := s.Delete(ctx, "org-a", "store-a", "menu-1", rec.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, "org-a", "store-a", "menu-1", rec.ID); !errors.Is(err, recipes.ErrNotFound) {
		t.Fatalf("get setelah delete harus NotFound, got %v", err)
	}
}

func TestService_ListScope(t *testing.T) {
	s, _, _ := newSvc()
	ctx := context.Background()

	if _, err := s.Create(ctx, "org-a", "store-a", "menu-1", "user-a", "Kopi Susu", 1, 1, true, "", items()); err != nil {
		t.Fatalf("Create: %v", err)
	}
	byProduct, err := s.List(ctx, "org-a", "store-a", "menu-1")
	if err != nil || len(byProduct) != 1 {
		t.Fatalf("list per product: %v %d", err, len(byProduct))
	}
	byStore, err := s.List(ctx, "org-a", "store-a", "")
	if err != nil || len(byStore) != 1 {
		t.Fatalf("list per store: %v %d", err, len(byStore))
	}
	if _, err := s.List(ctx, "org-a", "store-x", ""); !errors.Is(err, recipes.ErrStoreNotFound) {
		t.Fatalf("store lain harus ErrStoreNotFound, got %v", err)
	}
}
