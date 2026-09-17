package stores

import (
	"context"
	"errors"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/users"
)

// Error domain module stores.
var (
	ErrNotFound        = errors.New("store: not found")
	ErrInvalidInput    = errors.New("store: invalid input")
	ErrInvalidCode     = errors.New("store: invalid code format")
	ErrCodeTaken       = errors.New("store: code already taken")
	ErrAlreadyAssigned = errors.New("store: user already assigned")
	ErrNotAssigned     = errors.New("store: user not assigned to store")
	ErrUserNotFound    = errors.New("store: user not found")
	ErrCrossOrg        = errors.New("store: user and store must belong to the same organization")
)

// Store adalah outlet fisik milik satu Organization.
type Store struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Code           string    `json:"code"`
	Address        string    `json:"address,omitempty"`
	Phone          string    `json:"phone,omitempty"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UserStore adalah akses satu user ke satu store (pivot user_stores).
type UserStore struct {
	UserID    string    `json:"user_id"`
	StoreID   string    `json:"store_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Repository adalah kontrak penyimpanan store dan akses user-store.
type Repository interface {
	Create(ctx context.Context, s *Store) error
	// FindByIDInOrg tidak membocorkan store organisasi lain: di luar org = NotFound.
	FindByIDInOrg(ctx context.Context, orgID, id string) (*Store, error)
	ListByOrg(ctx context.Context, orgID string) ([]Store, error)
	// Update tidak boleh memindahkan store antar organization (kolom
	// organization_id tidak pernah ditulis).
	Update(ctx context.Context, s *Store) error
	SetActive(ctx context.Context, orgID, id string, active bool) (*Store, error)
	AssignUser(ctx context.Context, userID, storeID string) error
	IsAssigned(ctx context.Context, userID, storeID string) (bool, error)
	RemoveUser(ctx context.Context, userID, storeID string) error
	ListUserStores(ctx context.Context, orgID, userID string) ([]Store, error)
	ListStoreUsers(ctx context.Context, orgID, storeID string) ([]users.User, error)
}

// OrgChecker adalah port keanggotaan organisasi untuk validasi same-org.
// Dipenuhi struktural oleh organizations repository (method IsMember).
type OrgChecker interface {
	IsMember(ctx context.Context, orgID, userID string) (bool, error)
}
