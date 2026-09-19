package stock

// MessageResponse adalah envelope pesan singkat untuk delete.
type MessageResponse struct {
	Message string `json:"message"`
}

// ListResponse adalah envelope list dengan total untuk paginasi.
type ListResponse struct {
	Data  []StockMovement `json:"data"`
	Total int             `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
}

// StockResponse adalah ringkasan stok live + riwayat movement satu produk.
type StockResponse struct {
	Summary StockSummary    `json:"summary"`
	History []StockMovement `json:"history"`
}
