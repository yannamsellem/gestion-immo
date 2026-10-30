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

// JournalEntryRepo implements domain.JournalEntryRepository using postgres.DBTX.
type JournalEntryRepo struct {
	pool *pgxpool.Pool
	db   postgres.DBTX
}

// NewJournalEntryRepo constructs a new PostgreSQL repository for JournalEntry aggregates.
func NewJournalEntryRepo(pool *pgxpool.Pool, db postgres.DBTX) *JournalEntryRepo {
	if db == nil {
		db = pool
	}
	return &JournalEntryRepo{pool: pool, db: db}
}

// SaveJournalEntry inserts a JournalEntry aggregate root and its child JournalLines atomically.
func (r *JournalEntryRepo) SaveJournalEntry(ctx context.Context, entry domain.JournalEntry) error {
	return postgres.WithinTransaction(ctx, r.pool, func(txCtx context.Context) error {
		conn := postgres.Conn(txCtx, r.db)

		const insertEntry = `
			INSERT INTO journal_entries (
				id, legal_entity_id, fiscal_year, journal_code, journal_label,
				entry_number, entry_date, document_ref, document_date, label, validation_date, created_at
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10, $11, $12
			)
		`
		if _, err := conn.Exec(txCtx, insertEntry,
			entry.ID,
			entry.LegalEntityID,
			entry.FiscalYear,
			entry.JournalCode,
			entry.JournalLabel,
			entry.EntryNumber,
			entry.EntryDate,
			entry.DocumentRef,
			entry.DocumentDate,
			entry.Label,
			entry.ValidationDate,
			entry.CreatedAt,
		); err != nil {
			if postgres.IsForeignKeyViolation(err) {
				return fmt.Errorf("%w: referenced legal entity does not exist", domain.ErrValidation)
			}
			return fmt.Errorf("accounting/postgres/journal_entry_repo: insert journal entry %s: %w", entry.ID, err)
		}

		const insertLine = `
			INSERT INTO journal_lines (
				id, journal_entry_id, legal_entity_id, fiscal_year, line_id,
				account_code, account_label, sub_account_code, sub_account_label, section, label,
				debit_cents, credit_cents, net_n_minus_1_cents, reconciliation_code, reconciliation_date,
				foreign_currency_amount_cents, currency_code, created_at
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10, $11,
				$12, $13, $14, $15, $16,
				$17, $18, $19
			)
		`
		if len(entry.Lines) > 0 {
			batch := &pgx.Batch{}
			for _, line := range entry.Lines {
				batch.Queue(insertLine,
					line.ID,
					entry.ID,
					entry.LegalEntityID,
					entry.FiscalYear,
					line.LineID,
					line.AccountCode,
					line.AccountLabel,
					line.SubAccountCode,
					line.SubAccountLabel,
					line.Section,
					line.Label,
					int64(line.DebitCents),
					int64(line.CreditCents),
					int64(line.NetNMinus1Cents),
					line.ReconciliationCode,
					line.ReconciliationDate,
					int64(line.ForeignCurrencyAmountCents),
					line.CurrencyCode,
					line.CreatedAt,
				)
			}

			br := conn.SendBatch(txCtx, batch)
			for _, line := range entry.Lines {
				if _, err := br.Exec(); err != nil {
					_ = br.Close()
					if postgres.IsForeignKeyViolation(err) {
						return fmt.Errorf("%w: referenced record does not exist", domain.ErrValidation)
					}
					return fmt.Errorf("accounting/postgres/journal_entry_repo: insert journal line %s: %w", line.ID, err)
				}
			}
			if err := br.Close(); err != nil {
				if postgres.IsForeignKeyViolation(err) {
					return fmt.Errorf("%w: referenced record does not exist", domain.ErrValidation)
				}
				return fmt.Errorf("accounting/postgres/journal_entry_repo: close batch: %w", err)
			}
		}

		return nil
	})
}

