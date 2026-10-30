package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// PaymentRepo implements domain.PaymentRepository using postgres.DBTX.
type PaymentRepo struct {
	pool *pgxpool.Pool
	db   postgres.DBTX
}

// NewPaymentRepo constructs a new PostgreSQL repository for Payment aggregates.
func NewPaymentRepo(pool *pgxpool.Pool, db postgres.DBTX) *PaymentRepo {
	if db == nil {
		db = pool
	}
	return &PaymentRepo{pool: pool, db: db}
}

// SavePayment inserts a Payment aggregate root and its child RentAllocations atomically.
// If ctx already carries an active UnitOfWork transaction, it reuses that transaction;
// otherwise it opens an aggregate-scoped transaction.
func (r *PaymentRepo) SavePayment(ctx context.Context, payment domain.Payment) error {
	return postgres.WithinTransaction(ctx, r.pool, func(txCtx context.Context) error {
		conn := postgres.Conn(txCtx, r.db)

		const insertPaymentQuery = `
			INSERT INTO payments (
				id, legal_entity_id, lease_id, tenant_id,
				amount_cents, unallocated_cents, method, label,
				external_ref, receipt_number, payment_date, created_at
			) VALUES (
				$1, $2, $3, $4,
				$5, $6, $7, $8,
				$9, $10, $11, $12
			)
		`
		_, err := conn.Exec(txCtx, insertPaymentQuery,
			payment.ID,
			payment.LegalEntityID,
			payment.LeaseID,
			payment.TenantID,
			int64(payment.AmountCents),
			int64(payment.UnallocatedCents),
			string(payment.Method),
			payment.Label,
			payment.ExternalRef,
			payment.ReceiptNumber,
			payment.PaymentDate,
			payment.CreatedAt,
		)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				return fmt.Errorf("%w: referenced record does not exist", domain.ErrValidation)
			}
			return fmt.Errorf("payment/postgres/payment_repo: insert payment %s: %w", payment.ID, err)
		}

		if len(payment.Allocations) > 0 {
			const insertAllocationQuery = `
				INSERT INTO rent_allocations (
					id, payment_id, rent_call_id, period_month, applied_cents, created_at
				) VALUES ($1, $2, $3, $4, $5, $6)
			`
			batch := &pgx.Batch{}
			for _, alloc := range payment.Allocations {
				batch.Queue(insertAllocationQuery,
					alloc.ID,
					payment.ID,
					alloc.RentCallID,
					alloc.PeriodMonth,
					int64(alloc.AppliedCents),
					alloc.CreatedAt,
				)
			}

			br := conn.SendBatch(txCtx, batch)
			for _, alloc := range payment.Allocations {
				if _, err := br.Exec(); err != nil {
					_ = br.Close()
					if postgres.IsForeignKeyViolation(err) {
						return fmt.Errorf("%w: referenced record does not exist", domain.ErrValidation)
					}
					return fmt.Errorf("payment/postgres/payment_repo: insert allocation %s: %w", alloc.ID, err)
				}
			}
			if err := br.Close(); err != nil {
				if postgres.IsForeignKeyViolation(err) {
					return fmt.Errorf("%w: referenced record does not exist", domain.ErrValidation)
				}
				return fmt.Errorf("payment/postgres/payment_repo: close batch: %w", err)
			}
		}

		return nil
	})
}

