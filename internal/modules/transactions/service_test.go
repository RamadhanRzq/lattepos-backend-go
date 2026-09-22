package transactions_test

import (
	"context"
	"testing"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/sales"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	"github.com/ramadhanrzq/backend-go/internal/modules/transactions"
)

type stubSales struct {
	list []sales.Sale
	err  error
	got  sales.Filter
}

func (s *stubSales) List(_ context.Context, _, _, _ string, f sales.Filter) ([]sales.Sale, int, error) {
	s.got = f
	if s.err != nil {
		return nil, 0, s.err
	}
	return s.list, len(s.list), nil
}

type stubStores struct{}

func (stubStores) FindByIDInOrg(_ context.Context, _, _ string) (*stores.Store, error) {
	return &stores.Store{ID: "store-a", OrganizationID: "org-a"}, nil
}

func (stubStores) IsAssigned(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

// Daily harus kirim window 00:00-23:59.999 hari diminta dan agregat benar.
func TestDailyAggregatesDay(t *testing.T) {
	day := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	mk := func(h int, method, status string, grand int64, pid string, qty int, sub int64) sales.Sale {
		return sales.Sale{
			StoreID: "store-a", PaymentMethod: method, Status: status, GrandTotal: grand,
			CreatedAt: day.Add(time.Duration(h) * time.Hour),
			Items:     []sales.SaleItem{{ProductID: pid, Quantity: qty, Subtotal: sub}},
		}
	}
	src := &stubSales{list: []sales.Sale{
		mk(9, "cash", sales.StatusCompleted, 50000, "kopi", 2, 50000),
		mk(10, "qris", sales.StatusCompleted, 30000, "kopi", 1, 30000),
		mk(11, "cash", sales.StatusCancelled, 20000, "teh", 1, 20000),
	}}
	svc := transactions.NewService(src, stubStores{})

	r, err := svc.Daily(context.Background(), "org-a", "store-a", "u1", "2026-09-22")
	if err != nil {
		t.Fatal(err)
	}
	if r.Count != 3 || r.GrandTotal != 100000 {
		t.Fatalf("count/grand salah: %+v", r)
	}
	if r.ByStatus[sales.StatusCompleted] != 80000 || r.ByStatus[sales.StatusCancelled] != 20000 {
		t.Fatalf("byStatus salah: %+v", r.ByStatus)
	}
	if r.ByPayment["cash"] != 70000 || r.ByPayment["qris"] != 30000 {
		t.Fatalf("byPayment salah: %+v", r.ByPayment)
	}
	if r.ByHour["09"] != 50000 || r.ByHour["10"] != 30000 || r.ByHour["11"] != 20000 {
		t.Fatalf("byHour salah: %+v", r.ByHour)
	}
	found := map[string]transactions.DailyItem{}
	for _, it := range r.Items {
		found[it.ProductID] = it
	}
	if found["kopi"].Quantity != 3 || found["kopi"].Subtotal != 80000 {
		t.Fatalf("agregat kopi salah: %+v", found["kopi"])
	}
	if src.got.From.Format("15:04") != "00:00" {
		t.Fatalf("from bukan awal hari: %v", src.got.From)
	}
}

// Date rusak harus 400-level (ErrInvalidInput), bukan 500.
func TestDailyBadDate(t *testing.T) {
	svc := transactions.NewService(&stubSales{}, stubStores{})
	if _, err := svc.Daily(context.Background(), "org-a", "store-a", "u1", "22-09-2026"); err == nil {
		t.Fatal("date rusak harus error")
	}
}
