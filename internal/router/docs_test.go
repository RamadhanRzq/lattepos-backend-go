package router

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestDocsCatalogIsPostmanV21 memastikan api-docs.json bisa diimport Postman.
func TestDocsCatalogIsPostmanV21(t *testing.T) {
	var col struct {
		Info struct {
			Schema string `json:"schema"`
		} `json:"info"`
		Item []struct {
			Name string `json:"name"`
			Item []struct {
				Name    string `json:"name"`
				Request struct {
					Method string `json:"method"`
					URL    struct {
						Raw  string   `json:"raw"`
						Path []string `json:"path"`
					} `json:"url"`
					Body *struct {
						Mode string `json:"mode"`
						Raw  string `json:"raw"`
					} `json:"body"`
				} `json:"request"`
			} `json:"item"`
		} `json:"item"`
	}
	if err := json.Unmarshal(apiDocsJSON, &col); err != nil {
		t.Fatalf("api-docs.json bukan JSON valid: %v", err)
	}
	if col.Info.Schema != "https://schema.getpostman.com/json/collection/v2.1.0/collection.json" {
		t.Fatalf("schema bukan Postman v2.1: %q", col.Info.Schema)
	}
	n := 0
	for _, f := range col.Item {
		if len(f.Item) == 0 {
			t.Errorf("folder %q kosong", f.Name)
		}
		for _, it := range f.Item {
			n++
			if it.Request.Method == "" || it.Request.URL.Raw == "" || len(it.Request.URL.Path) == 0 {
				t.Errorf("request %q/%q tanpa method/url", f.Name, it.Name)
			}
			if it.Request.Body != nil && it.Request.Body.Mode == "raw" {
				var v any
				if err := json.Unmarshal([]byte(it.Request.Body.Raw), &v); err != nil {
					t.Errorf("body %q/%q bukan JSON valid: %v", f.Name, it.Name, err)
				}
			}
		}
	}
	if n == 0 {
		t.Fatal("koleksi tanpa request")
	}
}

// TestDocsCatalogCoversMux memastikan tiap pola route mux ada di katalog.
func TestDocsCatalogCoversMux(t *testing.T) {
	var col struct {
		Item []struct {
			Item []struct {
				Request struct {
					Method string `json:"method"`
					URL    struct {
						Raw string `json:"raw"`
					} `json:"url"`
				} `json:"request"`
			} `json:"item"`
		} `json:"item"`
	}
	if err := json.Unmarshal(apiDocsJSON, &col); err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	for _, f := range col.Item {
		for _, it := range f.Item {
			raw := strings.ReplaceAll(it.Request.URL.Raw, "{{baseUrl}}", "")
			raw = muxPattern(raw)
			paths[it.Request.Method+" "+raw] = true
		}
	}
	for _, r := range muxPatterns() {
		if !paths[r] {
			t.Errorf("route mux tanpa katalog: %s", r)
		}
	}
}

// TestDocsQueryParamsMatchHandlers menjaga keselarasan nama query param antara
// katalog dan handler. Drift nyata: katalog menulis "only_active" padahal
// handler membaca "active" — user Postman dapat hasil tak terfilter tanpa error.
func TestDocsQueryParamsMatchHandlers(t *testing.T) {
	documented := map[string]bool{}
	var col struct {
		Item []struct {
			Item []struct {
				Request struct {
					URL struct {
						Query []struct {
							Key string `json:"key"`
						} `json:"query"`
					} `json:"url"`
				} `json:"request"`
			} `json:"item"`
		} `json:"item"`
	}
	if err := json.Unmarshal(apiDocsJSON, &col); err != nil {
		t.Fatal(err)
	}
	for _, f := range col.Item {
		for _, it := range f.Item {
			for _, q := range it.Request.URL.Query {
				documented[q.Key] = true
			}
		}
	}

	files, err := filepath.Glob("../../internal/modules/*/handler.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("tidak menemukan handler module")
	}
	used := map[string]bool{}
	for _, name := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Get" {
				return true
			}
			if !isQuerySource(sel.X) {
				return true
			}
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if key, err := strconv.Unquote(lit.Value); err == nil {
					used[key] = true
				}
			}
			return true
		})
	}

	for k := range used {
		if !documented[k] {
			t.Errorf("handler baca query %q tapi tidak ada di api-docs.json", k)
		}
	}
	for k := range documented {
		if !used[k] {
			t.Errorf("api-docs.json dokumentasikan query %q tapi tidak ada handler yang membacanya", k)
		}
	}
}

// isQuerySource melaporkan apakah expr adalah `r.URL.Query()` atau variabel
// hasilnya (`q := r.URL.Query()`).
func isQuerySource(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "q"
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Query"
}

// muxPattern mengubah path collection (":slug") ke pola mux ("{slug}").
func muxPattern(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if strings.HasPrefix(s, ":") {
			segs[i] = "{" + s[1:] + "}"
		}
	}
	return strings.Join(segs, "/")
}

