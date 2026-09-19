package middleware

import (
	"net/http"
	"strings"
)

// CORSConfig menyimpan daftar origin yang diizinkan.
type CORSConfig struct {
	AllowedOrigins []string
}

// NewCORSConfig mem-parse string origin yang dipisah koma.
func NewCORSConfig(origins string) CORSConfig {
	origins = strings.TrimSpace(origins)
	if origins == "" || origins == "*" {
		return CORSConfig{AllowedOrigins: []string{"*"}}
	}

	parts := strings.Split(origins, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if o := strings.TrimSpace(p); o != "" {
			out = append(out, o)
		}
	}
	return CORSConfig{AllowedOrigins: out}
}

// CORS menambahkan header Cross-Origin Resource Sharing.
func CORS(cfg CORSConfig, next http.Handler) http.Handler {
	wildcard := len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*"

	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		allowed[o] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if wildcard {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if _, ok := allowed[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
