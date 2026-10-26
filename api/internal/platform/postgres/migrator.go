package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

// RunMigrations executes database migrations from migrationsPath in the given direction ("up" or "down").
func RunMigrations(dbURL string, migrationsPath string, direction string) error {
	if strings.TrimSpace(dbURL) == "" {
		return fmt.Errorf("postgres: database URL is required")
	}
	if strings.TrimSpace(migrationsPath) == "" {
		return fmt.Errorf("postgres: migrations path is required")
	}

	dir := strings.ToLower(strings.TrimSpace(direction))
	if dir != "up" && dir != "down" {
		return fmt.Errorf("postgres: unsupported migration direction %q (must be \"up\" or \"down\")", direction)
	}

	sourceURL := migrationsPath
	if !strings.HasPrefix(sourceURL, "file://") {
		sourceURL = "file://" + sourceURL
	}

	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		return fmt.Errorf("postgres: failed to initialize migrator: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	var runErr error
	if dir == "up" {
		runErr = m.Up()
	} else {
		runErr = m.Steps(-1)
	}
	if runErr != nil && !errors.Is(runErr, migrate.ErrNoChange) {
		return fmt.Errorf("postgres: migration %s failed: %w", dir, runErr)
	}
	return nil
}
