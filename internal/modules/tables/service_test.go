package tables_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	"github.com/ramadhanrzq/backend-go/internal/modules/tables"
)

// stubRepo mengimplementasikan tables.Repository di memori per store.
type stubRepo struct {
	tables.Repository
	items map[string]*tables.Table
}

func newStubRepo() *stubRepo {
	return &stubRepo{items: map[string]*tables.Table{}}
}

func (s *stubRepo) Create(_ context.Context, t *tables.Table) error {
	for _, cur := range s.items {
		if cur.StoreID == t.StoreID && cur.Name == t.Name {
			return tables.ErrNameExists
		}
	}
	t.ID = "tbl-" + t.Name
	cp := *t
	s.items[t.ID] = &cp
	return nil
}

func (s *stubRepo) FindByID(_ context.Context, orgID, storeID, id string) (*tables.Table, error) {
	t, ok := s.items[id]
	if !ok || t.OrganizationID != orgID || t.StoreID != storeID {
		return nil, tables.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (s *stubRepo) FindByStore(_ context.Context, orgID, storeID string, filter tables.Filter) ([]tables.Table, int, error) {
	var out []tables.Table
	for _, t := range s.items {
		if t.OrganizationID != orgID || t.StoreID != storeID {
			continue
		}
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		if filter.Area != "" && t.Area != filter.Area {
			continue
		}
		out = append(out, *t)
	}
	return out, len(out), nil
}

func (s *stubRepo) Update(_ context.Context, t *tables.Table) error {
	cur, ok := s.items[t.ID]
	if !ok || cur.OrganizationID != t.OrganizationID || cur.StoreID != t.StoreID {
		return tables.ErrNotFound
	}
	for _, other := range s.items {
		if other.ID != t.ID && other.StoreID == t.StoreID && other.Name == t.Name {
			return tables.ErrNameExists
		}
	}
	cp := *t
	s.items[t.ID] = &cp
	return nil
}

func (s *stubRepo) UpdateStatus(_ context.Context, orgID, storeID, id, status string) (*tables.Table, error) {
	t, ok := s.items[id]
	if !ok || t.OrganizationID != orgID || t.StoreID != storeID {
		return nil, tables.ErrNotFound
	}
	t.Status = status
	cp := *t
	return &cp, nil
}

func (s *stubRepo) Delete(_ context.Context, orgID, storeID, id string) error {
	t, ok := s.items[id]
	if !ok || t.OrganizationID != orgID || t.StoreID != storeID {
		return tables.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *stubRepo) ExistsByName(_ context.Context, storeID, name string, excludeID *string) (bool, error) {
	for _, t := range s.items {
		if t.StoreID != storeID || t.Name != name {
			continue
		}
		if excludeID != nil && t.ID == *excludeID {
			continue
		}
		return true, nil
	}
	return false, nil
}

type stubStores struct{}

func (stubStores) FindByIDInOrg(_ context.Context, orgID, id string) (*stores.Store, error) {
	if orgID == "org-a" && id == "store-a" {
		return &stores.Store{ID: id, OrganizationID: orgID}, nil
	}
	return nil, stores.ErrNotFound
}

func svc() (*tables.Service, *stubRepo) {
	repo := newStubRepo()
	return tables.NewService(repo, stubStores{}), repo
}

func create(t *testing.T, s *tables.Service, name string) *tables.Table {
	t.Helper()
	got, err := s.Create(context.Background(), "org-a", "store-a", tables.CreateRequest{Name: name, Area: "Indoor", Capacity: 4})
	if err != nil {
		t.Fatalf("Create(%q): %v", name, err)
	}
	return got
}

func TestService_CreateDefaults(t *testing.T) {
	s, _ := svc()

	got := create(t, s, "Meja 1")
	if got.Status != tables.StatusAvailable {
		t.Fatalf("status default harus available, got %q", got.Status)
	}
	if !got.IsActive {
		t.Fatalf("is_active default harus true")
	}
}

func TestService_CreateInvalidInput(t *testing.T) {
	s, _ := svc()
	ctx := context.Background()

	cases := []struct {
		name string
		req  tables.CreateRequest
		org  string
	}{
		{"nama kosong", tables.CreateRequest{Name: "  "}, "org-a"},
		{"org kosong", tables.CreateRequest{Name: "Meja"}, ""},
		{"capacity negatif", tables.CreateRequest{Name: "Meja", Capacity: -1}, "org-a"},
	}
	for _, c := range cases {
		if _, err := s.Create(ctx, c.org, "store-a", c.req); !errors.Is(err, tables.ErrInvalidInput) {
			t.Fatalf("%s harus ErrInvalidInput, got %v", c.name, err)
		}
	}
}

func TestService_CreateInvalidStatus(t *testing.T) {
	s, _ := svc()

	_, err := s.Create(context.Background(), "org-a", "store-a", tables.CreateRequest{Name: "Meja", Status: "dirty"})
	if !errors.Is(err, tables.ErrInvalidStatus) {
		t.Fatalf("status tak dikenal harus ErrInvalidStatus, got %v", err)
	}
}

func TestService_CreateDuplicateName(t *testing.T) {
	s, _ := svc()
	create(t, s, "Meja 1")

	_, err := s.Create(context.Background(), "org-a", "store-a", tables.CreateRequest{Name: "Meja 1"})
	if !errors.Is(err, tables.ErrNameExists) {
		t.Fatalf("nama duplikat harus ErrNameExists, got %v", err)
	}
}

func TestService_ListFilterStatus(t *testing.T) {
	s, _ := svc()
	ctx := context.Background()
	create(t, s, "Meja 1")
	create(t, s, "Meja 2")

	if _, err := s.UpdateStatus(ctx, "org-a", "store-a", "tbl-Meja 2", tables.StatusOccupied); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	busy, err := s.List(ctx, "org-a", "store-a", tables.Filter{Status: tables.StatusOccupied})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(busy) != 1 || busy[0].Name != "Meja 2" {
		t.Fatalf("filter occupied salah: %+v", busy)
	}

	if _, err := s.List(ctx, "org-a", "store-a", tables.Filter{Status: "dirty"}); !errors.Is(err, tables.ErrInvalidStatus) {
		t.Fatalf("filter status invalid harus ErrInvalidStatus, got %v", err)
	}
}

func TestService_UpdateKeepsStatusAndRejectsDupName(t *testing.T) {
	s, _ := svc()
	ctx := context.Background()
	first := create(t, s, "Meja 1")
	create(t, s, "Meja 2")

	if _, err := s.UpdateStatus(ctx, "org-a", "store-a", first.ID, tables.StatusOccupied); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, err := s.Update(ctx, "org-a", "store-a", first.ID, tables.UpdateRequest{Name: "Meja Utama", Area: "Outdoor", Capacity: 6})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Status != tables.StatusOccupied {
		t.Fatalf("update tidak boleh mengubah status, got %q", got.Status)
	}
	if got.Area != "Outdoor" || got.Capacity != 6 {
		t.Fatalf("field update tak tersimpan: %+v", got)
	}

	if _, err := s.Update(ctx, "org-a", "store-a", first.ID, tables.UpdateRequest{Name: "Meja 2", Capacity: 2}); !errors.Is(err, tables.ErrNameExists) {
		t.Fatalf("rename ke nama terpakai harus ErrNameExists, got %v", err)
	}
}

func TestService_CrossOrgAndStoreNotFound(t *testing.T) {
	s, _ := svc()
	ctx := context.Background()
	got := create(t, s, "Meja 1")

	if _, err := s.Get(ctx, "org-b", "store-a", got.ID); !errors.Is(err, tables.ErrNotFound) {
		t.Fatalf("org lain harus NotFound, got %v", err)
	}
	if _, err := s.Get(ctx, "org-a", "store-b", got.ID); !errors.Is(err, tables.ErrNotFound) {
		t.Fatalf("store lain harus NotFound, got %v", err)
	}
	if _, err := s.List(ctx, "org-a", "store-b", tables.Filter{}); !errors.Is(err, tables.ErrNotFound) {
		t.Fatalf("store tak dikenal harus NotFound, got %v", err)
	}
	if err := s.Delete(ctx, "org-b", "store-a", got.ID); !errors.Is(err, tables.ErrNotFound) {
		t.Fatalf("delete org lain harus NotFound, got %v", err)
	}
}

func TestService_UpdateStatusUnknownTable(t *testing.T) {
	s, _ := svc()

	if _, err := s.UpdateStatus(context.Background(), "org-a", "store-a", "tbl-x", tables.StatusOccupied); !errors.Is(err, tables.ErrNotFound) {
		t.Fatalf("meja tak ada harus NotFound, got %v", err)
	}
}
