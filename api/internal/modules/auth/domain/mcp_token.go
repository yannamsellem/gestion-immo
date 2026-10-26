package domain

import (
	"context"
	"time"
)

// MCPApiToken is the aggregate representing a hashed long-lived API token used by Antigravity to authenticate against /mcp.
// Plaintext MCP tokens are never stored.
type MCPApiToken struct {
	ID         string
	UserID     string
	Label      string
	TokenHash  string
	RevokedAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

// IsActive reports whether the MCP token has not been revoked.
func (t MCPApiToken) IsActive() bool {
	return t.RevokedAt == nil
}

// MCPTokenRepository defines the persistence contract dedicated to the MCPApiToken aggregate.
type MCPTokenRepository interface {
	SaveMCPToken(ctx context.Context, token MCPApiToken) error
	FindMCPTokenByHash(ctx context.Context, tokenHash string) (*MCPApiToken, error)
	RevokeAllUserMCPTokens(ctx context.Context, userID string, revokedAt time.Time) error
}
