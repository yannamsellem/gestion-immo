package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

// NewPool creates and pings a PostgreSQL connection pool using pgxpool.
func NewPool(ctx context.Context, dbURL secret.String) (*pgxpool.Pool, error) {
	if dbURL.IsEmpty() {
		return nil, fmt.Errorf("postgres: database URL must not be empty")
	}
	poolCfg, err := pgxpool.ParseConfig(dbURL.Value())
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to parse pool config: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping failed: %w", err)
	}
	return pool, nil
}
