package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// UnitOfWork implements domain.UnitOfWork for atomic cross-aggregate transactions in the lease module.
type UnitOfWork struct {
	pool *pgxpool.Pool
}

// NewUnitOfWork constructs a new PostgreSQL UnitOfWork for the lease module.
func NewUnitOfWork(pool *pgxpool.Pool) *UnitOfWork {
	return &UnitOfWork{pool: pool}
}

// WithinTransaction executes fn inside a PostgreSQL transaction, providing transactional aggregate repositories.
func (u *UnitOfWork) WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos domain.Repositories) error) error {
	return postgres.WithinTransaction(ctx, u.pool, func(txCtx context.Context) error {
		txConn := postgres.Conn(txCtx, u.pool)
		repos := domain.Repositories{
			Tenants:   NewTenantRepo(txConn),
			Leases:    NewLeaseRepo(txConn),
			RentCalls: NewRentCallRepo(u.pool, txConn),
		}
		return fn(txCtx, repos)
	})
}
