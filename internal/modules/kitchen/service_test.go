package kitchen_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/kitchen"
	"github.com/ramadhanrzq/backend-go/internal/modules/sales"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// fakeRepo menyimpan kitchen sale di memori per ID dan per sale.
type fakeRepo struct {
	byID   map[string]*kitchen.KitchenSale
	bySale map[string]*kitchen.KitchenSale
	seq    int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[string]*kitchen.KitchenSale{}, bySale: map[string]*kitchen.KitchenSale{}}
}

func (f *fakeRepo) Create(_ context.Context, ks *kitchen.KitchenSale) error {
	if _, dup := f.bySale[ks.SaleID]; dup {
		return kitchen.ErrInvalidInput
	}
	f.seq++
	ks.ID = fmt.Sprintf("kitchen-%d", f.seq)
	ks.Status = kitchen.StatusPending
	for i := range ks.Items {
		ks.Items[i].ID = fmt.Sprintf("kitem-%d-%d", f.seq, i)
		ks.Items[i].KitchenSaleID = ks.ID
		ks.Items[i].Status = kitchen.StatusPending
	}
	cp := *ks
	cp.Items = append([]kitchen.KitchenSaleItem(nil), ks.Items...)
	f.byID[ks.ID] = &cp
	f.bySale[ks.SaleID] = &cp
	return nil
}

func (f *fakeRepo) FindByID(_ context.Context, orgID, _, id string) (*kitchen.KitchenSale, error) {
	ks, ok := f.byID[id]
	if !ok || ks.OrganizationID != orgID {
		return nil, kitchen.ErrNotFound
	}
	cp := *ks
	cp.Items = append([]kitchen.KitchenSaleItem(nil), ks.Items...)
	return &cp, nil
}

func (f *fakeRepo) FindQueue(_ context.Context, orgID, _ string) ([]kitchen.KitchenSale, error) {
	var out []kitchen.KitchenSale
	for _, ks := range f.byID {
		if ks.OrganizationID != orgID {
			continue
		}
		if ks.Status == kitchen.StatusPending || ks.Status == kitchen.StatusPreparing {
			cp := *ks
			cp.Items = append([]kitchen.KitchenSaleItem(nil), ks.Items...)
			out = append(out, cp)
		}
	}
	if out == nil {
		out = []kitchen.KitchenSale{}
	}
	return out, nil
}

func (f *fakeRepo) FindBySaleID(_ context.Context, saleID string) (*kitchen.KitchenSale, error) {
	ks, ok := f.bySale[saleID]
	if !ok {
		return nil, kitchen.ErrNotFound
	}
	cp := *ks
	cp.Items = append([]kitchen.KitchenSaleItem(nil), ks.Items...)
	return &cp, nil
}

func (f *fakeRepo) UpdateStatus(_ context.Context, _, _, id, status string, started, completed *time.Time) (*kitchen.KitchenSale, error) {
	ks, ok := f.byID[id]
	if !ok {
		return nil, kitchen.ErrNotFound
	}
	ks.Status = status
	if started != nil {
		ks.StartedAt = started
	}
	if completed != nil {
		ks.CompletedAt = completed
	}
	cp := *ks
	cp.Items = append([]kitchen.KitchenSaleItem(nil), ks.Items...)
	return &cp, nil
}

func (f *fakeRepo) UpdateItemStatus(_ context.Context, _, _, kitchenID, itemID, status string) (*kitchen.KitchenSaleItem, error) {
	ks, ok := f.byID[kitchenID]
	if !ok {
		return nil, kitchen.ErrNotFound
	}
	for i := range ks.Items {
		if ks.Items[i].ID == itemID {
			ks.Items[i].Status = status
			cp := ks.Items[i]
			return &cp, nil
		}
	}
	return nil, kitchen.ErrNotFound
}

func (f *fakeRepo) FindItems(_ context.Context, kitchenID string) ([]kitchen.KitchenSaleItem, error) {
	ks, ok := f.byID[kitchenID]
	if !ok {
		return nil, kitchen.ErrNotFound
	}
	return append([]kitchen.KitchenSaleItem(nil), ks.Items...), nil
}

// stubStores: store-a milik org-a, user cook1 punya akses.
type stubStores struct{ assigned bool }

func (s stubStores) FindByIDInOrg(_ context.Context, orgID, id string) (*stores.Store, error) {
	if orgID != "org-a" || id != "store-a" {
		return nil, stores.ErrNotFound
	}
	return &stores.Store{ID: "store-a", OrganizationID: "org-a"}, nil
}

func (s stubStores) IsAssigned(_ context.Context, _, _ string) (bool, error) {
	return s.assigned, nil
}

