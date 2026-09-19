package variants

// MessageResponse adalah envelope pesan singkat untuk delete.
type MessageResponse struct {
	Message string `json:"message"`
}

// ListResponse adalah envelope list variant dalam satu product.
type ListResponse struct {
	Data []ProductVariant `json:"data"`
}
