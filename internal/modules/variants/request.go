package variants

// CreateRequest adalah payload POST .../products/{productId}/variants.
type CreateRequest struct {
	Name  string `json:"name"`
	SKU   string `json:"sku"`
	Stock int    `json:"stock"`
}

// UpdateRequest adalah payload PUT .../variants/{variantId}.
type UpdateRequest struct {
	Name     string `json:"name"`
	SKU      string `json:"sku"`
	Stock    int    `json:"stock"`
	IsActive *bool  `json:"is_active"`
}
