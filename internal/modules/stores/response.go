package stores

// MessageResponse adalah envelope pesan singkat untuk assign/remove/status.
type MessageResponse struct {
	Message string `json:"message"`
}