// stubSales: sale-1 pending milik org-a/store-a; sale-x cancelled.
type stubSales struct{}

func (stubSales) FindByID(_ context.Context, orgID, storeID, id string) (*sales.Sale, error) {
	if orgID != "org-a" || storeID != "store-a" {
		return nil, sales.ErrNotFound
	}
	switch id {
	case "sale-1":
		return &sales.Sale{ID: "sale-1", OrganizationID: "org-a", StoreID: "store-a", Status: sales.StatusPending}, nil
	case "sale-x":
		return &sales.Sale{ID: "sale-x", OrganizationID: "org-a", StoreID: "store-a", Status: sales.StatusCancelled}, nil
	default:
		return nil, sales.ErrNotFound
	}
}

func svc(assigned bool) (*kitchen.Service, *fakeRepo) {
	repo := newFakeRepo()
	return kitchen.NewService(repo, stubStores{assigned: assigned}, stubSales{}), repo
}

func refs() []kitchen.SaleItemRef {
	return []kitchen.SaleItemRef{
		{SaleItemID: "si-1", ProductID: "prod-1", Quantity: 2},
		{SaleItemID: "si-2", ProductID: "prod-2", Quantity: 1},
	}
}

func TestService_CreateForSaleOK(t *testing.T) {
	s, _ := svc(true)
	ks, err := s.CreateForSale(context.Background(), "org-a", "store-a", "sale-1", "cook1", "", refs())
	if err != nil {
		t.Fatalf("create ok, got %v", err)
	}
	if ks.ID == "" || ks.Status != kitchen.StatusPending {
		t.Fatalf("bad kitchen sale: %+v", ks)
	}
	for _, it := range ks.Items {
		if it.Status != kitchen.StatusPending {
			t.Fatalf("item harus pending, got %s", it.Status)
		}
	}
}

func TestService_CreateForSaleInvalidInput(t *testing.T) {
	s, _ := svc(true)
	if _, err := s.CreateForSale(context.Background(), "org-a", "store-a", "sale-1", "cook1", "", nil); !errors.Is(err, kitchen.ErrInvalidInput) {
		t.Fatalf("items kosong harus ErrInvalidInput, got %v", err)
	}
	if _, err := s.CreateForSale(context.Background(), "", "store-a", "sale-1", "cook1", "", refs()); !errors.Is(err, kitchen.ErrInvalidInput) {
		t.Fatalf("org kosong harus ErrInvalidInput, got %v", err)
	}
}

func TestService_CreateForSaleDuplicate(t *testing.T) {
	s, _ := svc(true)
	if _, err := s.CreateForSale(context.Background(), "org-a", "store-a", "sale-1", "cook1", "", refs()); err != nil {
		t.Fatalf("create pertama, got %v", err)
	}
	if _, err := s.CreateForSale(context.Background(), "org-a", "store-a", "sale-1", "cook1", "", refs()); !errors.Is(err, kitchen.ErrInvalidInput) {
		t.Fatalf("duplikat sale harus ErrInvalidInput, got %v", err)
	}
}

func TestService_CreateForSaleCrossOrgNotFound(t *testing.T) {
	s, _ := svc(true)
	_, err := s.Queue(context.Background(), "org-b", "store-a", "cook1")
	if !errors.Is(err, kitchen.ErrStoreNotFound) {
		t.Fatalf("store org lain harus ErrStoreNotFound, got %v", err)
	}
	if _, err := s.CreateForSale(context.Background(), "org-a", "store-a", "ghost", "cook1", "", refs()); !errors.Is(err, kitchen.ErrSaleNotFound) {
		t.Fatalf("sale hilang harus ErrSaleNotFound, got %v", err)
	}
}

func TestService_IllegalJumpPendingReadyRejected(t *testing.T) {
	s, _ := svc(true)
	ks, err := s.CreateForSale(context.Background(), "org-a", "store-a", "sale-1", "cook1", "", refs())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateStatus(context.Background(), "org-a", "store-a", "cook1", ks.ID, kitchen.StatusReady); !errors.Is(err, kitchen.ErrInvalidTransition) {
		t.Fatalf("pending->ready harus ErrInvalidTransition, got %v", err)
	}
}

func TestService_AnyToCancelledOK(t *testing.T) {
	for _, from := range []string{kitchen.StatusPending, kitchen.StatusPreparing, kitchen.StatusReady} {
		s, repo := svc(true)
		ks, err := s.CreateForSale(context.Background(), "org-a", "store-a", "sale-1", "cook1", "", refs())
		if err != nil {
			t.Fatal(err)
		}
		repo.byID[ks.ID].Status = from
		got, err := s.UpdateStatus(context.Background(), "org-a", "store-a", "cook1", ks.ID, kitchen.StatusCancelled)
		if err != nil {
			t.Fatalf("%s->cancelled harus ok, got %v", from, err)
		}
		if got.Status != kitchen.StatusCancelled || got.CompletedAt == nil {
			t.Fatalf("%s->cancelled harus set completed_at, got %+v", from, got)
		}
	}
}

