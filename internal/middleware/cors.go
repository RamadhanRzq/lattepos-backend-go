package middleware

import "net/http"

// CORS menambahkan header Cross-Origin Resource Sharing.
// Mengizinkan semua origin untuk development; batasi via env jika perlu.
// ponytail: hardcoded allow-all; tambahkan configurable allowed-origins saat deploy production.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
