package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/yannamsellem/gestion-immo/internal/modules/document/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// GLIDossierRepo implements domain.GLIDossierRepository using postgres.DBTX.
type GLIDossierRepo struct {
	db postgres.DBTX
}

// NewGLIDossierRepo constructs a new PostgreSQL repository for GLIDossier aggregates.
func NewGLIDossierRepo(db postgres.DBTX) *GLIDossierRepo {
	return &GLIDossierRepo{db: db}
}

// SaveGLIDossier inserts a GLIDossier record into PostgreSQL.
func (r *GLIDossierRepo) SaveGLIDossier(ctx context.Context, dossier domain.GLIDossier) error {
	const query = `
		INSERT INTO gli_dossiers (
			id, lease_id, legal_entity_id, template_id, gli_policy_number,
			total_unpaid_cents, unpaid_months, archive_r2_key, included_files, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10
		)
	`
	_, err := postgres.Conn(ctx, r.db).Exec(ctx, query,
		dossier.ID,
		dossier.LeaseID,
		dossier.LegalEntityID,
		dossier.TemplateID,
		dossier.GLIPolicyNumber,
		int64(dossier.TotalUnpaidCents),
		dossier.UnpaidMonths,
		dossier.ArchiveR2Key,
		dossier.IncludedFiles,
		dossier.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("document/postgres/gli_dossier_repo: save GLI dossier %s: %w", dossier.ID, err)
	}
	return nil
}

// FindGLIDossierByID retrieves a GLIDossier by ID.
func (r *GLIDossierRepo) FindGLIDossierByID(ctx context.Context, id string) (*domain.GLIDossier, error) {
	const query = `
		SELECT id, lease_id, legal_entity_id, template_id, gli_policy_number,
		       total_unpaid_cents, unpaid_months, archive_r2_key, included_files, created_at
		FROM gli_dossiers
		WHERE id = $1
	`
	var d domain.GLIDossier
	var totalUnpaid int64
	err := postgres.Conn(ctx, r.db).QueryRow(ctx, query, id).Scan(
		&d.ID,
		&d.LeaseID,
		&d.LegalEntityID,
		&d.TemplateID,
		&d.GLIPolicyNumber,
		&totalUnpaid,
		&d.UnpaidMonths,
		&d.ArchiveR2Key,
		&d.IncludedFiles,
		&d.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrGLIDossierNotFound
		}
		return nil, fmt.Errorf("document/postgres/gli_dossier_repo: find GLI dossier by id: %w", err)
	}
	d.TotalUnpaidCents = money.AmountCents(totalUnpaid)
	return &d, nil
}

// ListGLIDossiersByLease lists all GLIDossiers generated for a lease.
func (r *GLIDossierRepo) ListGLIDossiersByLease(ctx context.Context, leaseID string) ([]domain.GLIDossier, error) {
	const query = `
		SELECT id, lease_id, legal_entity_id, template_id, gli_policy_number,
		       total_unpaid_cents, unpaid_months, archive_r2_key, included_files, created_at
		FROM gli_dossiers
		WHERE lease_id = $1
		ORDER BY created_at DESC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, leaseID)
	if err != nil {
		return nil, fmt.Errorf("document/postgres/gli_dossier_repo: list GLI dossiers: %w", err)
	}
	defer rows.Close()

	var out []domain.GLIDossier
	for rows.Next() {
		var d domain.GLIDossier
		var totalUnpaid int64
		if err := rows.Scan(
			&d.ID,
			&d.LeaseID,
			&d.LegalEntityID,
			&d.TemplateID,
			&d.GLIPolicyNumber,
			&totalUnpaid,
			&d.UnpaidMonths,
			&d.ArchiveR2Key,
			&d.IncludedFiles,
			&d.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("document/postgres/gli_dossier_repo: scan GLI dossier: %w", err)
		}
		d.TotalUnpaidCents = money.AmountCents(totalUnpaid)
		out = append(out, d)
	}
	return out, rows.Err()
}
