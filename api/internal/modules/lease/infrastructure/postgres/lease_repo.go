package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// LeaseRepo implements domain.LeaseRepository against PostgreSQL.
type LeaseRepo struct {
	db postgres.DBTX
}

// NewLeaseRepo constructs a PostgreSQL repository dedicated to Lease.
func NewLeaseRepo(db postgres.DBTX) *LeaseRepo {
	return &LeaseRepo{db: db}
}

// CreateLease inserts a new lease row into PostgreSQL.
func (r *LeaseRepo) CreateLease(ctx context.Context, lease domain.Lease) error {
	const query = `
		INSERT INTO leases (
			id, legal_entity_id, lot_id, tenant_id, lease_type,
			rent_net_cents, charges_cents, subject_to_vat, vat_rate_basis_points,
			security_deposit_cents, due_day_of_month, gli_policy_number, guarantor_name,
			start_date, end_date, active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17, $18
		)
	`
	_, err := postgres.Conn(ctx, r.db).Exec(ctx, query,
		lease.ID,
		lease.LegalEntityID,
		lease.LotID,
		lease.TenantID,
		string(lease.LeaseType),
		int64(lease.RentNetCents),
		int64(lease.ChargesCents),
		lease.SubjectToVAT,
		lease.VATRateBasisPoints,
		int64(lease.SecurityDepositCents),
		lease.DueDayOfMonth,
		lease.GLIPolicyNumber,
		lease.GuarantorName,
		lease.StartDate,
		lease.EndDate,
		lease.Active,
		lease.CreatedAt,
		lease.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("lease/postgres/lease_repo: create lease: %w", err)
	}
	return nil
}

// FindLeaseByID retrieves a lease by primary key.
func (r *LeaseRepo) FindLeaseByID(ctx context.Context, id string) (*domain.Lease, error) {
	const query = `
		SELECT id, legal_entity_id, lot_id, tenant_id, lease_type,
		       rent_net_cents, charges_cents, subject_to_vat, vat_rate_basis_points,
		       security_deposit_cents, due_day_of_month, gli_policy_number, guarantor_name,
		       start_date, end_date, active, created_at, updated_at
		FROM leases
		WHERE id = $1
	`
	row := postgres.Conn(ctx, r.db).QueryRow(ctx, query, id)
	return scanLease(row)
}

// ListLeases lists all leases ordered by creation date.
func (r *LeaseRepo) ListLeases(ctx context.Context) ([]domain.Lease, error) {
	const query = `
		SELECT id, legal_entity_id, lot_id, tenant_id, lease_type,
		       rent_net_cents, charges_cents, subject_to_vat, vat_rate_basis_points,
		       security_deposit_cents, due_day_of_month, gli_policy_number, guarantor_name,
		       start_date, end_date, active, created_at, updated_at
		FROM leases
		ORDER BY created_at ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("lease/postgres/lease_repo: list leases: %w", err)
	}
	defer rows.Close()

	var leases []domain.Lease
	for rows.Next() {
		l, err := scanLease(rows)
		if err != nil {
			return nil, err
		}
		leases = append(leases, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lease/postgres/lease_repo: iterate leases: %w", err)
	}
	return leases, nil
}

// ListLeasesByEntity lists all leases belonging to a legal entity.
func (r *LeaseRepo) ListLeasesByEntity(ctx context.Context, entityID string) ([]domain.Lease, error) {
	const query = `
		SELECT id, legal_entity_id, lot_id, tenant_id, lease_type,
		       rent_net_cents, charges_cents, subject_to_vat, vat_rate_basis_points,
		       security_deposit_cents, due_day_of_month, gli_policy_number, guarantor_name,
		       start_date, end_date, active, created_at, updated_at
		FROM leases
		WHERE legal_entity_id = $1
		ORDER BY created_at ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, entityID)
	if err != nil {
		return nil, fmt.Errorf("lease/postgres/lease_repo: list leases by entity: %w", err)
	}
	defer rows.Close()

	var leases []domain.Lease
	for rows.Next() {
		l, err := scanLease(rows)
		if err != nil {
			return nil, err
		}
		leases = append(leases, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lease/postgres/lease_repo: iterate leases by entity: %w", err)
	}
	return leases, nil
}

func scanLease(row pgx.Row) (*domain.Lease, error) {
	var l domain.Lease
	var leaseTypeStr string
	var rentNet, charges, deposit int64
	err := row.Scan(
		&l.ID,
		&l.LegalEntityID,
		&l.LotID,
		&l.TenantID,
		&leaseTypeStr,
		&rentNet,
		&charges,
		&l.SubjectToVAT,
		&l.VATRateBasisPoints,
		&deposit,
		&l.DueDayOfMonth,
		&l.GLIPolicyNumber,
		&l.GuarantorName,
		&l.StartDate,
		&l.EndDate,
		&l.Active,
		&l.CreatedAt,
		&l.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrLeaseNotFound
		}
		return nil, fmt.Errorf("lease/postgres/lease_repo: scan lease: %w", err)
	}
	l.LeaseType = domain.LeaseType(leaseTypeStr)
	l.RentNetCents = money.AmountCents(rentNet)
	l.ChargesCents = money.AmountCents(charges)
	l.SecurityDepositCents = money.AmountCents(deposit)
	return &l, nil
}