// ListJournalEntries returns all JournalEntries and their JournalLines for an entity and fiscal year.
func (r *JournalEntryRepo) ListJournalEntries(ctx context.Context, entityID string, fiscalYear int) ([]domain.JournalEntry, error) {
	if !id.IsValidUUID(entityID) {
		return []domain.JournalEntry{}, nil
	}
	const query = `
		SELECT id, legal_entity_id, fiscal_year, journal_code, journal_label,
		       entry_number, entry_date, document_ref, document_date, label, validation_date, created_at
		FROM journal_entries
		WHERE legal_entity_id = $1 AND ($2 <= 0 OR fiscal_year = $2)
		ORDER BY entry_date ASC, entry_number ASC, created_at ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, entityID, fiscalYear)
	if err != nil {
		return nil, fmt.Errorf("accounting/postgres/journal_entry_repo: list journal entries: %w", err)
	}
	defer rows.Close()

	var entries []domain.JournalEntry
	for rows.Next() {
		var e domain.JournalEntry
		if err := rows.Scan(
			&e.ID,
			&e.LegalEntityID,
			&e.FiscalYear,
			&e.JournalCode,
			&e.JournalLabel,
			&e.EntryNumber,
			&e.EntryDate,
			&e.DocumentRef,
			&e.DocumentDate,
			&e.Label,
			&e.ValidationDate,
			&e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("accounting/postgres/journal_entry_repo: scan journal entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if len(entries) == 0 {
		return entries, nil
	}

	entryIDs := make([]string, len(entries))
	for i := range entries {
		entryIDs[i] = entries[i].ID
	}

	linesByEntry, err := r.loadJournalLinesByEntryIDs(ctx, entryIDs)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		entries[i].Lines = linesByEntry[entries[i].ID]
	}
	return entries, nil
}

func (r *JournalEntryRepo) loadJournalLinesByEntryIDs(ctx context.Context, entryIDs []string) (map[string][]domain.JournalLine, error) {
	validIDs := make([]string, 0, len(entryIDs))
	for _, entryID := range entryIDs {
		if id.IsValidUUID(entryID) {
			validIDs = append(validIDs, entryID)
		}
	}
	if len(validIDs) == 0 {
		return map[string][]domain.JournalLine{}, nil
	}

	const query = `
		SELECT id, journal_entry_id, legal_entity_id, fiscal_year, line_id,
		       account_code, account_label, sub_account_code, sub_account_label, section, label,
		       debit_cents, credit_cents, net_n_minus_1_cents, reconciliation_code, reconciliation_date,
		       foreign_currency_amount_cents, currency_code, created_at
		FROM journal_lines
		WHERE journal_entry_id = ANY($1::uuid[])
		ORDER BY id ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, validIDs)
	if err != nil {
		return nil, fmt.Errorf("accounting/postgres/journal_entry_repo: load journal lines: %w", err)
	}
	defer rows.Close()

	grouped := make(map[string][]domain.JournalLine, len(validIDs))
	for rows.Next() {
		var l domain.JournalLine
		var debit, credit, nMinus1, foreignCurr int64
		if err := rows.Scan(
			&l.ID,
			&l.JournalEntryID,
			&l.LegalEntityID,
			&l.FiscalYear,
			&l.LineID,
			&l.AccountCode,
			&l.AccountLabel,
			&l.SubAccountCode,
			&l.SubAccountLabel,
			&l.Section,
			&l.Label,
			&debit,
			&credit,
			&nMinus1,
			&l.ReconciliationCode,
			&l.ReconciliationDate,
			&foreignCurr,
			&l.CurrencyCode,
			&l.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("accounting/postgres/journal_entry_repo: scan journal line: %w", err)
		}
		l.DebitCents = money.AmountCents(debit)
		l.CreditCents = money.AmountCents(credit)
		l.NetNMinus1Cents = money.AmountCents(nMinus1)
		l.ForeignCurrencyAmountCents = money.AmountCents(foreignCurr)
		grouped[l.JournalEntryID] = append(grouped[l.JournalEntryID], l)
	}
	return grouped, rows.Err()
}

// ListFECEntries returns all 18-column FEC entries for an entity and fiscal year ordered chronologically.
func (r *JournalEntryRepo) ListFECEntries(ctx context.Context, entityID string, fiscalYear int) ([]domain.FECEntry, error) {
	if !id.IsValidUUID(entityID) {
		return []domain.FECEntry{}, nil
	}
	const query = `
		SELECT e.journal_code, e.journal_label, e.entry_number, e.entry_date,
		       l.account_code, l.account_label, l.sub_account_code, l.sub_account_label,
		       e.document_ref, e.document_date, COALESCE(NULLIF(l.label, ''), e.label),
		       l.debit_cents, l.credit_cents, l.reconciliation_code, l.reconciliation_date,
		       e.validation_date, l.foreign_currency_amount_cents, l.currency_code
		FROM journal_lines l
		INNER JOIN journal_entries e ON e.id = l.journal_entry_id
		WHERE e.legal_entity_id = $1 AND ($2 <= 0 OR e.fiscal_year = $2)
		ORDER BY e.entry_date ASC, e.entry_number ASC, l.id ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, entityID, fiscalYear)
	if err != nil {
		return nil, fmt.Errorf("accounting/postgres/journal_entry_repo: list fec entries: %w", err)
	}
	defer rows.Close()

	var out []domain.FECEntry
	for rows.Next() {
		var row domain.FECEntry
		var debit, credit, foreignCurr int64
		if err := rows.Scan(
			&row.JournalCode,
			&row.JournalLabel,
			&row.EntryNumber,
			&row.EntryDate,
			&row.AccountCode,
			&row.AccountLabel,
			&row.SubAccountCode,
			&row.SubAccountLabel,
			&row.DocumentRef,
			&row.DocumentDate,
			&row.EntryLabel,
			&debit,
			&credit,
			&row.ReconciliationCode,
			&row.ReconciliationDate,
			&row.ValidationDate,
			&foreignCurr,
			&row.CurrencyCode,
		); err != nil {
			return nil, fmt.Errorf("accounting/postgres/journal_entry_repo: scan fec entry: %w", err)
		}
		row.DebitCents = money.AmountCents(debit)
		row.CreditCents = money.AmountCents(credit)
		row.ForeignCurrencyAmountCents = money.AmountCents(foreignCurr)
		out = append(out, row)
	}
	return out, rows.Err()
}
