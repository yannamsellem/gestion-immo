package memory

import (
	"context"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
)

// UnitOfWork implements domain.UnitOfWork in memory for unit and integration tests.
type UnitOfWork struct {
	users         domain.UserRepository
	refreshTokens domain.RefreshTokenRepository
	mcpTokens     domain.MCPTokenRepository
}

// NewUnitOfWork constructs a new in-memory UnitOfWork for the auth module.
func NewUnitOfWork(
	users domain.UserRepository,
	refreshTokens domain.RefreshTokenRepository,
	mcpTokens domain.MCPTokenRepository,
) *UnitOfWork {
	return &UnitOfWork{
		users:         users,
		refreshTokens: refreshTokens,
		mcpTokens:     mcpTokens,
	}
}

// WithinTransaction executes fn with the in-memory aggregate repositories.
func (u *UnitOfWork) WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos domain.Repositories) error) error {
	return fn(ctx, domain.Repositories{
		Users:         u.users,
		RefreshTokens: u.refreshTokens,
		MCPTokens:     u.mcpTokens,
	})
}
