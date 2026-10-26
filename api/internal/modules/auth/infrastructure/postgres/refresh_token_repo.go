package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	platformpg "github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// RefreshTokenRepo implements domain.RefreshTokenRepository against PostgreSQL.
type RefreshTokenRepo struct {
	pool *pgxpool.Pool
	db   platformpg.DBTX
}

// NewRefreshTokenRepo constructs a PostgreSQL repository dedicated to the RefreshTokenRecord aggregate.
func NewRefreshTokenRepo(pool *pgxpool.Pool, db platformpg.DBTX) *RefreshTokenRepo {
	if db == nil {
		db = pool
	}
	return &RefreshTokenRepo{pool: pool, db: db}
}

// SaveRefreshToken persists a hashed refresh token record.
func (r *RefreshTokenRepo) SaveRefreshToken(ctx context.Context, record domain.RefreshTokenRecord) error {
	const query = `
		INSERT INTO refresh_tokens (id, user_id, family_id, token_hash, replaced_by_id, expires_at, revoked_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := platformpg.Conn(ctx, r.db).Exec(ctx, query,
		record.ID,
		record.UserID,
		record.FamilyID,
		record.TokenHash,
		record.ReplacedByID,
		record.ExpiresAt,
		record.RevokedAt,
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("refresh_token_postgres_repo: save refresh token: %w", err)
	}
	return nil
}

// FindRefreshTokenByHash looks up a refresh token record by its SHA-256 hex digest.
func (r *RefreshTokenRepo) FindRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshTokenRecord, error) {
	const query = `
		SELECT id, user_id, family_id, token_hash, replaced_by_id, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	var rec domain.RefreshTokenRecord
	err := platformpg.Conn(ctx, r.db).QueryRow(ctx, query, tokenHash).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.FamilyID,
		&rec.TokenHash,
		&rec.ReplacedByID,
		&rec.ExpiresAt,
		&rec.RevokedAt,
		&rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidToken
		}
		return nil, fmt.Errorf("refresh_token_postgres_repo: find refresh token: %w", err)
	}
	return &rec, nil
}

// RotateRefreshToken atomically inserts the successor refresh token and marks the predecessor as replaced/revoked.
func (r *RefreshTokenRepo) RotateRefreshToken(ctx context.Context, oldTokenID string, newRecord domain.RefreshTokenRecord, rotatedAt time.Time) error {
	return platformpg.WithinTransaction(ctx, r.pool, func(txCtx context.Context) error {
		conn := platformpg.Conn(txCtx, r.db)

		const insertQuery = `
			INSERT INTO refresh_tokens (id, user_id, family_id, token_hash, replaced_by_id, expires_at, revoked_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		if _, err := conn.Exec(txCtx, insertQuery,
			newRecord.ID,
			newRecord.UserID,
			newRecord.FamilyID,
			newRecord.TokenHash,
			newRecord.ReplacedByID,
			newRecord.ExpiresAt,
			newRecord.RevokedAt,
			newRecord.CreatedAt,
		); err != nil {
			return fmt.Errorf("refresh_token_postgres_repo: insert rotated refresh token: %w", err)
		}

		const updateQuery = `
			UPDATE refresh_tokens
			SET revoked_at = $2,
			    replaced_by_id = $3
			WHERE id = $1 AND revoked_at IS NULL AND replaced_by_id IS NULL
		`
		tag, err := conn.Exec(txCtx, updateQuery, oldTokenID, rotatedAt, newRecord.ID)
		if err != nil {
			return fmt.Errorf("refresh_token_postgres_repo: mark old refresh token rotated: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrRefreshTokenReuseDetected
		}
		return nil
	})
}

// RevokeRefreshTokenByHash marks a refresh token as revoked.
func (r *RefreshTokenRepo) RevokeRefreshTokenByHash(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE token_hash = $1 AND revoked_at IS NULL
	`
	_, err := platformpg.Conn(ctx, r.db).Exec(ctx, query, tokenHash, revokedAt)
	if err != nil {
		return fmt.Errorf("refresh_token_postgres_repo: revoke refresh token: %w", err)
	}
	return nil
}

// RevokeRefreshTokenFamily revokes all active refresh tokens belonging to familyID (compromise mitigation on token reuse).
func (r *RefreshTokenRepo) RevokeRefreshTokenFamily(ctx context.Context, familyID string, revokedAt time.Time) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE family_id = $1 AND revoked_at IS NULL
	`
	_, err := platformpg.Conn(ctx, r.db).Exec(ctx, query, familyID, revokedAt)
	if err != nil {
		return fmt.Errorf("refresh_token_postgres_repo: revoke refresh token family: %w", err)
	}
	return nil
}

// RevokeAllUserRefreshTokens revokes all active refresh tokens belonging to userID.
func (r *RefreshTokenRepo) RevokeAllUserRefreshTokens(ctx context.Context, userID string, revokedAt time.Time) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	_, err := platformpg.Conn(ctx, r.db).Exec(ctx, query, userID, revokedAt)
	if err != nil {
		return fmt.Errorf("refresh_token_postgres_repo: revoke all user refresh tokens: %w", err)
	}
	return nil
}
