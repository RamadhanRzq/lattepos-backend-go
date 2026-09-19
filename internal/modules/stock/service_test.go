package stock_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/modules/stock"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// fakeRepo simulasi stok live di memori; Record meniru semantik tx postgres.
type fakeRepo struct {
	productStock int
	hasProduct   bool
	summaries    map[string]*stock.StockSummary
	err          error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{productStock: 10, hasProduct: true, summaries: map[string]*stock.StockSummary{}}
}

func (f *fakeRepo) Record(_ context.Context, m *stock.StockMovement, allowNegative bool) error {
	if f.err != nil {
		return f.err
	}
	if !f.hasProduct {
		return stock.ErrProductNotFound
	}
	var after int
	switch m.Type {
	case stock.TypeIn, stock.TypeReturn:
		after = f.productStock + m.Quantity
	case stock.TypeOut:
		after = f.productStock - m.Quantity
		if after < 0 {
			return stock.ErrInsufficient
		}
	case stock.TypeAdjustment:
		after = f.productStock + m.Quantity
		if after < 0 && !allowNegative {
			return stock.ErrInsufficient
		}
	default:
		return stock.ErrInvalidType
	}
	m.ID = "mov-1"
	m.StockBefore = f.productStock
	m.StockAfter = after
	f.productStock = after
	return nil
}

func (f *fakeRepo) FindByStore(_ context.Context, _, _ string, _ stock.Filter) ([]stock.StockMovement, int, error) {
	return []stock.StockMovement{}, 0, nil
}

func (f *fakeRepo) FindByProduct(_ context.Context, _, _, _ string) ([]stock.StockMovement, error) {
	return []stock.StockMovement{}, nil
}

func (f *fakeRepo) GetSummary(_ context.Context, orgID, storeID, productID string) (*stock.StockSummary, error) {
	if !f.hasProduct {
		return nil, stock.ErrProductNotFound
	}
	if s, ok := f.summaries[productID]; ok {
		return s, nil
	}
	return &stock.StockSummary{
		ProductID:    productID,
		ProductStock: f.productStock,
		Variants:     []stock.VariantStock{{VariantID: "var-1", Stock: 3}},
	}, nil
}

// fakeStores: store-a milik org-a, kasir1 punya akses.
type fakeStores struct{ assigned bool }

func (s fakeStores) FindByIDInOrg(_ context.Context, orgID, id string) (*stores.Store, error) {
	if orgID == "org-a" && id == "store-a" {
		return &stores.Store{ID: "store-a", OrganizationID: "org-a"}, nil
	}
	return nil, stores.ErrNotFound
}

func (s fakeStores) IsAssigned(_ context.Context, _, _ string) (bool, error) {
	return s.assigned, nil
}

func svc(assigned bool) (*stock.Service, *fakeRepo) {
	repo := newFakeRepo()
	return stock.NewService(repo, fakeStores{assigned: assigned}), repo
}

func TestService_RecordOK(t *testing.T) {
	s, _ := svc(true)
	got, err := s.Record(context.Background(), "org-a", "store-a", "prod-1", nil, "in", 5, "", "", "", "kasir1", false)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if got.StockBefore != 10 || got.StockAfter != 15 {
		t.Fatalf("before=%d after=%d, mau 10/15", got.StockBefore, got.StockAfter)
	}
}

func TestService_RecordInvalidInput(t *testing.T) {
	s, _ := svc(true)
	if _, err := s.Record(context.Background(), "org-a", "store-a", "prod-1", nil, "bogus", 5, "", "", "", "kasir1", false); !errors.Is(err, stock.ErrInvalidType) {
		t.Fatalf("bad type mau ErrInvalidType, dapat %v", err)
	}
	if _, err := s.Record(context.Background(), "org-a", "store-a", "prod-1", nil, "in", 0, "", "", "", "kasir1", false); !errors.Is(err, stock.ErrInvalidInput) {
		t.Fatalf("qty 0 mau ErrInvalidInput, dapat %v", err)
	}
}

func TestService_RecordNoAccess(t *testing.T) {
	s, _ := svc(false)
	if _, err := s.Record(context.Background(), "org-a", "store-a", "prod-1", nil, "in", 1, "", "", "", "intruder", false); !errors.Is(err, stock.ErrNoAccess) {
		t.Fatalf("mau ErrNoAccess, dapat %v", err)
	}
}

func TestService_RecordCrossOrgNotFound(t *testing.T) {
	s, _ := svc(true)
	if _, err := s.Record(context.Background(), "org-b", "store-a", "prod-1", nil, "in", 1, "", "", "", "kasir1", false); !errors.Is(err, stock.ErrStoreNotFound) {
		t.Fatalf("cross-org mau ErrStoreNotFound, dapat %v", err)
	}
}

func TestService_RecordInsufficient(t *testing.T) {
	s, _ := svc(true)
	if _, err := s.Record(context.Background(), "org-a", "store-a", "prod-1", nil, "out", 99, "", "", "", "kasir1", false); !errors.Is(err, stock.ErrInsufficient) {
		t.Fatalf("out over-stock mau ErrInsufficient, dapat %v", err)
	}
	// adjustment negatif (delta) hanya di level repo: service menolak qty<=0,
	// repo melipat delta negatif ke before+qty dan menolak tanpa allowNegative.
	repo := newFakeRepo()
	m := &stock.StockMovement{Type: stock.TypeAdjustment, Quantity: -99}
	if err := repo.Record(context.Background(), m, false); !errors.Is(err, stock.ErrInsufficient) {
		t.Fatalf("adjustment negatif tanpa izin mau ErrInsufficient, dapat %v", err)
	}
	m2 := &stock.StockMovement{Type: stock.TypeAdjustment, Quantity: -99}
	if err := repo.Record(context.Background(), m2, true); err != nil {
		t.Fatalf("adjustment negatif dengan allowNegative: %v", err)
	}
}

func TestService_RecordDuplicateMapped(t *testing.T) {
	s, repo := svc(true)
	repo.err = stock.ErrInvalidInput
	if _, err := s.Record(context.Background(), "org-a", "store-a", "prod-1", nil, "in", 1, "", "", "", "kasir1", false); !errors.Is(err, stock.ErrInvalidInput) {
		t.Fatalf("FK gagal mau ErrInvalidInput, dapat %v", err)
	}
}

func TestService_StockByProductShape(t *testing.T) {
	s, repo := svc(true)
	repo.summaries["prod-1"] = &stock.StockSummary{
		ProductID:    "prod-1",
		ProductStock: 10,
		Variants:     []stock.VariantStock{{VariantID: "var-1", Stock: 3}},
	}
	sum, history, err := s.StockByProduct(context.Background(), "org-a", "store-a", "kasir1", "prod-1")
	if err != nil {
		t.Fatalf("StockByProduct: %v", err)
	}
	if sum.ProductID != "prod-1" || sum.ProductStock != 10 || len(sum.Variants) != 1 || sum.Variants[0].VariantID != "var-1" {
		t.Fatalf("summary salah: %+v", sum)
	}
	if history == nil {
		t.Fatalf("history harus [] bukan nil")
	}
}
