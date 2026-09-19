package categories

// CreateRequest adalah payload POST /api/v1/org/{slug}/stores/{storeId}/categories.
type CreateRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	ParentID    *string `json:"parent_id"`
}

// UpdateRequest adalah payload PUT .../categories/{id}.
// Full replace kecuali store_id/organization_id (lihat service).
type UpdateRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	ParentID    *string `json:"parent_id"`
}
