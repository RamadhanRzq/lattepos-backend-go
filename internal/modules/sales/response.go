package sales

// ListResponse adalah envelope list dengan total untuk paginasi.
type ListResponse struct {
	Data  []Sale `json:"data"`
	Total int    `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}
