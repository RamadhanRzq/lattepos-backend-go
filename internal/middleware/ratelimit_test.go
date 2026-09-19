package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("allows within limit", func(t *testing.T) {
		rl := NewRateLimiter(3, time.Minute)
		wrapped := rl.Wrap(handler)

		for i := 0; i < 3; i++ {
			r := httptest.NewRequest(http.MethodPost, "/login", nil)
			r.RemoteAddr = "1.2.3.4:1234"
			w := httptest.NewRecorder()
			wrapped(w, r)

			if w.Code != http.StatusOK {
				t.Fatalf("request %d: want 200, got %d", i, w.Code)
			}
		}
	})

	t.Run("blocks over limit", func(t *testing.T) {
		rl := NewRateLimiter(2, time.Minute)
		wrapped := rl.Wrap(handler)

		for i := 0; i < 2; i++ {
			r := httptest.NewRequest(http.MethodPost, "/login", nil)
			r.RemoteAddr = "5.6.7.8:1234"
			w := httptest.NewRecorder()
			wrapped(w, r)
		}

		r := httptest.NewRequest(http.MethodPost, "/login", nil)
		r.RemoteAddr = "5.6.7.8:1234"
		w := httptest.NewRecorder()
		wrapped(w, r)

		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("want 429, got %d", w.Code)
		}
	})

	t.Run("different IPs independent", func(t *testing.T) {
		rl := NewRateLimiter(1, time.Minute)
		wrapped := rl.Wrap(handler)

		r1 := httptest.NewRequest(http.MethodPost, "/login", nil)
		r1.RemoteAddr = "10.0.0.1:1234"
		w1 := httptest.NewRecorder()
		wrapped(w1, r1)

		r2 := httptest.NewRequest(http.MethodPost, "/login", nil)
		r2.RemoteAddr = "10.0.0.2:1234"
		w2 := httptest.NewRecorder()
		wrapped(w2, r2)

		if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
			t.Fatalf("different IPs should be independent: %d, %d", w1.Code, w2.Code)
		}
	})
}
