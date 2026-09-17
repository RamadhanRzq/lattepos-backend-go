package organizations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ramadhanrzq/backend-go/internal/modules/organizations"
	"github.com/ramadhanrzq/backend-go/internal/modules/rbac"
)

// stubOrgRepo mengimplementasikan hanya method yang dipakai Service.Create;
// sisanya diwarisi dari interface (nil) supaya tidak ada boilerplate.
type stubOrgRepo struct {
	organizations.Repository
	createFn    func(ctx context.Context, org *organizations.Organization) error
	addMemberFn func(ctx context.Context, orgID, userID string) error
}

func (s stubOrgRepo) Create(ctx context.Context, org *organizations.Organization) error {
	return s.createFn(ctx, org)
}

func (s stubOrgRepo) AddMember(ctx context.Context, orgID, userID string) error {
	return s.addMemberFn(ctx, orgID, userID)
}

// stubRoleRepo selalu mengembalikan ErrRoleNotFound: tidak ada role admin untuk ditugaskan.
type stubRoleRepo struct {
	rbac.RoleRepository
}

func (s stubRoleRepo) FindByName(ctx context.Context, name, orgID string) (*rbac.Role, error) {
	return nil, rbac.ErrRoleNotFound
}

func TestService_Create(t *testing.T) {
	tests := []struct {
		name       string
		orgName    string
		slug       string
		ownerID    string
		createErr  error
		wantErr    error
		wantSlug   string
		wantMember string
	}{
		{
			name:       "slug dinormalisasi dan owner jadi member",
			orgName:    "LattePOS Central",
			slug:       "  LattePOS-Central  ",
			ownerID:    "owner-1",
			wantSlug:   "lattepos-central",
			wantMember: "owner-1",
		},
		{
			name:    "slug kurang dari 3 karakter ditolak",
			orgName: "Tiny",
			slug:    "ab",
			ownerID: "owner-1",
			wantErr: organizations.ErrInvalidSlug,
		},
		{
			name:    "slug dengan karakter tidak valid ditolak",
			orgName: "Bad Slug",
			slug:    "bad slug!",
			ownerID: "owner-1",
			wantErr: organizations.ErrInvalidSlug,
		},
		{
			name:    "name kosong ditolak",
			orgName: "   ",
			slug:    "valid-slug",
			ownerID: "owner-1",
			wantErr: organizations.ErrInvalidInput,
		},
		{
			name:    "owner kosong ditolak",
			orgName: "Valid",
			slug:    "valid-slug",
			ownerID: "   ",
			wantErr: organizations.ErrInvalidInput,
		},
		{
			name:      "slug yang sudah dipakai diteruskan sebagai error domain",
			orgName:   "Duplicate",
			slug:      "taken-slug",
			ownerID:   "owner-1",
			createErr: organizations.ErrSlugTaken,
			wantErr:   organizations.ErrSlugTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var memberOf string
			repo := stubOrgRepo{
				createFn: func(ctx context.Context, org *organizations.Organization) error {
					if tt.createErr != nil {
						return tt.createErr
					}
					org.ID = "org-1"
					return nil
				},
				addMemberFn: func(ctx context.Context, orgID, userID string) error {
					memberOf = userID
					return nil
				},
			}
			svc := organizations.NewService(repo, stubRoleRepo{})

			org, err := svc.Create(context.Background(), tt.orgName, tt.slug, tt.ownerID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				if memberOf != "" {
					t.Fatalf("owner tidak boleh ditambahkan saat input invalid, dapat %q", memberOf)
				}
				return
			}
			if org.Slug != tt.wantSlug {
				t.Fatalf("expected slug %q, got %q", tt.wantSlug, org.Slug)
			}
			if memberOf != tt.wantMember {
				t.Fatalf("expected owner %q menjadi member, got %q", tt.wantMember, memberOf)
			}
		})
	}
}
