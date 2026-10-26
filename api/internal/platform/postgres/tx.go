package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is the common query execution interface satisfied by both *pgxpool.Pool and pgx.Tx.
// Repositories depend on DBTX so they can operate transparently either standalone (on the pool)
// or inside a cross-repository Unit of Work transaction (on pgx.Tx).
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

type txContextKey struct{}

// WithTx returns a new context carrying the active pgx.Tx.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	if tx == nil {
		return ctx
	}
	return context.WithValue(ctx, txContextKey{}, tx)
}

// TxFromContext extracts the active pgx.Tx from ctx, or returns nil if no transaction is active.
func TxFromContext(ctx context.Context) pgx.Tx {
	if ctx == nil {
		return nil
	}
	tx, _ := ctx.Value(txContextKey{}).(pgx.Tx)
	return tx
}

// Conn returns the active pgx.Tx from ctx if one is present, or falls back to db.
func Conn(ctx context.Context, db DBTX) DBTX {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return db
}

// WithinTransaction executes fn inside a PostgreSQL transaction managed according to the Unit of Work pattern.
// If ctx already carries an active pgx.Tx (nested Unit of Work), fn is executed within that existing transaction.
// If pool is nil (e.g. in-memory unit tests), fn is executed directly with ctx.
func WithinTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(txCtx context.Context) error) (err error) {
	if pool == nil {
		return fn(ctx)
	}
	if existingTx := TxFromContext(ctx); existingTx != nil {
		return fn(ctx)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin unit of work transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	txCtx := WithTx(ctx, tx)
	if err = fn(txCtx); err != nil {
		return err
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return fmt.Errorf("postgres: commit unit of work transaction: %w", commitErr)
	}
	return nil
}
