package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/document/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// UnitOfWork implements domain.UnitOfWork using pgx transactions.
type UnitOfWork struct {
	pool *pgxpool.Pool
}

// NewUnitOfWork constructs a new PostgreSQL UnitOfWork for the document bounded context.
func NewUnitOfWork(pool *pgxpool.Pool) *UnitOfWork {
	return &UnitOfWork{pool: pool}
}

// WithinTransaction executes fn inside a single PostgreSQL transaction with transaction-bound repositories.
func (u *UnitOfWork) WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos domain.Repositories) error) error {
	return postgres.WithinTransaction(ctx, u.pool, func(txCtx context.Context) error {
		conn := postgres.Conn(txCtx, u.pool)
		repos := domain.Repositories{
			Templates:   NewDocumentTemplateRepo(conn),
			StoredFiles: NewStoredFileRepo(conn),
			GLIDossiers: NewGLIDossierRepo(conn),
		}
		return fn(txCtx, repos)
	})
}
