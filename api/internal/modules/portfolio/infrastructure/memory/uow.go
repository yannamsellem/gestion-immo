package memory

import (
	"context"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
)

// UnitOfWork implements domain.UnitOfWork for in-memory tests in the portfolio module.
type UnitOfWork struct {
	repos domain.Repositories
}

// NewUnitOfWork constructs an in-memory UnitOfWork for the portfolio module.
func NewUnitOfWork(
	entityRepo domain.LegalEntityRepository,
	shareholderRepo domain.ShareholderRepository,
	propertyRepo domain.PropertyRepository,
	lotRepo domain.LotRepository,
) *UnitOfWork {
	return &UnitOfWork{
		repos: domain.Repositories{
			LegalEntities: entityRepo,
			Shareholders:  shareholderRepo,
			Properties:    propertyRepo,
			Lots:          lotRepo,
		},
	}
}

// WithinTransaction executes fn against the in-memory aggregate repositories.
func (u *UnitOfWork) WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos domain.Repositories) error) error {
	return fn(ctx, u.repos)
}
