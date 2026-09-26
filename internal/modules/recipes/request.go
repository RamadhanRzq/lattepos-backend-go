package recipes

// CreateRequest adalah payload POST .../products/{productId}/recipes.
// items full replace; is_active true akan menonaktifkan versi lain.
type CreateRequest struct {
	Name          string      `json:"name"`
	Version       int         `json:"version"`
	YieldQuantity float64     `json:"yield_quantity"`
	IsActive      bool        `json:"is_active"`
	Notes         string      `json:"notes"`
	Items         []ItemInput `json:"items"`
}

// UpdateRequest adalah payload PUT .../recipes/{recipeId}.
// Full replace kecuali product_id/organization_id/store_id (lihat service).
type UpdateRequest struct {
	Name          string      `json:"name"`
	Version       int         `json:"version"`
	YieldQuantity float64     `json:"yield_quantity"`
	IsActive      bool        `json:"is_active"`
	Notes         string      `json:"notes"`
	Items         []ItemInput `json:"items"`
}
