package products

// CreateRequest adalah payload POST /api/v1/org/{slug}/stores/{storeId}/products.
// product_type opsional; kosong berarti MENU (barang jual).
type CreateRequest struct {
	Name        string  `json:"name"`
	SKU         string  `json:"sku"`
	Description string  `json:"description"`
	ProductType string  `json:"product_type"`
	Price       int64   `json:"price"`
	Stock       int     `json:"stock"`
	Unit        string  `json:"unit"`
	CategoryID  *string `json:"category_id"`
	ImageURL    string  `json:"image_url"`
}

// UpdateRequest adalah payload PUT .../products/{id}.
// Full replace kecuali store_id/organization_id (lihat service).
type UpdateRequest struct {
	Name        string  `json:"name"`
	SKU         string  `json:"sku"`
	Description string  `json:"description"`
	ProductType string  `json:"product_type"`
	Price       int64   `json:"price"`
	Stock       int     `json:"stock"`
	Unit        string  `json:"unit"`
	CategoryID  *string `json:"category_id"`
	ImageURL    string  `json:"image_url"`
	IsActive    *bool   `json:"is_active"`
}
