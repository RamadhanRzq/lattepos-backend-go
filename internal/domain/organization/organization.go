package organization

import (
	"context"
	"errors"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/domain/user"
)

var (
	ErrNotFound       = errors.New("organization: not found")
	ErrSlugTaken      = errors.New("organization: slug already taken")
	ErrInvalidInput   = errors.New("organization: invalid input")
	ErrNotMember      = errors.New("organization: user is not a member")
	ErrAlreadyMember  = errors.New("organization: user is already a member")
	ErrInvalidSlug    = errors.New("organization: invalid slug format")
)

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

type Member struct {
	ID       string    `json:"id"`
	OrgID    string    `json:"org_id"`
	UserID   string    `json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}

type OrgRepository interface {
	Create(ctx context.Context, org *Organization) error
	FindByID(ctx context.Context, id string) (*Organization, error)
	FindBySlug(ctx context.Context, slug string) (*Organization, error)
	List(ctx context.Context) ([]Organization, error)
	AddMember(ctx context.Context, orgID, userID string) error
	RemoveMember(ctx context.Context, orgID, userID string) error
	IsMember(ctx context.Context, orgID, userID string) (bool, error)
	ListMembers(ctx context.Context, orgID string) ([]user.User, error)
}

type OrgService interface {
	Create(ctx context.Context, name, slug string, ownerID string) (*Organization, error)
	GetBySlug(ctx context.Context, slug string) (*Organization, error)
	List(ctx context.Context) ([]Organization, error)
	AddMember(ctx context.Context, orgID, userID string) error
	RemoveMember(ctx context.Context, orgID, userID string) error
	ListMembers(ctx context.Context, orgID string) ([]user.User, error)
	IsUserMember(ctx context.Context, orgID, userID string) (bool, error)
}
