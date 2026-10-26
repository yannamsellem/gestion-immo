package domain

import (
	"context"
	"time"
)

// RefreshTokenRecord is the aggregate representing a hashed refresh token persisted as part of a rotation family.
// Plaintext refresh tokens are never stored.
type RefreshTokenRecord struct {
	ID           string
	UserID       string
	FamilyID     string
	TokenHash    string
	ReplacedByID *string
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	CreatedAt    time.Time
}

// IsRevoked reports whether the refresh token has already been rotated or explicitly revoked.
func (r RefreshTokenRecord) IsRevoked() bool {
	return r.RevokedAt != nil || r.ReplacedByID != nil
}

// IsExpired reports whether the refresh token has expired at now.
func (r RefreshTokenRecord) IsExpired(now time.Time) bool {
	return !now.Before(r.ExpiresAt)
}

// IsActive reports whether the refresh token has neither been revoked/rotated nor expired at now.
func (r RefreshTokenRecord) IsActive(now time.Time) bool {
	return !r.IsRevoked() && !r.IsExpired(now)
}

// RefreshTokenRepository defines the persistence contract dedicated to the RefreshTokenRecord aggregate.
type RefreshTokenRepository interface {
	SaveRefreshToken(ctx context.Context, record RefreshTokenRecord) error
	FindRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshTokenRecord, error)
	RotateRefreshToken(ctx context.Context, oldTokenID string, newRecord RefreshTokenRecord, rotatedAt time.Time) error
	RevokeRefreshTokenByHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	RevokeRefreshTokenFamily(ctx context.Context, familyID string, revokedAt time.Time) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID string, revokedAt time.Time) error
}