// FindPaymentByID retrieves a payment and its allocations by ID.
func (r *PaymentRepo) FindPaymentByID(ctx context.Context, paymentID string) (*domain.Payment, error) {
	if !id.IsValidUUID(paymentID) {
		return nil, domain.ErrPaymentNotFound
	}
	const query = `
		SELECT id, legal_entity_id, lease_id, tenant_id,
		       amount_cents, unallocated_cents, method, label,
		       external_ref, receipt_number, payment_date, created_at
		FROM payments
		WHERE id = $1
	`
	p, err := scanPayment(postgres.Conn(ctx, r.db).QueryRow(ctx, query, paymentID))
	if err != nil {
		return nil, err
	}
	allocs, err := r.listAllocationsByPayment(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.Allocations = allocs
	return p, nil
}

// ListPaymentsByLease returns all payments for a lease ordered by payment date ascending.
func (r *PaymentRepo) ListPaymentsByLease(ctx context.Context, leaseID string) ([]domain.Payment, error) {
	if !id.IsValidUUID(leaseID) {
		return []domain.Payment{}, nil
	}
	const query = `
		SELECT id, legal_entity_id, lease_id, tenant_id,
		       amount_cents, unallocated_cents, method, label,
		       external_ref, receipt_number, payment_date, created_at
		FROM payments
		WHERE lease_id = $1
		ORDER BY payment_date ASC, created_at ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, leaseID)
	if err != nil {
		return nil, fmt.Errorf("payment/postgres/payment_repo: list payments by lease: %w", err)
	}
	defer rows.Close()

	return r.collectPaymentsWithAllocations(ctx, rows)
}

// ListPaymentsByEntity returns all payments for a legal entity ordered by payment date ascending.
func (r *PaymentRepo) ListPaymentsByEntity(ctx context.Context, entityID string) ([]domain.Payment, error) {
	if !id.IsValidUUID(entityID) {
		return []domain.Payment{}, nil
	}
	const query = `
		SELECT id, legal_entity_id, lease_id, tenant_id,
		       amount_cents, unallocated_cents, method, label,
		       external_ref, receipt_number, payment_date, created_at
		FROM payments
		WHERE legal_entity_id = $1
		ORDER BY payment_date ASC, created_at ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, entityID)
	if err != nil {
		return nil, fmt.Errorf("payment/postgres/payment_repo: list payments by entity: %w", err)
	}
	defer rows.Close()

	return r.collectPaymentsWithAllocations(ctx, rows)
}

func (r *PaymentRepo) collectPaymentsWithAllocations(ctx context.Context, rows pgx.Rows) ([]domain.Payment, error) {
	var payments []domain.Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("payment/postgres/payment_repo: iterate payments: %w", err)
	}
	rows.Close()

	if len(payments) == 0 {
		return payments, nil
	}

	paymentIDs := make([]string, len(payments))
	for i := range payments {
		paymentIDs[i] = payments[i].ID
	}

	allocsByPayment, err := r.listAllocationsByPaymentIDs(ctx, paymentIDs)
	if err != nil {
		return nil, err
	}
	for i := range payments {
		payments[i].Allocations = allocsByPayment[payments[i].ID]
	}
	return payments, nil
}

func (r *PaymentRepo) listAllocationsByPayment(ctx context.Context, paymentID string) ([]domain.RentAllocation, error) {
	grouped, err := r.listAllocationsByPaymentIDs(ctx, []string{paymentID})
	if err != nil {
		return nil, err
	}
	return grouped[paymentID], nil
}

func (r *PaymentRepo) listAllocationsByPaymentIDs(ctx context.Context, paymentIDs []string) (map[string][]domain.RentAllocation, error) {
	validIDs := make([]string, 0, len(paymentIDs))
	for _, pid := range paymentIDs {
		if id.IsValidUUID(pid) {
			validIDs = append(validIDs, pid)
		}
	}
	if len(validIDs) == 0 {
		return map[string][]domain.RentAllocation{}, nil
	}
	const query = `
		SELECT id, payment_id, rent_call_id, period_month, applied_cents, created_at
		FROM rent_allocations
		WHERE payment_id = ANY($1::uuid[])
		ORDER BY created_at ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, validIDs)
	if err != nil {
		return nil, fmt.Errorf("payment/postgres/payment_repo: list allocations for payments: %w", err)
	}
	defer rows.Close()

	grouped := make(map[string][]domain.RentAllocation, len(paymentIDs))
	for rows.Next() {
		var a domain.RentAllocation
		var applied int64
		if err := rows.Scan(&a.ID, &a.PaymentID, &a.RentCallID, &a.PeriodMonth, &applied, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("payment/postgres/payment_repo: scan allocation: %w", err)
		}
		a.AppliedCents = money.AmountCents(applied)
		grouped[a.PaymentID] = append(grouped[a.PaymentID], a)
	}
	return grouped, rows.Err()
}

func scanPayment(row pgx.Row) (*domain.Payment, error) {
	var p domain.Payment
	var amount, unalloc int64
	var methodStr string
	err := row.Scan(
		&p.ID,
		&p.LegalEntityID,
		&p.LeaseID,
		&p.TenantID,
		&amount,
		&unalloc,
		&methodStr,
		&p.Label,
		&p.ExternalRef,
		&p.ReceiptNumber,
		&p.PaymentDate,
		&p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("payment/postgres/payment_repo: scan payment: %w", err)
	}
	p.AmountCents = money.AmountCents(amount)
	p.UnallocatedCents = money.AmountCents(unalloc)
	p.Method = domain.PaymentMethod(methodStr)
	return &p, nil
}
