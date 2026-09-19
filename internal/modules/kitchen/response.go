package kitchen

// QueueResponse adalah envelope antrian aktif dapur.
type QueueResponse struct {
	Data []KitchenSale `json:"data"`
}
