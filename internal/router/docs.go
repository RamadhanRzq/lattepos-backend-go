package router

import (
	"encoding/json"
	"html"
	"net/http"
	"strings"
)

// docEndpoint adalah satu baris katalog API.
type docEndpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Auth   string `json:"auth"`
	Desc   string `json:"desc"`
}

// docGroup mengelompokkan endpoint per module.
type docGroup struct {
	Name      string        `json:"name"`
	Endpoints []docEndpoint `json:"endpoints"`
}

// apiDocs adalah katalog endpoint; tambah grup/baris baru tiap ada RegisterRoutes baru.
// Path ditulis persis seperti pattern mux ("{slug}" = slug organisasi).
var apiDocs = []docGroup{
	{"Health", []docEndpoint{
		{"GET", "/api/v1/health", "publik", "Liveness aplikasi"},
	}},
	{"Auth", []docEndpoint{
		{"POST", "/api/v1/register", "publik", "Registrasi user baru → JWT langsung"},
		{"POST", "/api/v1/login", "publik", "Login username+password → JWT"},
		{"GET", "/api/v1/me", "Bearer", "Profil user dari token"},
	}},
	{"Users", []docEndpoint{
		{"GET", "/api/v1/users", "users:read", "List user global"},
		{"POST", "/api/v1/users", "users:create", "Buat user"},
		{"GET", "/api/v1/users/{id}", "users:read", "Detail user"},
		{"GET", "/api/v1/org/{slug}/users", "org+users:read", "List user satu organisasi"},
		{"POST", "/api/v1/org/{slug}/users", "org+users:create", "Buat user dalam organisasi"},
		{"GET", "/api/v1/org/{slug}/users/{id}", "org+users:read", "Detail user dalam organisasi"},
	}},
	{"RBAC", []docEndpoint{
		{"GET", "/api/v1/permissions", "permissions:read", "List permission"},
		{"POST", "/api/v1/permissions", "permissions:create", "Buat permission"},
		{"GET", "/api/v1/roles", "roles:read", "List role global"},
		{"POST", "/api/v1/roles", "roles:create", "Buat role global"},
		{"GET", "/api/v1/roles/{id}", "roles:read", "Detail role global"},
		{"POST", "/api/v1/roles/{id}/permissions", "roles:update", "Tambah permission ke role"},
		{"POST", "/api/v1/users/{id}/roles", "users:update", "Tambah role ke user"},
		{"GET", "/api/v1/org/{slug}/roles", "org+roles:read", "List role organisasi"},
		{"POST", "/api/v1/org/{slug}/roles", "org+roles:create", "Buat role organisasi"},
		{"GET", "/api/v1/org/{slug}/roles/{id}", "org+roles:read", "Detail role organisasi"},
		{"POST", "/api/v1/org/{slug}/roles/{id}/permissions", "org+roles:update", "Tambah permission ke role organisasi"},
		{"POST", "/api/v1/org/{slug}/users/{id}/roles", "org+users:update", "Tambah role organisasi ke user"},
	}},
	{"Organizations", []docEndpoint{
		{"POST", "/api/v1/organizations", "Bearer", "Buat organisasi (pembuat jadi owner)"},
		{"GET", "/api/v1/organizations", "Bearer", "List organisasi"},
		{"GET", "/api/v1/org/{slug}", "Bearer", "Detail organisasi"},
		{"POST", "/api/v1/org/{slug}/auth/select", "Bearer", "Pilih organisasi aktif → token berkonteks org"},
		{"GET", "/api/v1/org/{slug}/members", "org", "List member organisasi"},
		{"POST", "/api/v1/org/{slug}/members", "org", "Tambah member"},
		{"DELETE", "/api/v1/org/{slug}/members/{uid}", "org", "Hapus member"},
	}},
	{"Stores", []docEndpoint{
		{"POST", "/api/v1/org/{slug}/stores", "org", "Buat store"},
		{"GET", "/api/v1/org/{slug}/stores", "org", "List store organisasi"},
		{"GET", "/api/v1/org/{slug}/stores/{id}", "org", "Detail store"},
		{"PUT", "/api/v1/org/{slug}/stores/{id}", "org", "Ubah store penuh"},
		{"PATCH", "/api/v1/org/{slug}/stores/{id}/status", "org", "Aktif/nonaktif store"},
		{"POST", "/api/v1/org/{slug}/stores/{id}/users", "org", "Beri user akses ke store"},
		{"DELETE", "/api/v1/org/{slug}/stores/{id}/users/{userId}", "org", "Cabut akses user dari store"},
		{"GET", "/api/v1/org/{slug}/stores/{id}/users", "org", "List user yang punya akses ke store"},
		{"GET", "/api/v1/org/{slug}/users/{id}/stores", "org", "List store yang bisa diakses user"},
	}},
	{"Products", []docEndpoint{
		{"POST", "/api/v1/org/{slug}/stores/{storeId}/products", "org", "Buat product (SKU unik per store)"},
		{"GET", "/api/v1/org/{slug}/stores/{storeId}/products", "org", "List product (?search, ?category_id, ?is_active, ?page, ?limit)"},
		{"GET", "/api/v1/org/{slug}/stores/{storeId}/products/{id}", "org", "Detail product"},
		{"PUT", "/api/v1/org/{slug}/stores/{storeId}/products/{id}", "org", "Ubah product penuh"},
		{"DELETE", "/api/v1/org/{slug}/stores/{storeId}/products/{id}", "org", "Hapus product (soft delete)"},
	}},
	{"Docs", []docEndpoint{
		{"GET", "/api/v1/docs", "publik", "Halaman dokumentasi ini"},
		{"GET", "/api/v1/docs.json", "publik", "Katalog endpoint format JSON"},
	}},
}

