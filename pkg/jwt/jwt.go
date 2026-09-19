package jwt

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims adalah payload standar access token.
// UserID diduplikasi dari RegisteredClaims.Subject (claim "sub") supaya
// caller bisa akses field .UserID tanpa .Subject.
type Claims struct {
	UserID  string `json:"-"`
	OrgID   string `json:"org_id,omitempty"`
	OrgSlug string `json:"org_slug,omitempty"`
	jwt.RegisteredClaims
}

// Manager menangani generate & verifikasi access dan refresh token.
type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewManager(accessSecret string, accessTTL time.Duration, refreshSecret string, refreshTTL time.Duration) *Manager {
	return &Manager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

// AccessTTL mengembalikan durasi masa berlaku access token.
func (m *Manager) AccessTTL() time.Duration { return m.accessTTL }

// RefreshTTL mengembalikan durasi masa berlaku refresh token.
func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }

// GenerateAccess membuat access token HS256 dengan jti.
func (m *Manager) GenerateAccess(userID string) (string, error) {
	return m.GenerateAccessWithOrg(userID, "", "")
}

// GenerateAccessWithOrg membuat access token HS256 dengan konteks organisasi.
func (m *Manager) GenerateAccessWithOrg(userID, orgID, orgSlug string) (string, error) {
	jti, err := randomID()
	if err != nil {
		return "", fmt.Errorf("generate jti: %w", err)
	}

	now := time.Now()
	claims := Claims{
		OrgID:   orgID,
		OrgSlug: orgSlug,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "lattepos",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.accessSecret)
}

// GenerateRefresh membuat opaque refresh token (hex-encoded random bytes).
// Mengembalikan raw token (diberikan ke client) dan expiry.
func (m *Manager) GenerateRefresh() (string, time.Time, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}
	return hex.EncodeToString(b), time.Now().Add(m.refreshTTL), nil
}

// Verify memvalidasi access token dan mengembalikan claims-nya.
// UserID di-populate dari Subject (claim "sub").
func (m *Manager) Verify(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return m.accessSecret, nil
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

	if claims.Subject == "" {
		return nil, errors.New("token tidak valid: missing sub")
	}

	claims.UserID = claims.Subject
	return claims, nil
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
