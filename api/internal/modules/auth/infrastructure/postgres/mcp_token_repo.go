package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	platformpg "github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// MCPTokenRepo implements domain.MCPTokenRepository against PostgreSQL.
type MCPTokenRepo struct {
	db platformpg.DBTX
}

// NewMCPTokenRepo constructs a PostgreSQL repository dedicated to the MCPApiToken aggregate.
func NewMCPTokenRepo(db platformpg.DBTX) *MCPTokenRepo {
	return &MCPTokenRepo{db: db}
}

// SaveMCPToken persists a hashed MCP API token.
func (r *MCPTokenRepo) SaveMCPToken(ctx context.Context, token domain.MCPApiToken) error {
	const query = `
		INSERT INTO mcp_api_tokens (id, user_id, label, token_hash, revoked_at, last_used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := platformpg.Conn(ctx, r.db).Exec(ctx, query,
		token.ID,
		token.UserID,
		token.Label,
		token.TokenHash,
		token.RevokedAt,
		token.LastUsedAt,
		token.CreatedAt,
	)
	if err != nil {
		if platformpg.IsForeignKeyViolation(err) {
			return fmt.Errorf("%w: referenced user does not exist", domain.ErrValidation)
		}
		return fmt.Errorf("mcp_token_postgres_repo: save mcp token: %w", err)
	}
	return nil
}

// FindMCPTokenByHash looks up an MCP API token by its SHA-256 hex digest.
func (r *MCPTokenRepo) FindMCPTokenByHash(ctx context.Context, tokenHash string) (*domain.MCPApiToken, error) {
	const query = `
		SELECT id, user_id, label, token_hash, revoked_at, last_used_at, created_at
		FROM mcp_api_tokens
		WHERE token_hash = $1
	`
	var tok domain.MCPApiToken
	err := platformpg.Conn(ctx, r.db).QueryRow(ctx, query, tokenHash).Scan(
		&tok.ID,
		&tok.UserID,
		&tok.Label,
		&tok.TokenHash,
		&tok.RevokedAt,
		&tok.LastUsedAt,
		&tok.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidToken
		}
		return nil, fmt.Errorf("mcp_token_postgres_repo: find mcp token: %w", err)
	}
	return &tok, nil
}

// RevokeAllUserMCPTokens revokes all active MCP API tokens belonging to userID.
func (r *MCPTokenRepo) RevokeAllUserMCPTokens(ctx context.Context, userID string, revokedAt time.Time) error {
	if !id.IsValidUUID(userID) {
		return nil
	}
	const query = `
		UPDATE mcp_api_tokens
		SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	_, err := platformpg.Conn(ctx, r.db).Exec(ctx, query, userID, revokedAt)
	if err != nil {
		return fmt.Errorf("mcp_token_postgres_repo: revoke all user mcp tokens: %w", err)
	}
	return nil
}
