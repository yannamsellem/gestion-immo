package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// AccountBalanceRepo implements domain.AccountBalanceRepository using postgres.DBTX.
type AccountBalanceRepo struct {
	pool *pgxpool.Pool
	db   postgres.DBTX
}

// NewAccountBalanceRepo constructs a new PostgreSQL repository for AccountBalanceLine aggregates.
func NewAccountBalanceRepo(pool *pgxpool.Pool, db postgres.DBTX) *AccountBalanceRepo {
	if db == nil {
		db = pool
	}
	return &AccountBalanceRepo{pool: pool, db: db}
}

// UpsertAccountBalanceLines inserts or updates account balance lines for a legal entity and fiscal year atomically.
func (r *AccountBalanceRepo) UpsertAccountBalanceLines(ctx context.Context, entityID string, fiscalYear int, lines []domain.AccountBalanceLine) error {
	if !id.IsValidUUID(entityID) {
		return fmt.Errorf("%w: invalid legal_entity_id", domain.ErrValidation)
	}
	if len(lines) == 0 {
		return nil
	}

	return postgres.WithinTransaction(ctx, r.pool, func(txCtx context.Context) error {
		conn := postgres.Conn(txCtx, r.db)

		const query = `
			INSERT INTO account_balance_lines (
				id, legal_entity_id, fiscal_year, line_id, account_code, label,
				section, cerfa_code, gross_n_cents, depreciation_n_cents, net_n_cents, net_n_minus_1_cents,
				created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11, $12,
				NOW(), NOW()
			)
			ON CONFLICT (legal_entity_id, fiscal_year, line_id) DO UPDATE SET
				account_code = EXCLUDED.account_code,
				label = EXCLUDED.label,
				section = EXCLUDED.section,
				cerfa_code = EXCLUDED.cerfa_code,
				gross_n_cents = EXCLUDED.gross_n_cents,
				depreciation_n_cents = EXCLUDED.depreciation_n_cents,
				net_n_cents = EXCLUDED.net_n_cents,
				net_n_minus_1_cents = EXCLUDED.net_n_minus_1_cents,
				updated_at = NOW()
		`
		batch := &pgx.Batch{}
		for _, line := range lines {
			lineUUID := line.ID
			if lineUUID == "" {
				var err error
				lineUUID, err = id.NewV7()
				if err != nil {
					return err
				}
			} else if !id.IsValidUUID(lineUUID) {
				return fmt.Errorf("%w: invalid account balance line id", domain.ErrValidation)
			}
			batch.Queue(query,
				lineUUID,
				entityID,
				fiscalYear,
				line.LineID,
				line.AccountCode,
				line.Label,
				line.Section,
				line.CerfaCode,
				int64(line.GrossNCents),
				int64(line.DepreciationNCents),
				int64(line.NetNCents),
				int64(line.NetNMinus1Cents),
			)
		}

		br := conn.SendBatch(txCtx, batch)
		for _, line := range lines {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				if postgres.IsForeignKeyViolation(err) {
					return fmt.Errorf("%w: referenced legal entity does not exist", domain.ErrValidation)
				}
				return fmt.Errorf("accounting/postgres/account_balance_repo: upsert account balance line %s: %w", line.LineID, err)
			}
		}
		if err := br.Close(); err != nil {
			if postgres.IsForeignKeyViolation(err) {
				return fmt.Errorf("%w: referenced legal entity does not exist", domain.ErrValidation)
			}
			return fmt.Errorf("accounting/postgres/account_balance_repo: close batch: %w", err)
		}

		return nil
	})
}

// ListAccountBalanceLines retrieves all AccountBalanceLines for a legal entity and fiscal year.
func (r *AccountBalanceRepo) ListAccountBalanceLines(ctx context.Context, entityID string, fiscalYear int) ([]domain.AccountBalanceLine, error) {
	if !id.IsValidUUID(entityID) {
		return []domain.AccountBalanceLine{}, nil
	}
	const query = `
		SELECT id, legal_entity_id, fiscal_year, line_id, account_code, label,
		       section, cerfa_code, gross_n_cents, depreciation_n_cents, net_n_cents, net_n_minus_1_cents
		FROM account_balance_lines
		WHERE legal_entity_id = $1 AND ($2 <= 0 OR fiscal_year = $2)
		ORDER BY account_code ASC, line_id ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, entityID, fiscalYear)
	if err != nil {
		return nil, fmt.Errorf("accounting/postgres/account_balance_repo: list account balance lines: %w", err)
	}
	defer rows.Close()

	var out []domain.AccountBalanceLine
	for rows.Next() {
		var l domain.AccountBalanceLine
		var gross, depr, netN, netNMinus1 int64
		if err := rows.Scan(
			&l.ID,
			&l.LegalEntityID,
			&l.FiscalYear,
			&l.LineID,
			&l.AccountCode,
			&l.Label,
			&l.Section,
			&l.CerfaCode,
			&gross,
			&depr,
			&netN,
			&netNMinus1,
		); err != nil {
			return nil, fmt.Errorf("accounting/postgres/account_balance_repo: scan account balance line: %w", err)
		}
		l.GrossNCents = money.AmountCents(gross)
		l.DepreciationNCents = money.AmountCents(depr)
		l.NetNCents = money.AmountCents(netN)
		l.NetNMinus1Cents = money.AmountCents(netNMinus1)
		out = append(out, l)
	}
	return out, rows.Err()
}
