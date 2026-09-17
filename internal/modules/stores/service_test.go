package stores_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
)

// stubRepo mengimplementasikan stores.Repository di memori per org.
type stubRepo struct {
	stores.Repository
	org       string
	items     map[string]*stores.Store
	assigned  map[string]bool
	findErr   error
	updateErr error
}

func newStubRepo(org string) *stubRepo {
	return &stubRepo{org: org, items: map[string]*stores.Store{}, assigned: map[string]bool{}}
}

func key(userID, storeID string) string { return userID + "|" + storeID }

func (s *stubRepo) Create(_ context.Context, st *stores.Store) error {
	if st.Code == "TAKEN" {
		return stores.ErrCodeTaken
	}
	st.ID = "store-" + st.Code
	st.IsActive = true
	cp := *st
	s.items[st.ID] = &cp
	return nil
}

func (s *stubRepo) FindByIDInOrg(_ context.Context, orgID, id string) (*stores.Store, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	st, ok := s.items[id]
	if !ok || orgID != s.org || st.OrganizationID != orgID {
		return nil, stores.ErrNotFound
	}
	cp := *st
	return &cp, nil
}

func (s *stubRepo) ListByOrg(_ context.Context, orgID string) ([]stores.Store, error) {
	var out []stores.Store
	for _, st := range s.items {
		if st.OrganizationID == orgID {
			out = append(out, *st)
		}
	}
	return out, nil
}

func (s *stubRepo) Update(_ context.Context, st *stores.Store) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	cur, ok := s.items[st.ID]
	if !ok || cur.OrganizationID != st.OrganizationID {
		return stores.ErrNotFound
	}
	*cur = *st
	return nil
}

func (s *stubRepo) SetActive(_ context.Context, orgID, id string, active bool) (*stores.Store, error) {
	st, ok := s.items[id]
	if !ok || orgID != s.org || st.OrganizationID != orgID {
		return nil, stores.ErrNotFound
	}
	st.IsActive = active
	cp := *st
	return &cp, nil
}

func (s *stubRepo) AssignUser(_ context.Context, userID, storeID string) error {
	if s.assigned[key(userID, storeID)] {
		return stores.ErrAlreadyAssigned
	}
	s.assigned[key(userID, storeID)] = true
	return nil
}

func (s *stubRepo) IsAssigned(_ context.Context, userID, storeID string) (bool, error) {
	return s.assigned[key(userID, storeID)], nil
}

func (s *stubRepo) RemoveUser(_ context.Context, userID, storeID string) error {
	if !s.assigned[key(userID, storeID)] {
		return stores.ErrNotAssigned
	}
	delete(s.assigned, key(userID, storeID))
	return nil
}

func (s *stubRepo) ListUserStores(_ context.Context, orgID, userID string) ([]stores.Store, error) {
	var out []stores.Store
	for _, st := range s.items {
		if st.OrganizationID == orgID && s.assigned[key(userID, st.ID)] {
			out = append(out, *st)
		}
	}
	return out, nil
}

// stubOrgs: hanya user-a member org-a.
type stubOrgs struct{ member bool }

func (s stubOrgs) IsMember(_ context.Context, orgID, userID string) (bool, error) {
	if orgID == "org-a" && userID == "user-a" {
		return true, nil
	}
	return false, nil
}

func findUserOK(_ context.Context, id string) (*users.User, error) {
	if id == "ghost" {
		return nil, users.ErrNotFound
	}
	return &users.User{ID: id}, nil
}

func svc(org string) (*stores.Service, *stubRepo) {
	repo := newStubRepo(org)
	return stores.NewService(repo, stubOrgs{}, findUserOK), repo
}