// registerDocs mendaftarkan rute dokumentasi API: halaman HTML + katalog JSON.
func registerDocs(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/docs", docsPage)
	mux.HandleFunc("GET /api/v1/docs.json", docsJSON)
}

// docsJSON menulis katalog endpoint sebagai JSON.
func docsJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiDocs)
}

// docsPage menulis katalog endpoint sebagai halaman HTML mandiri (tanpa dependensi).
func docsPage(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="id"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>LattePOS API Docs</title><style>` +
		`body{font-family:system-ui,sans-serif;max-width:960px;margin:2rem auto;padding:0 1rem;color:#111}` +
		`code{background:#f1f1f1;padding:.1rem .3rem;border-radius:4px}` +
		`table{border-collapse:collapse;width:100%;margin-bottom:2rem}` +
		`th,td{border:1px solid #ddd;padding:.4rem .6rem;text-align:left;font-size:.9rem}` +
		`th{background:#f7f7f7}` +
		`.m{display:inline-block;min-width:3.2rem;text-align:center;color:#fff;border-radius:4px;font-size:.75rem;padding:.1rem .3rem}` +
		`.get{background:#2f7d32}.post{background:#1565c0}.put{background:#ef6c00}.patch{background:#6a1b9a}.delete{background:#c62828}` +
		`</style></head><body><h1>LattePOS API Docs</h1>` +
		`<p>Header auth: <code>Authorization: Bearer &lt;token&gt;</code>. ` +
		`<code>{slug}</code> = slug organisasi. Katalog mesin: <a href="/api/v1/docs.json"><code>/api/v1/docs.json</code></a>.</p>`)

	for _, g := range apiDocs {
		b.WriteString("<h2>" + html.EscapeString(g.Name) + "</h2><table><tr><th>Method</th><th>Path</th><th>Auth</th><th>Deskripsi</th></tr>")
		for _, e := range g.Endpoints {
			b.WriteString("<tr><td><span class=\"m " + strings.ToLower(e.Method) + "\">" + e.Method + "</span></td>" +
				"<td><code>" + html.EscapeString(e.Path) + "</code></td>" +
				"<td><code>" + html.EscapeString(e.Auth) + "</code></td>" +
				"<td>" + html.EscapeString(e.Desc) + "</td></tr>")
		}
		b.WriteString("</table>")
	}
	b.WriteString("</body></html>")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(b.String()))
}
