package postgres

import (
	"context"
	"fmt"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// AnnualAccountReportRepo implements domain.AnnualAccountReportRepository using postgres.DBTX.
type AnnualAccountReportRepo struct {
	db postgres.DBTX
}

// NewAnnualAccountReportRepo constructs a new PostgreSQL repository for AnnualAccountReport aggregates.
func NewAnnualAccountReportRepo(db postgres.DBTX) *AnnualAccountReportRepo {
	return &AnnualAccountReportRepo{db: db}
}

// SaveAnnualAccountReport persists a generated AnnualAccountReport record in PostgreSQL.
func (r *AnnualAccountReportRepo) SaveAnnualAccountReport(ctx context.Context, report domain.AnnualAccountReport) error {
	const query = `
		INSERT INTO annual_account_reports (
			id, legal_entity_id, fiscal_year, disabled_line_ids,
			net_income_n_cents, pdf_r2_key, file_name, size_bytes, created_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9
		)
	`
	disabled := report.DisabledLineIDs
	if disabled == nil {
		disabled = []string{}
	}
	_, err := postgres.Conn(ctx, r.db).Exec(ctx, query,
		report.ID,
		report.LegalEntityID,
		report.FiscalYear,
		disabled,
		int64(report.NetIncomeNCents),
		report.PDFR2Key,
		report.FileName,
		report.SizeBytes,
		report.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("accounting/postgres/annual_account_report_repo: save annual account report %s: %w", report.ID, err)
	}
	return nil
}

// ListAnnualAccountReports returns all generated AnnualAccountReports for an entity and fiscal year.
func (r *AnnualAccountReportRepo) ListAnnualAccountReports(ctx context.Context, entityID string, fiscalYear int) ([]domain.AnnualAccountReport, error) {
	const query = `
		SELECT id, legal_entity_id, fiscal_year, disabled_line_ids,
		       net_income_n_cents, pdf_r2_key, file_name, size_bytes, created_at
		FROM annual_account_reports
		WHERE legal_entity_id = $1 AND ($2 <= 0 OR fiscal_year = $2)
		ORDER BY created_at DESC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, entityID, fiscalYear)
	if err != nil {
		return nil, fmt.Errorf("accounting/postgres/annual_account_report_repo: list annual account reports: %w", err)
	}
	defer rows.Close()

	var out []domain.AnnualAccountReport
	for rows.Next() {
		var rep domain.AnnualAccountReport
		var netIncomeCents int64
		if err := rows.Scan(
			&rep.ID,
			&rep.LegalEntityID,
			&rep.FiscalYear,
			&rep.DisabledLineIDs,
			&netIncomeCents,
			&rep.PDFR2Key,
			&rep.FileName,
			&rep.SizeBytes,
			&rep.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("accounting/postgres/annual_account_report_repo: scan annual account report: %w", err)
		}
		rep.NetIncomeNCents = money.AmountCents(netIncomeCents)
		out = append(out, rep)
	}
	return out, rows.Err()
}