// muxPatterns adalah seluruh pola route di bawah /api/v1 (sumber: RegisterRoutes).
func muxPatterns() []string {
	return []string{
		"GET /api/v1/health",
		"GET /api/v1/docs",
		"GET /api/v1/docs.json",
		"POST /api/v1/register",
		"POST /api/v1/login",
		"GET /api/v1/me",
		"GET /api/v1/users",
		"POST /api/v1/users",
		"GET /api/v1/users/{id}",
		"GET /api/v1/org/{slug}/users",
		"POST /api/v1/org/{slug}/users",
		"GET /api/v1/org/{slug}/users/{id}",
		"GET /api/v1/permissions",
		"POST /api/v1/permissions",
		"GET /api/v1/roles",
		"POST /api/v1/roles",
		"GET /api/v1/roles/{id}",
		"POST /api/v1/roles/{id}/permissions",
		"POST /api/v1/users/{id}/roles",
		"GET /api/v1/org/{slug}/roles",
		"POST /api/v1/org/{slug}/roles",
		"GET /api/v1/org/{slug}/roles/{id}",
		"POST /api/v1/org/{slug}/roles/{id}/permissions",
		"POST /api/v1/org/{slug}/users/{id}/roles",
		"POST /api/v1/organizations",
		"GET /api/v1/organizations",
		"GET /api/v1/org/{slug}",
		"POST /api/v1/org/{slug}/auth/select",
		"GET /api/v1/org/{slug}/members",
		"POST /api/v1/org/{slug}/members",
		"DELETE /api/v1/org/{slug}/members/{uid}",
		"POST /api/v1/org/{slug}/stores",
		"GET /api/v1/org/{slug}/stores",
		"GET /api/v1/org/{slug}/stores/{id}",
		"PUT /api/v1/org/{slug}/stores/{id}",
		"PATCH /api/v1/org/{slug}/stores/{id}/status",
		"POST /api/v1/org/{slug}/stores/{id}/users",
		"DELETE /api/v1/org/{slug}/stores/{id}/users/{userId}",
		"GET /api/v1/org/{slug}/stores/{id}/users",
		"GET /api/v1/org/{slug}/users/{id}/stores",
		"POST /api/v1/org/{slug}/stores/{storeId}/products",
		"GET /api/v1/org/{slug}/stores/{storeId}/products",
		"GET /api/v1/org/{slug}/stores/{storeId}/products/{id}",
		"PUT /api/v1/org/{slug}/stores/{storeId}/products/{id}",
		"DELETE /api/v1/org/{slug}/stores/{storeId}/products/{id}",
		"POST /api/v1/org/{slug}/stores/{storeId}/sales",
		"GET /api/v1/org/{slug}/stores/{storeId}/sales",
		"GET /api/v1/org/{slug}/stores/{storeId}/sales/{id}",
		"PATCH /api/v1/org/{slug}/stores/{storeId}/sales/{id}/cancel",
		"GET /api/v1/org/{slug}/stores/{storeId}/transactions/daily",
		"POST /api/v1/org/{slug}/stores/{storeId}/transactions",
		"GET /api/v1/org/{slug}/stores/{storeId}/transactions",
		"GET /api/v1/org/{slug}/stores/{storeId}/transactions/{id}",
		"PATCH /api/v1/org/{slug}/stores/{storeId}/transactions/{id}/cancel",
		"POST /api/v1/org/{slug}/stores/{storeId}/categories",
		"GET /api/v1/org/{slug}/stores/{storeId}/categories",
		"GET /api/v1/org/{slug}/stores/{storeId}/categories/{id}",
		"PUT /api/v1/org/{slug}/stores/{storeId}/categories/{id}",
		"DELETE /api/v1/org/{slug}/stores/{storeId}/categories/{id}",
		"POST /api/v1/org/{slug}/stores/{storeId}/products/{productId}/variants",
		"GET /api/v1/org/{slug}/stores/{storeId}/products/{productId}/variants",
		"PUT /api/v1/org/{slug}/stores/{storeId}/products/{productId}/variants/{variantId}",
		"DELETE /api/v1/org/{slug}/stores/{storeId}/products/{productId}/variants/{variantId}",
		"POST /api/v1/org/{slug}/stores/{storeId}/products/{productId}/prices",
		"GET /api/v1/org/{slug}/stores/{storeId}/products/{productId}/prices",
		"PUT /api/v1/org/{slug}/stores/{storeId}/products/{productId}/prices/{priceId}",
		"DELETE /api/v1/org/{slug}/stores/{storeId}/products/{productId}/prices/{priceId}",
		"POST /api/v1/org/{slug}/stores/{storeId}/stock-movements",
		"GET /api/v1/org/{slug}/stores/{storeId}/stock-movements",
		"GET /api/v1/org/{slug}/stores/{storeId}/products/{productId}/stock",
		"POST /api/v1/org/{slug}/stores/{storeId}/products/{productId}/recipes",
		"GET /api/v1/org/{slug}/stores/{storeId}/products/{productId}/recipes",
		"GET /api/v1/org/{slug}/stores/{storeId}/recipes",
		"GET /api/v1/org/{slug}/stores/{storeId}/products/{productId}/recipes/{recipeId}",
		"PUT /api/v1/org/{slug}/stores/{storeId}/products/{productId}/recipes/{recipeId}",
		"DELETE /api/v1/org/{slug}/stores/{storeId}/products/{productId}/recipes/{recipeId}",
		"PATCH /api/v1/org/{slug}/stores/{storeId}/products/{productId}/recipes/{recipeId}/activate",
		"GET /api/v1/org/{slug}/stores/{storeId}/kitchen/queue",
		"GET /api/v1/org/{slug}/stores/{storeId}/kitchen/{kitchenSaleId}",
		"PATCH /api/v1/org/{slug}/stores/{storeId}/kitchen/{kitchenSaleId}/status",
		"PATCH /api/v1/org/{slug}/stores/{storeId}/kitchen/{kitchenSaleId}/items/{itemId}/status",
	}
}
