package products

// MessageResponse adalah envelope pesan singkat untuk delete.
type MessageResponse struct {
	Message string `json:"message"`
}

// ListResponse adalah envelope list dengan total untuk paginasi.
type ListResponse struct {
	Data  []Product `json:"data"`
	Total int       `json:"total"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
}
