package transactions

import (
	"errors"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/sales"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Error domain laporan harian.
var (
	ErrInvalidInput  = errors.New("transactions: invalid input")
	ErrStoreNotFound = errors.New("transactions: store not found")
	ErrNoAccess      = errors.New("transactions: no store access")
)

// SaleSource adalah port baca sale untuk agregasi.
// Dipenuhi struktural oleh sales.Service (List dengan Filter tanggal).
// skipped: query agregasi SQL, add when volume butuh COUNT/SUM di DB.
type SaleSource interface {
	List(ctx ctx, orgID, storeID, userID string, filter sales.Filter) ([]sales.Sale, int, error)
}

// StoreChecker adalah port kepemilikan dan akses store.
// Dipenuhi struktural oleh stores repository (FindByIDInOrg + IsAssigned).
type StoreChecker interface {
	FindByIDInOrg(ctx ctx, orgID, id string) (*stores.Store, error)
	IsAssigned(ctx ctx, userID, storeID string) (bool, error)
}

// Service mengagregasi laporan harian dari sale dalam satu store.
type Service struct {
	sales  SaleSource
	stores StoreChecker
}

func NewService(sales SaleSource, stores StoreChecker) *Service {
	return &Service{sales: sales, stores: stores}
}

// Daily menarik sale satu hari kalender (date YYYY-MM-DD, zona server),
// lalu menghitung ringkasan. Default date = hari ini bila kosong.
func (s *Service) Daily(c ctx, orgID, storeID, userID, date string) (*DailyReport, error) {
	day, err := parseDay(date)
	if err != nil {
		return nil, ErrInvalidInput
	}
	if err := s.authorize(c, orgID, storeID, userID); err != nil {
		return nil, err
	}
	from, to := dayBounds(day)

	list, _, err := s.sales.List(c, orgID, storeID, userID, sales.Filter{From: from, To: to, Page: 1, Limit: 100})
	if err != nil {
		if errors.Is(err, sales.ErrStoreNotFound) {
			return nil, ErrStoreNotFound
		}
		if errors.Is(err, sales.ErrNoAccess) {
			return nil, ErrNoAccess
		}
		return nil, err
	}
	return Summarize(day, list), nil
}

func (s *Service) authorize(c ctx, orgID, storeID, userID string) error {
	orgID = trim(orgID)
	storeID = trim(storeID)
	userID = trim(userID)
	if orgID == "" || storeID == "" || userID == "" {
		return ErrInvalidInput
	}
	if _, err := s.stores.FindByIDInOrg(c, orgID, storeID); err != nil {
		if errors.Is(err, stores.ErrNotFound) {
			return ErrStoreNotFound
		}
		return err
	}
	ok, err := s.stores.IsAssigned(c, userID, storeID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoAccess
	}
	return nil
}

// dayBounds mengembalikan [00:00:00, 23:59:59.999] hari itu.
func dayBounds(day time.Time) (time.Time, time.Time) {
	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	return from, from.Add(24*time.Hour - time.Millisecond)
}
