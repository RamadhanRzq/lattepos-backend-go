package organizations

import "time"

type Response struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

type SelectResponse struct {
	Token     string   `json:"token"`
	TokenType string   `json:"token_type"`
	Org       Response `json:"organization"`
}

func newResponse(o Organization) Response {
	return Response{
		ID:        o.ID,
		Name:      o.Name,
		Slug:      o.Slug,
		CreatedAt: o.CreatedAt,
	}
}
