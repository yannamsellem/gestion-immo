package memory

import (
	"context"
	"sync"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
)

// RefreshTokenRepo is a thread-safe in-memory implementation of domain.RefreshTokenRepository for tests.
type RefreshTokenRepo struct {
	mu            sync.RWMutex
	refreshByHash map[string]domain.RefreshTokenRecord
}

// NewRefreshTokenRepo constructs an empty thread-safe in-memory RefreshTokenRepo.
func NewRefreshTokenRepo() *RefreshTokenRepo {
	return &RefreshTokenRepo{
		refreshByHash: make(map[string]domain.RefreshTokenRecord),
	}
}

// SaveRefreshToken saves a hashed refresh token record in memory.
func (r *RefreshTokenRepo) SaveRefreshToken(_ context.Context, record domain.RefreshTokenRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refreshByHash[record.TokenHash] = record
	return nil
}

// FindRefreshTokenByHash looks up a refresh token record by its SHA-256 hash.
func (r *RefreshTokenRepo) FindRefreshTokenByHash(_ context.Context, tokenHash string) (*domain.RefreshTokenRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rec, ok := r.refreshByHash[tokenHash]
	if !ok {
		return nil, domain.ErrInvalidToken
	}
	copyRec := rec
	return &copyRec, nil
}

// RotateRefreshToken atomically inserts the successor refresh token and marks the predecessor as replaced/revoked.
func (r *RefreshTokenRepo) RotateRefreshToken(_ context.Context, oldTokenID string, newRecord domain.RefreshTokenRecord, rotatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var oldHash string
	var oldRec domain.RefreshTokenRecord
	found := false
	for h, rec := range r.refreshByHash {
		if rec.ID == oldTokenID {
			oldHash = h
			oldRec = rec
			found = true
			break
		}
	}
	if !found || oldRec.IsRevoked() {
		return domain.ErrRefreshTokenReuseDetected
	}

	ts := rotatedAt
	newID := newRecord.ID
	oldRec.RevokedAt = &ts
	oldRec.ReplacedByID = &newID
	r.refreshByHash[oldHash] = oldRec
	r.refreshByHash[newRecord.TokenHash] = newRecord
	return nil
}

// RevokeRefreshTokenByHash marks a refresh token record as revoked.
func (r *RefreshTokenRepo) RevokeRefreshTokenByHash(_ context.Context, tokenHash string, revokedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec, ok := r.refreshByHash[tokenHash]
	if !ok {
		return nil
	}
	ts := revokedAt
	rec.RevokedAt = &ts
	r.refreshByHash[tokenHash] = rec
	return nil
}

// RevokeRefreshTokenFamily revokes all active refresh tokens belonging to familyID.
func (r *RefreshTokenRepo) RevokeRefreshTokenFamily(_ context.Context, familyID string, revokedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ts := revokedAt
	for hash, rec := range r.refreshByHash {
		if rec.FamilyID == familyID && rec.RevokedAt == nil {
			rec.RevokedAt = &ts
			r.refreshByHash[hash] = rec
		}
	}
	return nil
}

// RevokeAllUserRefreshTokens revokes all active refresh tokens for a user.
func (r *RefreshTokenRepo) RevokeAllUserRefreshTokens(_ context.Context, userID string, revokedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ts := revokedAt
	for hash, rec := range r.refreshByHash {
		if rec.UserID == userID && rec.RevokedAt == nil {
			rec.RevokedAt = &ts
			r.refreshByHash[hash] = rec
		}
	}
	return nil
}
