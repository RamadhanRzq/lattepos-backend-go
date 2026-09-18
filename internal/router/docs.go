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
// Maintenance: edit internal/router/api-docs.json sebagai Postman Collection v2.1,
// lalu import file yang sama ke Postman. Tanpa sentuh Go.
var apiDocs = mustLoadDocs()

// docEndpoint adalah satu baris katalog API untuk halaman HTML.
type docEndpoint struct {
	Method string
	Path   string
	Auth   string
	Desc   string
}

// docGroup mengelompokkan endpoint per module untuk halaman HTML.
type docGroup struct {
	Name      string
	Endpoints []docEndpoint
}

// Struktur minimal Postman Collection v2.1; field lain diabaikan.
type postmanCollection struct {
	Info struct {
		Schema string `json:"schema"`
	} `json:"info"`
	Item []postmanFolder `json:"item"`
}

type postmanFolder struct {
	Name string        `json:"name"`
	Item []postmanItem `json:"item"`
}

type postmanItem struct {
	Name    string         `json:"name"`
	Request postmanRequest `json:"request"`
}

type postmanRequest struct {
	Method      string     `json:"method"`
	URL         postmanURL `json:"url"`
	Description string     `json:"description"`
}

type postmanURL struct {
	Path []string `json:"path"`
}

func mustLoadDocs() []docGroup {
	var col postmanCollection
	if err := json.Unmarshal(apiDocsJSON, &col); err != nil {
		panic("router: api-docs.json tidak valid: " + err.Error())
	}
	if !strings.Contains(col.Info.Schema, "schema.getpostman.com") {
		panic("router: api-docs.json bukan Postman Collection v2.1")
	}
	groups := make([]docGroup, 0, len(col.Item))
	for _, f := range col.Item {
		g := docGroup{Name: f.Name}
		for _, it := range f.Item {
			segs := make([]string, 0, len(it.Request.URL.Path))
			for _, s := range it.Request.URL.Path {
				if strings.HasPrefix(s, ":") && len(s) > 1 {
					s = "{" + s[1:] + "}"
				}
				segs = append(segs, s)
			}
			desc, auth := splitDescription(it.Request.Description)
			g.Endpoints = append(g.Endpoints, docEndpoint{
				Method: it.Request.Method,
				Path:   "/" + strings.Join(segs, "/"),
				Auth:   auth,
				Desc:   desc,
			})
		}
		groups = append(groups, g)
	}
	return groups
}

// splitDescription memecah "<desc>\n\nAuth: `<auth>`." hasil generate koleksi.
func splitDescription(s string) (desc, auth string) {
	desc = s
	const marker = "\n\nAuth: `"
	i := strings.LastIndex(s, marker)
	if i < 0 {
		return strings.TrimSpace(desc), ""
	}
	auth = strings.TrimSuffix(s[i+len(marker):], "`.")
	return strings.TrimSpace(s[:i]), strings.TrimSpace(auth)
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
		`<p>Header auth: bearer token koleksi (<code>{{token}}</code>). ` +
		`<code>{slug}</code> = slug organisasi. Koleksi Postman: <a href="/api/v1/docs.json"><code>/api/v1/docs.json</code></a> (import langsung ke Postman).</p>`)

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
