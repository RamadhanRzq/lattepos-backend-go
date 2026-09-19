package kitchen

// UpdateStatusRequest adalah payload PATCH .../kitchen/{id}/status
// maupun PATCH .../items/{itemId}/status.
type UpdateStatusRequest struct {
	Status string `json:"status"`
}
