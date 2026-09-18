package users

// CreateRequest adalah payload POST /api/v1/users.
type CreateRequest struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