func TestService_Create(t *testing.T) {
	s, _ := svc("org-a")

	got, err := s.Create(context.Background(), "org-a", "Jakarta", "JKT-01", "", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.OrganizationID != "org-a" || !got.IsActive {
		t.Fatalf("Create boundary/active: %+v", got)
	}

	if _, err := s.Create(context.Background(), "org-a", "", "X", "", ""); !errors.Is(err, stores.ErrInvalidInput) {
		t.Fatalf("name kosong harus ErrInvalidInput, got %v", err)
	}
	if _, err := s.Create(context.Background(), "org-a", "N", "bad code!", "", ""); !errors.Is(err, stores.ErrInvalidCode) {
		t.Fatalf("code invalid harus ErrInvalidCode, got %v", err)
	}
	if _, err := s.Create(context.Background(), "org-a", "N", "TAKEN", "", ""); !errors.Is(err, stores.ErrCodeTaken) {
		t.Fatalf("code duplikat harus ErrCodeTaken, got %v", err)
	}
}

func TestService_TenantIsolation(t *testing.T) {
	s, _ := svc("org-a")
	ctx := context.Background()

	st, err := s.Create(ctx, "org-a", "Jakarta", "JKT", "", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := s.Get(ctx, "org-b", st.ID); !errors.Is(err, stores.ErrNotFound) {
		t.Fatalf("store org lain harus NotFound, got %v", err)
	}

	list, err := s.List(ctx, "org-b")
	if err != nil || len(list) != 0 {
		t.Fatalf("list org lain harus kosong, got %v %+v", err, list)
	}
}

func TestService_Activate(t *testing.T) {
	s, _ := svc("org-a")
	ctx := context.Background()

	st, _ := s.Create(ctx, "org-a", "Jakarta", "JKT", "", "")

	off, err := s.SetActive(ctx, "org-a", st.ID, false)
	if err != nil || off.IsActive {
		t.Fatalf("deactivate: %v %+v", err, off)
	}
	on, err := s.SetActive(ctx, "org-a", st.ID, true)
	if err != nil || !on.IsActive {
		t.Fatalf("activate: %v %+v", err, on)
	}
	if _, err := s.SetActive(ctx, "org-b", st.ID, false); !errors.Is(err, stores.ErrNotFound) {
		t.Fatalf("status org lain harus NotFound, got %v", err)
	}
}

func TestService_Update(t *testing.T) {
	s, repo := svc("org-a")
	ctx := context.Background()

	st, _ := s.Create(ctx, "org-a", "Jakarta", "JKT", "", "")

	// Store tidak boleh pindah org: Update selalu pakai orgID boundary,
	// repository mock menolak ID yang organization_id-nya beda.
	got, err := s.Update(ctx, "org-a", st.ID, "Jakarta Baru", "JKT2", "Jl. Sudirman", "021")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Name != "Jakarta Baru" || got.OrganizationID != "org-a" {
		t.Fatalf("Update pindah org: %+v", got)
	}
	if repo.items[st.ID].OrganizationID != "org-a" {
		t.Fatalf("organization_id berubah di repo")
	}

	if _, err := s.Update(ctx, "org-b", st.ID, "X", "X", "", ""); !errors.Is(err, stores.ErrNotFound) {
		t.Fatalf("update org lain harus NotFound, got %v", err)
	}
}

func TestService_Assign(t *testing.T) {
	s, _ := svc("org-a")
	ctx := context.Background()

	st, _ := s.Create(ctx, "org-a", "Jakarta", "JKT", "", "")

	if err := s.AssignUser(ctx, "org-a", st.ID, "user-a"); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if err := s.AssignUser(ctx, "org-a", st.ID, "user-a"); !errors.Is(err, stores.ErrAlreadyAssigned) {
		t.Fatalf("duplicate harus ErrAlreadyAssigned, got %v", err)
	}
	if err := s.AssignUser(ctx, "org-a", st.ID, "user-b"); !errors.Is(err, stores.ErrCrossOrg) {
		t.Fatalf("user org lain harus ErrCrossOrg, got %v", err)
	}
	if err := s.AssignUser(ctx, "org-a", st.ID, "ghost"); !errors.Is(err, stores.ErrUserNotFound) {
		t.Fatalf("user hilang harus ErrUserNotFound, got %v", err)
	}
	if err := s.AssignUser(ctx, "org-a", "no-store", "user-a"); !errors.Is(err, stores.ErrNotFound) {
		t.Fatalf("store hilang harus ErrNotFound, got %v", err)
	}

	if err := s.RemoveUser(ctx, "org-a", st.ID, "user-a"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := s.RemoveUser(ctx, "org-a", st.ID, "user-a"); !errors.Is(err, stores.ErrNotAssigned) {
		t.Fatalf("remove ulang harus ErrNotAssigned, got %v", err)
	}

	if err := s.AssignUser(ctx, "org-a", st.ID, "user-a"); err != nil {
		t.Fatalf("Assign ulang: %v", err)
	}
	list, err := s.ListUserStores(ctx, "org-a", "user-a")
	if err != nil || len(list) != 1 {
		t.Fatalf("user punya 1 store, got %v %+v", err, list)
	}
	other, err := s.ListUserStores(ctx, "org-b", "user-a")
	if err != nil || len(other) != 0 {
		t.Fatalf("list lintas org harus kosong, got %v %+v", err, other)
	}
}
