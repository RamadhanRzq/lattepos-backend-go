package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"
)

// RefreshToken adalah entity domain refresh token.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	FamilyID  string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// RefreshTokenRepository adalah kontrak penyimpanan refresh token.
type RefreshTokenRepository interface {
	Create(ctx context.Context, rt *RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	RevokeFamily(ctx context.Context, familyID string) error
	RevokeAllByUser(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
}

// HashToken mengembalikan SHA-256 hash dari raw refresh token.
func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// postgresRefreshRepo mengimplementasikan RefreshTokenRepository di atas PostgreSQL.
type postgresRefreshRepo struct {
	db *sql.DB
}

var _ RefreshTokenRepository = (*postgresRefreshRepo)(nil)

func NewRefreshTokenRepository(db *sql.DB) RefreshTokenRepository {
	return &postgresRefreshRepo{db: db}
}

func (r *postgresRefreshRepo) Create(ctx context.Context, rt *RefreshToken) error {
	const q = `INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, q,
		rt.UserID, rt.TokenHash, rt.FamilyID, rt.ExpiresAt,
	).Scan(&rt.ID, &rt.CreatedAt, &rt.UpdatedAt)
}

func (r *postgresRefreshRepo) FindByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	const q = `SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at, updated_at
		FROM refresh_tokens WHERE token_hash = $1`

	rt := &RefreshToken{}
	err := r.db.QueryRowContext(ctx, q, hash).Scan(
		&rt.ID, &rt.UserID, &rt.TokenHash, &rt.FamilyID,
		&rt.ExpiresAt, &rt.RevokedAt, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func (r *postgresRefreshRepo) Revoke(ctx context.Context, id string) error {
	const q = `UPDATE refresh_tokens SET revoked_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *postgresRefreshRepo) RevokeFamily(ctx context.Context, familyID string) error {
	const q = `UPDATE refresh_tokens SET revoked_at = NOW(), updated_at = NOW()
		WHERE family_id = $1 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, q, familyID)
	return err
}

func (r *postgresRefreshRepo) RevokeAllByUser(ctx context.Context, userID string) error {
	const q = `UPDATE refresh_tokens SET revoked_at = NOW(), updated_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, q, userID)
	return err
}

func (r *postgresRefreshRepo) DeleteExpired(ctx context.Context) error {
	const q = `DELETE FROM refresh_tokens WHERE expires_at < NOW()`
	_, err := r.db.ExecContext(ctx, q)
	return err
}
