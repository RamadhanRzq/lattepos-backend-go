package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func testManager() *Manager {
	return NewManager("access-secret", 15*time.Minute, "refresh-secret", 24*time.Hour)
}

func TestManager_GenerateAndVerify(t *testing.T) {
	m := testManager()

	t.Run("valid access token", func(t *testing.T) {
		token, err := m.GenerateAccess("user-1")
		if err != nil {
			t.Fatal(err)
		}

		claims, err := m.Verify(token)
		if err != nil {
			t.Fatalf("verify failed: %v", err)
		}
		if claims.UserID != "user-1" {
			t.Errorf("UserID = %q, want user-1", claims.UserID)
		}
		if claims.Subject != "user-1" {
			t.Errorf("Subject = %q, want user-1", claims.Subject)
		}
		if claims.ID == "" {
			t.Error("jti should not be empty")
		}
		if claims.Issuer != "lattepos" {
			t.Errorf("Issuer = %q, want lattepos", claims.Issuer)
		}
	})

	t.Run("access token with org", func(t *testing.T) {
		token, err := m.GenerateAccessWithOrg("user-1", "org-1", "kopi")
		if err != nil {
			t.Fatal(err)
		}

		claims, err := m.Verify(token)
		if err != nil {
			t.Fatal(err)
		}
		if claims.OrgID != "org-1" {
			t.Errorf("OrgID = %q, want org-1", claims.OrgID)
		}
		if claims.OrgSlug != "kopi" {
			t.Errorf("OrgSlug = %q, want kopi", claims.OrgSlug)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		m := NewManager("secret", 0, "refresh", time.Hour)
		token, err := m.GenerateAccess("user-1")
		if err != nil {
			t.Fatal(err)
		}

		_, err = m.Verify(token)
		if err == nil {
			t.Fatal("expected error for expired token")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		m2 := NewManager("wrong-secret", 15*time.Minute, "wrong", time.Hour)
		token, _ := m.GenerateAccess("user-1")

		_, err := m2.Verify(token)
		if err == nil {
			t.Fatal("expected error for wrong secret")
		}
	})

	t.Run("wrong algorithm", func(t *testing.T) {
		// Craft a token with RS256 header but HMAC body
		token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
			"sub": "user-1",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		signed, _ := token.SignedString([]byte("access-secret"))

		_, err := m.Verify(signed)
		if err == nil {
			t.Fatal("expected error for wrong algorithm")
		}
	})

	t.Run("missing sub", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})
		signed, _ := token.SignedString([]byte("access-secret"))

		_, err := m.Verify(signed)
		if err == nil {
			t.Fatal("expected error for missing sub")
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := m.Verify("not.a.jwt")
		if err == nil {
			t.Fatal("expected error for malformed token")
		}
	})
}

func TestManager_GenerateRefresh(t *testing.T) {
	m := testManager()

	t.Run("produces unique tokens", func(t *testing.T) {
		t1, exp1, err := m.GenerateRefresh()
		if err != nil {
			t.Fatal(err)
		}
		t2, _, err := m.GenerateRefresh()
		if err != nil {
			t.Fatal(err)
		}

		if t1 == t2 {
			t.Error("refresh tokens should be unique")
		}
		if len(t1) != 64 {
			t.Errorf("refresh token length = %d, want 64 hex chars", len(t1))
		}
		if exp1.Before(time.Now()) {
			t.Error("expiry should be in the future")
		}
	})
}

func TestManager_TTL(t *testing.T) {
	m := testManager()

	if m.AccessTTL() != 15*time.Minute {
		t.Errorf("AccessTTL = %v, want 15m", m.AccessTTL())
	}
	if m.RefreshTTL() != 24*time.Hour {
		t.Errorf("RefreshTTL = %v, want 24h", m.RefreshTTL())
	}
}
