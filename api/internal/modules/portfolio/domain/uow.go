package domain

import "context"

// Repositories groups the per-aggregate repositories participating in a cross-aggregate Unit of Work transaction.
type Repositories struct {
	LegalEntities LegalEntityRepository
	Shareholders  ShareholderRepository
	Properties    PropertyRepository
	Lots          LotRepository
}

// UnitOfWork coordinates atomic cross-aggregate transactions within the portfolio bounded context.
type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos Repositories) error) error
}
