package tables

// CreateRequest adalah payload POST /api/v1/org/{slug}/stores/{storeId}/tables.
type CreateRequest struct {
	Name     string `json:"name"`
	Area     string `json:"area"`
	Capacity int    `json:"capacity"`
	Status   string `json:"status"`
	IsActive *bool  `json:"is_active"`
}

// UpdateRequest adalah payload PUT .../tables/{id}.
// Full replace kecuali store_id/organization_id (lihat service).
type UpdateRequest struct {
	Name     string `json:"name"`
	Area     string `json:"area"`
	Capacity int    `json:"capacity"`
	IsActive *bool  `json:"is_active"`
}

// UpdateStatusRequest adalah payload PATCH .../tables/{id}/status.
type UpdateStatusRequest struct {
	Status string `json:"status"`
}
