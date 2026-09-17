package auth

// LoginRequest adalah payload POST /login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
