package stores

// CreateRequest adalah payload POST /api/v1/org/{slug}/stores.
type CreateRequest struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

// UpdateRequest adalah payload PUT /api/v1/org/{slug}/stores/{id}.
// Full replace kecuali organization_id dan status (lihat StatusRequest).
type UpdateRequest struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

// StatusRequest adalah payload PATCH /api/v1/org/{slug}/stores/{id}/status.
// Pointer supaya is_active yang hilang ditolak, bukan dibaca sebagai false.
type StatusRequest struct {
	IsActive *bool `json:"is_active"`
}

// AssignRequest adalah payload POST /api/v1/org/{slug}/stores/{id}/users.
type AssignRequest struct {
	UserID string `json:"user_id"`
}
