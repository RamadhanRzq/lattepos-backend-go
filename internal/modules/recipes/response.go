package recipes

// MessageResponse adalah envelope pesan singkat untuk delete.
type MessageResponse struct {
	Message string `json:"message"`
}

// ConsumeResult adalah ringkasan konsumsi bahan satu product terjual.
type ConsumeResult struct {
	ProductID string               `json:"product_id"`
	RecipeID  string               `json:"recipe_id,omitempty"`
	Quantity  int                  `json:"quantity"`
	Items     []ConsumedIngredient `json:"items"`
}

// ConsumedIngredient adalah satu bahan yang dikurangi saat konsumsi resep.
type ConsumedIngredient struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name,omitempty"`
	QuantityUsed int    `json:"quantity_used"`
}
