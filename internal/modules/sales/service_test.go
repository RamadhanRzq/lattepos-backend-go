package sales_test

import (
	"context"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/sales"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// stubRepo menyimpan sale di memori; Cancel hanya dari pending.
type stubRepo struct {
	items map[string]*sales.Sale
}

func newStubRepo() *stubRepo { return &stubRepo{items: map[string]*sales.Sale{}} }

func (s *stubRepo) Create(_ context.Context, sa *sales.Sale) error {
	sa.ID = "sale-1"
	sa.Status = sales.StatusPending
	for i := range sa.Items {
		sa.Items[i].ID = "item"
		sa.Items[i].SaleID = sa.ID
	}
	cp := *sa
	s.items[sa.ID] = &cp
	return nil
}

func (s *stubRepo) FindByID(_ context.Context, _, _, id string) (*sales.Sale, error) {
	sa, ok := s.items[id]
	if !ok {
		return nil, sales.ErrNotFound
	}
	return sa, nil
}

func (s *stubRepo) FindItems(_ context.Context, saleID string) ([]sales.SaleItem, error) {
	sa, ok := s.items[saleID]
	if !ok {
		return nil, sales.ErrNotFound
	}
	return sa.Items, nil
}

func (s *stubRepo) FindByStore(_ context.Context, _, _ string, _ sales.Filter) ([]sales.Sale, int, error) {
	out := []sales.Sale{}
	for _, sa := range s.items {
		out = append(out, *sa)
	}
	return out, len(out), nil
}

func (s *stubRepo) Cancel(_ context.Context, _, _, id string) (*sales.Sale, error) {
	sa, ok := s.items[id]
	if !ok {
		return nil, sales.ErrNotFound
	}
	if sa.Status != sales.StatusPending {
		return nil, sales.ErrInvalidStatus
	}
	sa.Status = sales.StatusCancelled
	return sa, nil
}

// stubStores: store-a milik org-a, user kasir1 punya akses.
type stubStores struct{ assigned bool }

func (s stubStores) FindByIDInOrg(_ context.Context, orgID, id string) (*stores.Store, error) {
	if orgID == "org-a" && id == "store-a" {
		return &stores.Store{ID: "store-a", OrganizationID: "org-a"}, nil
	}
	return nil, stores.ErrNotFound
}

func (s stubStores) IsAssigned(_ context.Context, _, _ string) (bool, error) {
	return s.assigned, nil
}

type stubProducts struct{ price int64 }

func (s stubProducts) FindByID(_ context.Context, orgID, storeID, id string) (*products.Product, error) {
	if orgID != "org-a" || storeID != "store-a" || id == "" {
		return nil, products.ErrNotFound
	}
	return &products.Product{ID: id, StoreID: storeID, Price: s.price}, nil
}

func svc(assigned bool) (*sales.Service, *stubRepo) {
	repo := newStubRepo()
	return sales.NewService(repo, stubStores{assigned: assigned}, stubProducts{price: 15000}), repo
}

func lines() []sales.SaleLine {
	return []sales.SaleLine{{ProductID: "prod-1", Quantity: 2}, {ProductID: "prod-2", Quantity: 1}}
}

func TestService_CreateSnapshotsPriceAndTotals(t *testing.T) {
	s, _ := svc(true)
	got, err := s.Create(context.Background(), "org-a", "store-a", "kasir1", "cash", 5000, 1000, "", lines())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// total 3x15000=45000, grand 45000-5000+1000=41000.
	if got.TotalAmount != 45000 || got.GrandTotal != 41000 {
		t.Fatalf("total=%d grand=%d, mau 45000/41000", got.TotalAmount, got.GrandTotal)
	}
	for _, it := range got.Items {
		if it.UnitPrice != 15000 || it.Subtotal != int64(it.Quantity)*15000 {
			t.Fatalf("item bukan snapshot harga produk: %+v", it)
		}
	}
}

func TestService_CreateRejectsNoAccess(t *testing.T) {
	s, _ := svc(false)
	_, err := s.Create(context.Background(), "org-a", "store-a", "intruder", "cash", 0, 0, "", lines())
	if err != sales.ErrNoAccess {
		t.Fatalf("mau ErrNoAccess, dapat %v", err)
	}
}

func TestService_CancelPendingOnly(t *testing.T) {
	s, repo := svc(true)
	got, err := s.Create(context.Background(), "org-a", "store-a", "kasir1", "cash", 0, 0, "", lines())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := s.Cancel(context.Background(), "org-a", "store-a", "kasir1", got.ID); err != nil {
		t.Fatalf("Cancel pending: %v", err)
	}
	if _, err := s.Cancel(context.Background(), "org-a", "store-a", "kasir1", got.ID); err != sales.ErrInvalidStatus {
		t.Fatalf("cancel kedua mau ErrInvalidStatus, dapat %v", err)
	}
	repo.items["s-done"] = &sales.Sale{ID: "s-done", Status: sales.StatusCompleted}
	if _, err := s.Cancel(context.Background(), "org-a", "store-a", "kasir1", "s-done"); err != sales.ErrInvalidStatus {
		t.Fatalf("cancel completed mau ErrInvalidStatus, dapat %v", err)
	}
}
