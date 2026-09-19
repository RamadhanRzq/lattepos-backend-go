package prices

// MessageResponse adalah envelope pesan singkat untuk delete.
type MessageResponse struct {
	Message string `json:"message"`
}

// ListResponse adalah envelope list price dalam satu product.
type ListResponse struct {
	Data []ProductPrice `json:"data"`
}
