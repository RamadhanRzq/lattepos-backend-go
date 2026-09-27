package tables

import (
	"context"
	"errors"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Status valid table. Occupancy dikelola manual dari UI meja (PATCH status);
// tidak ada auto-occupy dari sale — tambahkan hook di sales.SideEffects bila
// POS perlu mengunci meja otomatis saat checkout.
const (
	StatusAvailable = "available"
	StatusOccupied  = "occupied"
	StatusReserved  = "reserved"
)

// Error domain module tables.
var (
	ErrNotFound      = errors.New("table: not found")
	ErrInvalidInput  = errors.New("table: invalid input")
	ErrInvalidStatus = errors.New("table: invalid status")
	ErrNameExists    = errors.New("table: name already exists in this store")
)

// Table adalah meja fisik dalam satu store.
type Table struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	StoreID        string    `json:"store_id"`
	Name           string    `json:"name"`
	Area           string    `json:"area,omitempty"`
	Capacity       int       `json:"capacity"`
	Status         string    `json:"status"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Repository adalah kontrak penyimpanan table.
// Semua query ter-scope org+store; cross-org ditutupi sebagai NotFound.
type Repository interface {
	Create(ctx context.Context, t *Table) error
	FindByID(ctx context.Context, orgID, storeID, id string) (*Table, error)
	FindByStore(ctx context.Context, orgID, storeID string, filter Filter) ([]Table, int, error)
	Update(ctx context.Context, t *Table) error
	UpdateStatus(ctx context.Context, orgID, storeID, id, status string) (*Table, error)
	Delete(ctx context.Context, orgID, storeID, id string) error
	ExistsByName(ctx context.Context, storeID, name string, excludeID *string) (bool, error)
}

// Filter adalah kriteria list table.
type Filter struct {
	Status string
	Area   string
}

// StoreChecker adalah port kepemilikan store untuk validasi same-org.
// Dipenuhi struktural oleh stores repository (method FindByIDInOrg).
type StoreChecker interface {
	FindByIDInOrg(ctx context.Context, orgID, id string) (*stores.Store, error)
}
