package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// RentCallRepo implements domain.RentCallRepository against PostgreSQL.
type RentCallRepo struct {
	pool *pgxpool.Pool
	db   postgres.DBTX
}

// NewRentCallRepo constructs a PostgreSQL repository dedicated to RentCall.
func NewRentCallRepo(pool *pgxpool.Pool, db postgres.DBTX) *RentCallRepo {
	if db == nil {
		db = pool
	}
	return &RentCallRepo{pool: pool, db: db}
}

// SaveRentCalls inserts or updates a batch of rent calls atomically.
func (r *RentCallRepo) SaveRentCalls(ctx context.Context, calls []domain.RentCall) error {
	if len(calls) == 0 {
		return nil
	}

	return postgres.WithinTransaction(ctx, r.pool, func(txCtx context.Context) error {
		conn := postgres.Conn(txCtx, r.db)

		const query = `
			INSERT INTO rent_calls (
				id, lease_id, legal_entity_id, tenant_id, period_month,
				rent_net_cents, vat_cents, charges_cents, total_due_cents, remaining_due_cents,
				status, due_date, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10,
				$11, $12, $13, $14
			)
			ON CONFLICT (id) DO UPDATE
			SET remaining_due_cents = EXCLUDED.remaining_due_cents,
			    status = EXCLUDED.status,
			    updated_at = EXCLUDED.updated_at
		`
		now := time.Now().UTC()
		for _, rc := range calls {
			createdAt := rc.CreatedAt
			if createdAt.IsZero() {
				createdAt = now
			}
			updatedAt := rc.UpdatedAt
			if updatedAt.IsZero() {
				updatedAt = now
			}
			_, err := conn.Exec(txCtx, query,
				rc.ID,
				rc.LeaseID,
				rc.LegalEntityID,
				rc.TenantID,
				rc.PeriodMonth,
				int64(rc.RentNetCents),
				int64(rc.VATCents),
				int64(rc.ChargesCents),
				int64(rc.TotalDueCents),
				int64(rc.RemainingDueCents),
				string(rc.Status),
				rc.DueDate,
				createdAt,
				updatedAt,
			)
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23505" {
					return domain.ErrDuplicatePeriod
				}
				return fmt.Errorf("lease/postgres/rent_call_repo: save rent call %s: %w", rc.ID, err)
			}
		}
		return nil
	})
}

// FindRentCallByLeaseAndPeriod retrieves a rent call by (lease_id, period_month).
func (r *RentCallRepo) FindRentCallByLeaseAndPeriod(ctx context.Context, leaseID, periodMonth string) (*domain.RentCall, error) {
	const query = `
		SELECT id, lease_id, legal_entity_id, tenant_id, period_month,
		       rent_net_cents, vat_cents, charges_cents, total_due_cents, remaining_due_cents,
		       status, due_date, created_at, updated_at
		FROM rent_calls
		WHERE lease_id = $1 AND period_month = $2
	`
	row := postgres.Conn(ctx, r.db).QueryRow(ctx, query, leaseID, periodMonth)
	return scanRentCall(row)
}

// ListRentCallsByEntity lists all rent calls belonging to a legal entity.
func (r *RentCallRepo) ListRentCallsByEntity(ctx context.Context, entityID string) ([]domain.RentCall, error) {
	const query = `
		SELECT id, lease_id, legal_entity_id, tenant_id, period_month,
		       rent_net_cents, vat_cents, charges_cents, total_due_cents, remaining_due_cents,
		       status, due_date, created_at, updated_at
		FROM rent_calls
		WHERE legal_entity_id = $1
		ORDER BY due_date ASC, period_month ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, entityID)
	if err != nil {
		return nil, fmt.Errorf("lease/postgres/rent_call_repo: list rent calls by entity: %w", err)
	}
	defer rows.Close()

	return collectRentCalls(rows)
}

// ListRentCallsByLease lists all rent calls belonging to a lease.
func (r *RentCallRepo) ListRentCallsByLease(ctx context.Context, leaseID string) ([]domain.RentCall, error) {
	const query = `
		SELECT id, lease_id, legal_entity_id, tenant_id, period_month,
		       rent_net_cents, vat_cents, charges_cents, total_due_cents, remaining_due_cents,
		       status, due_date, created_at, updated_at
		FROM rent_calls
		WHERE lease_id = $1
		ORDER BY due_date ASC, period_month ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, leaseID)
	if err != nil {
		return nil, fmt.Errorf("lease/postgres/rent_call_repo: list rent calls by lease: %w", err)
	}
	defer rows.Close()

	return collectRentCalls(rows)
}

// ListUnpaidRentCallsByLease lists all unpaid/partially paid rent calls for a lease ordered oldest first.
func (r *RentCallRepo) ListUnpaidRentCallsByLease(ctx context.Context, leaseID string) ([]domain.RentCall, error) {
	const query = `
		SELECT id, lease_id, legal_entity_id, tenant_id, period_month,
		       rent_net_cents, vat_cents, charges_cents, total_due_cents, remaining_due_cents,
		       status, due_date, created_at, updated_at
		FROM rent_calls
		WHERE lease_id = $1 AND remaining_due_cents > 0 AND status <> 'PAID'
		ORDER BY due_date ASC, period_month ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, leaseID)
	if err != nil {
		return nil, fmt.Errorf("lease/postgres/rent_call_repo: list unpaid rent calls by lease: %w", err)
	}
	defer rows.Close()

	return collectRentCalls(rows)
}

func collectRentCalls(rows pgx.Rows) ([]domain.RentCall, error) {
	var calls []domain.RentCall
	for rows.Next() {
		rc, err := scanRentCall(rows)
		if err != nil {
			return nil, err
		}
		calls = append(calls, *rc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lease/postgres/rent_call_repo: iterate rent calls: %w", err)
	}
	return calls, nil
}

func scanRentCall(row pgx.Row) (*domain.RentCall, error) {
	var rc domain.RentCall
	var rentNet, vat, charges, totalDue, remainingDue int64
	var statusStr string
	err := row.Scan(
		&rc.ID,
		&rc.LeaseID,
		&rc.LegalEntityID,
		&rc.TenantID,
		&rc.PeriodMonth,
		&rentNet,
		&vat,
		&charges,
		&totalDue,
		&remainingDue,
		&statusStr,
		&rc.DueDate,
		&rc.CreatedAt,
		&rc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRentCallNotFound
		}
		return nil, fmt.Errorf("lease/postgres/rent_call_repo: scan rent call: %w", err)
	}
	rc.RentNetCents = money.AmountCents(rentNet)
	rc.VATCents = money.AmountCents(vat)
	rc.ChargesCents = money.AmountCents(charges)
	rc.TotalDueCents = money.AmountCents(totalDue)
	rc.RemainingDueCents = money.AmountCents(remainingDue)
	rc.Status = domain.RentCallStatus(statusStr)
	return &rc, nil
}
