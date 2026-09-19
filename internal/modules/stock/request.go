package stock

// RecordRequest adalah payload POST .../stock-movements.
type RecordRequest struct {
	ProductID     string  `json:"product_id"`
	VariantID     *string `json:"variant_id"`
	Type          string  `json:"type"`
	Quantity      int     `json:"quantity"`
	ReferenceType string  `json:"reference_type"`
	ReferenceID   string  `json:"reference_id"`
	Notes         string  `json:"notes"`
	AllowNegative bool    `json:"allow_negative"`
}
