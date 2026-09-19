package prices

import "time"

// CreateRequest adalah payload POST .../products/{productId}/prices.
type CreateRequest struct {
	VariantID   *string    `json:"variant_id"`
	PriceType   string     `json:"price_type"`
	Price       int64      `json:"price"`
	MinQuantity int        `json:"min_quantity"`
	ValidFrom   *time.Time `json:"valid_from"`
	ValidUntil  *time.Time `json:"valid_until"`
}

// UpdateRequest adalah payload PUT .../prices/{priceId}.
type UpdateRequest struct {
	VariantID   *string    `json:"variant_id"`
	PriceType   string     `json:"price_type"`
	Price       int64      `json:"price"`
	MinQuantity int        `json:"min_quantity"`
	IsActive    *bool      `json:"is_active"`
	ValidFrom   *time.Time `json:"valid_from"`
	ValidUntil  *time.Time `json:"valid_until"`
}
