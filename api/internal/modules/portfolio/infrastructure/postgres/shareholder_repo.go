package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// ShareholderRepo implements domain.ShareholderRepository against PostgreSQL.
type ShareholderRepo struct {
	db postgres.DBTX
}

// NewShareholderRepo constructs a PostgreSQL repository dedicated to Shareholder.
func NewShareholderRepo(db postgres.DBTX) *ShareholderRepo {
	return &ShareholderRepo{db: db}
}

// UpsertShareholder inserts or updates a shareholder row matched by (legal_entity_id, full_name).
func (r *ShareholderRepo) UpsertShareholder(ctx context.Context, sh domain.Shareholder) (*domain.Shareholder, error) {
	const query = `
		INSERT INTO shareholders (
			id, legal_entity_id, full_name, shares_count, detention_basis_points,
			current_account_cents, remuneration_paid_cents, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (legal_entity_id, full_name) DO UPDATE
		SET shares_count = EXCLUDED.shares_count,
		    detention_basis_points = EXCLUDED.detention_basis_points,
		    current_account_cents = EXCLUDED.current_account_cents,
		    remuneration_paid_cents = EXCLUDED.remuneration_paid_cents,
		    updated_at = EXCLUDED.updated_at
		RETURNING id, legal_entity_id, full_name, shares_count, detention_basis_points,
		          current_account_cents, remuneration_paid_cents, created_at, updated_at
	`
	row := postgres.Conn(ctx, r.db).QueryRow(ctx, query,
		sh.ID,
		sh.LegalEntityID,
		sh.FullName,
		sh.SharesCount,
		sh.DetentionBasisPoints,
		int64(sh.CurrentAccountCents),
		int64(sh.RemunerationPaidCents),
		sh.CreatedAt,
		sh.UpdatedAt,
	)
	return scanShareholder(row)
}

// ListShareholdersByEntity lists all shareholders belonging to a legal entity.
func (r *ShareholderRepo) ListShareholdersByEntity(ctx context.Context, entityID string) ([]domain.Shareholder, error) {
	const query = `
		SELECT id, legal_entity_id, full_name, shares_count, detention_basis_points,
		       current_account_cents, remuneration_paid_cents, created_at, updated_at
		FROM shareholders
		WHERE legal_entity_id = $1
		ORDER BY shares_count DESC, full_name ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, entityID)
	if err != nil {
		return nil, fmt.Errorf("portfolio/postgres/shareholder_repo: list shareholders: %w", err)
	}
	defer rows.Close()

	var shareholders []domain.Shareholder
	for rows.Next() {
		sh, err := scanShareholder(rows)
		if err != nil {
			return nil, err
		}
		shareholders = append(shareholders, *sh)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("portfolio/postgres/shareholder_repo: iterate shareholders: %w", err)
	}
	return shareholders, nil
}

func scanShareholder(row pgx.Row) (*domain.Shareholder, error) {
	var sh domain.Shareholder
	var currentAccountCents, remunerationPaidCents int64
	err := row.Scan(
		&sh.ID,
		&sh.LegalEntityID,
		&sh.FullName,
		&sh.SharesCount,
		&sh.DetentionBasisPoints,
		&currentAccountCents,
		&remunerationPaidCents,
		&sh.CreatedAt,
		&sh.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrShareholderNotFound
		}
		return nil, fmt.Errorf("portfolio/postgres/shareholder_repo: scan shareholder: %w", err)
	}
	sh.CurrentAccountCents = money.AmountCents(currentAccountCents)
	sh.RemunerationPaidCents = money.AmountCents(remunerationPaidCents)
	return &sh, nil
}
