package memory

import (
	"context"
	"sync"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
)

// MCPTokenRepo is a thread-safe in-memory implementation of domain.MCPTokenRepository for tests.
type MCPTokenRepo struct {
	mu        sync.RWMutex
	mcpByHash map[string]domain.MCPApiToken
}

// NewMCPTokenRepo constructs an empty thread-safe in-memory MCPTokenRepo.
func NewMCPTokenRepo() *MCPTokenRepo {
	return &MCPTokenRepo{
		mcpByHash: make(map[string]domain.MCPApiToken),
	}
}

// SaveMCPToken saves a hashed MCP API token in memory.
func (r *MCPTokenRepo) SaveMCPToken(_ context.Context, token domain.MCPApiToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.mcpByHash[token.TokenHash] = token
	return nil
}

// FindMCPTokenByHash looks up an MCP API token by its SHA-256 hash.
func (r *MCPTokenRepo) FindMCPTokenByHash(_ context.Context, tokenHash string) (*domain.MCPApiToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tok, ok := r.mcpByHash[tokenHash]
	if !ok {
		return nil, domain.ErrInvalidToken
	}
	copyTok := tok
	return &copyTok, nil
}

// RevokeAllUserMCPTokens revokes all active MCP API tokens for a user.
func (r *MCPTokenRepo) RevokeAllUserMCPTokens(_ context.Context, userID string, revokedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ts := revokedAt
	for hash, tok := range r.mcpByHash {
		if tok.UserID == userID && tok.RevokedAt == nil {
			tok.RevokedAt = &ts
			r.mcpByHash[hash] = tok
		}
	}
	return nil
}
