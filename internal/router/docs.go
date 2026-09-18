package router

import (
	_ "embed"
	"encoding/json"
	"html"
	"net/http"
	"strings"
)

//go:embed api-docs.json
var apiDocsJSON []byte

// apiDocs dimuat sekali dari api-docs.json (sumber tunggal katalog).
// Maintenance: edit internal/router/api-docs.json saja, tanpa sentuh Go.
var apiDocs = mustLoadDocs()

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

func mustLoadDocs() []docGroup {
	var groups []docGroup
	if err := json.Unmarshal(apiDocsJSON, &groups); err != nil {
		panic("router: api-docs.json tidak valid: " + err.Error())
	}
	return groups
}

// registerDocs mendaftarkan rute dokumentasi API: halaman HTML + katalog JSON.
func registerDocs(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/docs", docsPage)
	mux.HandleFunc("GET /api/v1/docs.json", docsJSON)
}

// docsJSON menyajikan file katalog apa adanya (byte embed dari api-docs.json).
func docsJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(apiDocsJSON)
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
