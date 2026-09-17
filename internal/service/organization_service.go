package service

import (
	"context"
	"regexp"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/domain/organization"
	"github.com/ramadhanrzq/backend-go/internal/domain/rbac"
	"github.com/ramadhanrzq/backend-go/internal/domain/user"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type OrgService struct {
	orgRepo  organization.OrgRepository
	roleRepo rbac.RoleRepository
}

var _ organization.OrgService = (*OrgService)(nil)

func NewOrgService(orgRepo organization.OrgRepository, roleRepo rbac.RoleRepository) *OrgService {
	return &OrgService{
		orgRepo:  orgRepo,
		roleRepo: roleRepo,
	}
}

func (s *OrgService) Create(ctx context.Context, name, slug string, ownerID string) (*organization.Organization, error) {
	name = strings.TrimSpace(name)
	slug = strings.TrimSpace(strings.ToLower(slug))
	ownerID = strings.TrimSpace(ownerID)

	if name == "" || ownerID == "" {
		return nil, organization.ErrInvalidInput
	}

	if len(slug) < 3 || !slugRegex.MatchString(slug) {
		return nil, organization.ErrInvalidSlug
	}

	org := &organization.Organization{
		Name: name,
		Slug: slug,
	}

	if err := s.orgRepo.Create(ctx, org); err != nil {
		return nil, err
	}

	if err := s.orgRepo.AddMember(ctx, org.ID, ownerID); err != nil {
		return nil, err
	}

	// Owner gets admin role if available
	adminRole, err := s.roleRepo.FindByName(ctx, "admin")
	if err == nil && adminRole != nil {
		_ = s.roleRepo.AssignRoleToUser(ctx, ownerID, adminRole.ID)
	}

	return org, nil
}

func (s *OrgService) GetBySlug(ctx context.Context, slug string) (*organization.Organization, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return nil, organization.ErrInvalidInput
	}

	return s.orgRepo.FindBySlug(ctx, slug)
}

func (s *OrgService) List(ctx context.Context) ([]organization.Organization, error) {
	return s.orgRepo.List(ctx)
}

func (s *OrgService) AddMember(ctx context.Context, orgID, userID string) error {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)

	if orgID == "" || userID == "" {
		return organization.ErrInvalidInput
	}

	isMember, err := s.orgRepo.IsMember(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if isMember {
		return organization.ErrAlreadyMember
	}

	return s.orgRepo.AddMember(ctx, orgID, userID)
}

func (s *OrgService) RemoveMember(ctx context.Context, orgID, userID string) error {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)

	if orgID == "" || userID == "" {
		return organization.ErrInvalidInput
	}

	isMember, err := s.orgRepo.IsMember(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return organization.ErrNotMember
	}

	return s.orgRepo.RemoveMember(ctx, orgID, userID)
}

func (s *OrgService) ListMembers(ctx context.Context, orgID string) ([]user.User, error) {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return nil, organization.ErrInvalidInput
	}

	return s.orgRepo.ListMembers(ctx, orgID)
}

func (s *OrgService) IsUserMember(ctx context.Context, orgID, userID string) (bool, error) {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)

	if orgID == "" || userID == "" {
		return false, organization.ErrInvalidInput
	}

	return s.orgRepo.IsMember(ctx, orgID, userID)
}
