package organizations

import (
	"context"
	"regexp"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/modules/rbac"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Service menangani business rules organization dan keanggotaannya.
type Service struct {
	repo     Repository
	roleRepo rbac.RoleRepository
}

func NewService(repo Repository, roleRepo rbac.RoleRepository) *Service {
	return &Service{
		repo:     repo,
		roleRepo: roleRepo,
	}
}

func (s *Service) Create(ctx context.Context, name, slug string, ownerID string) (*Organization, error) {
	name = strings.TrimSpace(name)
	slug = strings.TrimSpace(strings.ToLower(slug))
	ownerID = strings.TrimSpace(ownerID)

	if name == "" || ownerID == "" {
		return nil, ErrInvalidInput
	}

	if len(slug) < 3 || !slugRegex.MatchString(slug) {
		return nil, ErrInvalidSlug
	}

	org := &Organization{
		Name: name,
		Slug: slug,
	}

	if err := s.repo.Create(ctx, org); err != nil {
		return nil, err
	}

	if err := s.repo.AddMember(ctx, org.ID, ownerID); err != nil {
		return nil, err
	}

	// Owner mendapat role admin pada organisasi ini (role milik organisasi
	// tersebut bila ada, kalau tidak jatuh ke role admin global).
	adminRole, err := s.roleRepo.FindByName(ctx, "admin", org.ID)
	if err == nil && adminRole != nil {
		_ = s.roleRepo.AssignRoleToUser(ctx, org.ID, ownerID, adminRole.ID)
	}

	return org, nil
}

func (s *Service) GetBySlug(ctx context.Context, slug string) (*Organization, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return nil, ErrInvalidInput
	}

	return s.repo.FindBySlug(ctx, slug)
}

// ResolveOrgID mengembalikan ID organization dari slug-nya. Dipakai middleware
// supaya package middleware tidak perlu mengimpor module ini.
func (s *Service) ResolveOrgID(ctx context.Context, slug string) (string, error) {
	org, err := s.GetBySlug(ctx, slug)
	if err != nil {
		return "", err
	}
	return org.ID, nil
}

func (s *Service) List(ctx context.Context) ([]Organization, error) {
	return s.repo.List(ctx)
}

func (s *Service) AddMember(ctx context.Context, orgID, userID string) error {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)

	if orgID == "" || userID == "" {
		return ErrInvalidInput
	}

	isMember, err := s.repo.IsMember(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if isMember {
		return ErrAlreadyMember
	}

	return s.repo.AddMember(ctx, orgID, userID)
}

func (s *Service) RemoveMember(ctx context.Context, orgID, userID string) error {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)

	if orgID == "" || userID == "" {
		return ErrInvalidInput
	}

	isMember, err := s.repo.IsMember(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrNotMember
	}

	return s.repo.RemoveMember(ctx, orgID, userID)
}

func (s *Service) ListMembers(ctx context.Context, orgID string) ([]users.User, error) {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return nil, ErrInvalidInput
	}

	return s.repo.ListMembers(ctx, orgID)
}

func (s *Service) IsUserMember(ctx context.Context, orgID, userID string) (bool, error) {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)

	if orgID == "" || userID == "" {
		return false, ErrInvalidInput
	}

	return s.repo.IsMember(ctx, orgID, userID)
}