func TestService_AllItemsReadyAutoFires(t *testing.T) {
	s, repo := svc(true)
	ks, err := s.CreateForSale(context.Background(), "org-a", "store-a", "sale-1", "cook1", "", refs())
	if err != nil {
		t.Fatal(err)
	}
	repo.byID[ks.ID].Status = kitchen.StatusPreparing
	// item pertama: pending->preparing->ready
	if _, err := s.UpdateItemStatus(context.Background(), "org-a", "store-a", "cook1", ks.ID, ks.Items[0].ID, kitchen.StatusPreparing); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateItemStatus(context.Background(), "org-a", "store-a", "cook1", ks.ID, ks.Items[0].ID, kitchen.StatusReady); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get(context.Background(), "org-a", "store-a", "cook1", ks.ID); got.Status == kitchen.StatusReady {
		t.Fatal("satu item ready belum boleh auto-fire")
	}
	// item kedua: pending->preparing->ready → semua ready, auto-fire
	if _, err := s.UpdateItemStatus(context.Background(), "org-a", "store-a", "cook1", ks.ID, ks.Items[1].ID, kitchen.StatusPreparing); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateItemStatus(context.Background(), "org-a", "store-a", "cook1", ks.ID, ks.Items[1].ID, kitchen.StatusReady); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(context.Background(), "org-a", "store-a", "cook1", ks.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != kitchen.StatusReady {
		t.Fatalf("semua ready harus auto-fire ke ready, got %s", got.Status)
	}
}

func TestService_CancelledSaleMutationBlocked(t *testing.T) {
	s, repo := svc(true)
	ks, err := s.CreateForSale(context.Background(), "org-a", "store-a", "sale-1", "cook1", "", refs())
	if err != nil {
		t.Fatal(err)
	}
	repo.byID[ks.ID].SaleID = "sale-x"
	if _, err := s.UpdateStatus(context.Background(), "org-a", "store-a", "cook1", ks.ID, kitchen.StatusPreparing); !errors.Is(err, kitchen.ErrSaleCancelled) {
		t.Fatalf("sale cancelled harus ErrSaleCancelled, got %v", err)
	}
	repo.byID[ks.ID].SaleID = "sale-x"
	if _, err := s.UpdateItemStatus(context.Background(), "org-a", "store-a", "cook1", ks.ID, ks.Items[0].ID, kitchen.StatusPreparing); !errors.Is(err, kitchen.ErrSaleCancelled) {
		t.Fatalf("item sale cancelled harus ErrSaleCancelled, got %v", err)
	}
}

func TestService_UnknownStatusRejected(t *testing.T) {
	s, _ := svc(true)
	ks, err := s.CreateForSale(context.Background(), "org-a", "store-a", "sale-1", "cook1", "", refs())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateStatus(context.Background(), "org-a", "store-a", "cook1", ks.ID, "flying"); !errors.Is(err, kitchen.ErrInvalidStatus) {
		t.Fatalf("status asing harus ErrInvalidStatus, got %v", err)
	}
}

func TestService_QueueOnlyActive(t *testing.T) {
	s, repo := svc(true)
	ks, err := s.CreateForSale(context.Background(), "org-a", "store-a", "sale-1", "cook1", "", refs())
	if err != nil {
		t.Fatal(err)
	}
	repo.byID[ks.ID].Status = kitchen.StatusServed
	q, err := s.Queue(context.Background(), "org-a", "store-a", "cook1")
	if err != nil {
		t.Fatal(err)
	}
	if len(q) != 0 {
		t.Fatalf("served keluar dari queue, got %d", len(q))
	}
}

func (f *fakeRepo) CancelBySale(_ context.Context, orgID, storeID, saleID string) (int64, error) {
	ks, ok := f.bySale[saleID]
	if !ok {
		return 0, nil
	}
	if ks.OrganizationID != orgID || ks.StoreID != storeID {
		return 0, nil
	}
	if ks.Status != kitchen.StatusPending && ks.Status != kitchen.StatusPreparing {
		return 0, nil
	}
	ks.Status = kitchen.StatusCancelled
	for i := range ks.Items {
		if ks.Items[i].Status == kitchen.StatusPending || ks.Items[i].Status == kitchen.StatusPreparing {
			ks.Items[i].Status = kitchen.StatusCancelled
		}
	}
	return 1, nil
}
