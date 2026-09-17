package organizations

// CreateRequest adalah payload POST /api/v1/organizations.
type CreateRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// AddMemberRequest adalah payload POST /api/v1/org/{slug}/members.
type AddMemberRequest struct {
	UserID string `json:"user_id"`
}
