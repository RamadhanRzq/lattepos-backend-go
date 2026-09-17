package response

import (
	"encoding/json"
	"net/http"
)

// JSON menulis data sebagai body JSON dengan status code yang diberikan.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(data)
}

// Error menulis pesan error ke client.
// Saat ini perilakunya identik dengan http.Error (plain text) supaya
// tidak ada perubahan pada kontrak API; kalau nanti mau error JSON,
// cukup ubah di sini.
func Error(w http.ResponseWriter, status int, message string) {
	http.Error(w, message, status)
}
