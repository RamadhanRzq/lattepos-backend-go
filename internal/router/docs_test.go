package router

import (
	"encoding/json"
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
		"GET /api/v1/org/{slug}/stores/{storeId}/kitchen/queue",
		"GET /api/v1/org/{slug}/stores/{storeId}/kitchen/{kitchenSaleId}",
		"PATCH /api/v1/org/{slug}/stores/{storeId}/kitchen/{kitchenSaleId}/status",
		"PATCH /api/v1/org/{slug}/stores/{storeId}/kitchen/{kitchenSaleId}/items/{itemId}/status",
	}
}
