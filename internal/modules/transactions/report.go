package transactions

import (
	"context"
	"strings"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/modules/sales"
)

// ctx adalah alias context.Context supaya signature port tetap pendek.
type ctx = context.Context

func trim(s string) string { return strings.TrimSpace(s) }

// DailyReport adalah ringkasan transaksi satu hari kalender per store.
type DailyReport struct {
	Date  string `json:"date"`
	Store string `json:"store_id"`
	Count int    `json:"count"`
	// Total rupiah per status; cancelled ikut dihitung terpisah supaya
	// laporan kasir bisa rekonsiliasi omzet vs batal.
	GrandTotal int64            `json:"grand_total"`
	ByStatus   map[string]int64 `json:"by_status"`
	ByPayment  map[string]int64 `json:"by_payment"`
	ByHour     map[string]int64 `json:"by_hour"`
	// Items agregat per produk: qty + omzet.
	Items []DailyItem `json:"items"`
}

// DailyItem adalah agregat satu produk dalam laporan harian.
type DailyItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Subtotal  int64  `json:"subtotal"`
}

// Summarize menghitung ringkasan dari list sale mentah.
func Summarize(day time.Time, list []sales.Sale) *DailyReport {
	r := &DailyReport{
		Date:      day.Format("2006-01-02"),
		ByStatus:  map[string]int64{},
		ByPayment: map[string]int64{},
		ByHour:    map[string]int64{},
		Items:     []DailyItem{},
	}
	byItem := map[string]*DailyItem{}
	for _, s := range list {
		if r.Store == "" {
			r.Store = s.StoreID
		}
		r.Count++
		r.GrandTotal += s.GrandTotal
		r.ByStatus[s.Status] += s.GrandTotal
		r.ByPayment[s.PaymentMethod] += s.GrandTotal
		r.ByHour[s.CreatedAt.Format("15")] += s.GrandTotal
		for _, it := range s.Items {
			ag, ok := byItem[it.ProductID]
			if !ok {
				ag = &DailyItem{ProductID: it.ProductID}
				byItem[it.ProductID] = ag
			}
			ag.Quantity += it.Quantity
			ag.Subtotal += it.Subtotal
		}
	}
	for _, ag := range byItem {
		r.Items = append(r.Items, *ag)
	}
	return r
}

// parseDay menerima YYYY-MM-DD; kosong berarti hari ini (zona server).
func parseDay(date string) (time.Time, error) {
	date = strings.TrimSpace(date)
	if date == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	}
	return time.Parse("2006-01-02", date)
}
