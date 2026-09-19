package sales

// CreateItem adalah satu baris payload POST .../sales.
type CreateItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// CreateRequest adalah payload POST /api/v1/org/{slug}/stores/{storeId}/sales.
// Total tidak diterima dari client: dihitung service dari snapshot harga produk.
type CreateRequest struct {
	PaymentMethod  string       `json:"payment_method"`
	DiscountAmount int64        `json:"discount_amount"`
	TaxAmount      int64        `json:"tax_amount"`
	Notes          string       `json:"notes"`
	Items          []CreateItem `json:"items"`
}
