package domain

import "context"

// Repositories groups the per-aggregate repositories participating in a cross-aggregate Unit of Work transaction.
type Repositories struct {
	Users         UserRepository
	RefreshTokens RefreshTokenRepository
	MCPTokens     MCPTokenRepository
}

// UnitOfWork coordinates atomic cross-aggregate transactions within the auth bounded context
// when a use-case must mutate more than one aggregate root (e.g., rejecting a User and revoking their RefreshTokens).
type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos Repositories) error) error
}
