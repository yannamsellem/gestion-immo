package memory

import (
	"context"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
)

// UnitOfWork implements domain.UnitOfWork for in-memory tests in the lease module.
type UnitOfWork struct {
	repos domain.Repositories
}

// NewUnitOfWork constructs an in-memory UnitOfWork for the lease module.
func NewUnitOfWork(
	tenantRepo domain.TenantRepository,
	leaseRepo domain.LeaseRepository,
	rentCallRepo domain.RentCallRepository,
) *UnitOfWork {
	return &UnitOfWork{
		repos: domain.Repositories{
			Tenants:   tenantRepo,
			Leases:    leaseRepo,
			RentCalls: rentCallRepo,
		},
	}
}

// WithinTransaction executes fn against the in-memory aggregate repositories.
func (u *UnitOfWork) WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos domain.Repositories) error) error {
	return fn(ctx, u.repos)
}
