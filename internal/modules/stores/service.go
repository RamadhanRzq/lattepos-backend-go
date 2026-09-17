package stores

import (
	"context"
	"regexp"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/modules/users"
)

// codeRegex longgar seperti slug tapi mengizinkan huruf besar dan underscore:
// JKT-01, jakarta_pusat valid; spasi dan simbol lain ditolak.
var codeRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Service menangani business rules store dan akses user-store.
// orgs adalah port keanggotaan organisasi untuk validasi same-org;
// findUser dipakai AssignUser untuk memastikan user exists sebelum assignment.
type Service struct {
	repo     Repository
	orgs     OrgChecker
	findUser func(ctx context.Context, id string) (*users.User, error)
}

func NewService(repo Repository, orgs OrgChecker, findUser func(ctx context.Context, id string) (*users.User, error)) *Service {
	return &Service{repo: repo, orgs: orgs, findUser: findUser}
}

func (s *Service) Create(ctx context.Context, orgID, name, code, address, phone string) (*Store, error) {
	name, code, address, phone = normalizeStoreInput(name, code, address, phone)
	if orgID = strings.TrimSpace(orgID); orgID == "" {
		return nil, ErrInvalidInput
	}
	if name == "" {
		return nil, ErrInvalidInput
	}
	if !codeRegex.MatchString(code) {
		return nil, ErrInvalidCode
	}

	store := &Store{
		OrganizationID: orgID,
		Name:           name,
		Code:           code,
		Address:        address,
		Phone:          phone,
	}
	if err := s.repo.Create(ctx, store); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Service) Get(ctx context.Context, orgID, id string) (*Store, error) {
	orgID = strings.TrimSpace(orgID)
	id = strings.TrimSpace(id)
	if orgID == "" || id == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.FindByIDInOrg(ctx, orgID, id)
}

func (s *Service) List(ctx context.Context, orgID string) ([]Store, error) {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.ListByOrg(ctx, orgID)
}

// Update full replace name/code/address/phone dalam organisasi yang sama.
// organization_id tidak pernah berubah: kolom itu tidak ditulis repository.
func (s *Service) Update(ctx context.Context, orgID, id, name, code, address, phone string) (*Store, error) {
	name, code, address, phone = normalizeStoreInput(name, code, address, phone)
	orgID = strings.TrimSpace(orgID)
	id = strings.TrimSpace(id)
	if orgID == "" || id == "" {
		return nil, ErrInvalidInput
	}
	if name == "" {
		return nil, ErrInvalidInput
	}
	if !codeRegex.MatchString(code) {
		return nil, ErrInvalidCode
	}

	store := &Store{
		ID:             id,
		OrganizationID: orgID,
		Name:           name,
		Code:           code,
		Address:        address,
		Phone:          phone,
	}
	if err := s.repo.Update(ctx, store); err != nil {
		return nil, err
	}
	return store, nil
}

// SetActive mengaktifkan/menonaktifkan store tanpa menghapus datanya.
func (s *Service) SetActive(ctx context.Context, orgID, id string, active bool) (*Store, error) {
	orgID = strings.TrimSpace(orgID)
	id = strings.TrimSpace(id)
	if orgID == "" || id == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.SetActive(ctx, orgID, id, active)
}

// AssignUser memberi user akses ke store. Semua check berjalan dalam orgID
// dari middleware context: store harus milik org itu, user harus member org
// itu. Urutan: store exists dulu (NotFound menutupi cross-org), lalu user
// exists, lalu same-org via membership, lalu duplicate.
func (s *Service) AssignUser(ctx context.Context, orgID, storeID, userID string) error {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	userID = strings.TrimSpace(userID)
	if orgID == "" || storeID == "" || userID == "" {
		return ErrInvalidInput
	}

	if _, err := s.repo.FindByIDInOrg(ctx, orgID, storeID); err != nil {
		return err
	}

	u, err := s.findUser(ctx, userID)
	if err != nil || u == nil {
		return ErrUserNotFound
	}

	isMember, err := s.orgs.IsMember(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrCrossOrg
	}

	assigned, err := s.repo.IsAssigned(ctx, userID, storeID)
	if err != nil {
		return err
	}
	if assigned {
		return ErrAlreadyAssigned
	}

	return s.repo.AssignUser(ctx, userID, storeID)
}

func (s *Service) RemoveUser(ctx context.Context, orgID, storeID, userID string) error {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	userID = strings.TrimSpace(userID)
	if orgID == "" || storeID == "" || userID == "" {
		return ErrInvalidInput
	}

	if _, err := s.repo.FindByIDInOrg(ctx, orgID, storeID); err != nil {
		return err
	}

	return s.repo.RemoveUser(ctx, userID, storeID)
}

func (s *Service) ListUserStores(ctx context.Context, orgID, userID string) ([]Store, error) {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)
	if orgID == "" || userID == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.ListUserStores(ctx, orgID, userID)
}

func (s *Service) ListStoreUsers(ctx context.Context, orgID, storeID string) ([]users.User, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	if orgID == "" || storeID == "" {
		return nil, ErrInvalidInput
	}

	if _, err := s.repo.FindByIDInOrg(ctx, orgID, storeID); err != nil {
		return nil, err
	}

	return s.repo.ListStoreUsers(ctx, orgID, storeID)
}

func normalizeStoreInput(name, code, address, phone string) (string, string, string, string) {
	return strings.TrimSpace(name), strings.TrimSpace(code),
		strings.TrimSpace(address), strings.TrimSpace(phone)
}
