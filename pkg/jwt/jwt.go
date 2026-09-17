package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims adalah payload standar yang ditanam di setiap token.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	OrgID    string `json:"org_id,omitempty"`
	OrgSlug  string `json:"org_slug,omitempty"`
	jwt.RegisteredClaims
}

// Manager menangani generate & verifikasi token.
type Manager struct {
	secret     []byte
	expiration time.Duration
}

func NewManager(secret string, expiration time.Duration) *Manager {
	return &Manager{
		secret:     []byte(secret),
		expiration: expiration,
	}
}

// Expiration mengembalikan durasi masa berlaku token yang diterbitkan.
func (m *Manager) Expiration() time.Duration {
	return m.expiration
}

// Generate membuat token HS256 untuk user standar.
func (m *Manager) Generate(userID, username, email, name string) (string, error) {
	return m.GenerateWithOrg(userID, username, email, name, "", "")
}

// GenerateWithOrg membuat token HS256 dengan konteks organisasi.
func (m *Manager) GenerateWithOrg(userID, username, email, name, orgID, orgSlug string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Name:     name,
		OrgID:    orgID,
		OrgSlug:  orgSlug,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "lattepos",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(m.secret)
}

// Verify memvalidasi token dan mengembalikan claims-nya.
func (m *Manager) Verify(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return nil, fmt.Errorf("token tidak valid: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("token tidak valid")
	}

	return claims, nil
}
