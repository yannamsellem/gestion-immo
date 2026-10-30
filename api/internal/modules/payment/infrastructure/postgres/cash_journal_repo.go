package postgres

import (
	"context"
	"fmt"

	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// CashJournalRepo implements domain.CashJournalRepository using postgres.DBTX.
type CashJournalRepo struct {
	db postgres.DBTX
}

// NewCashJournalRepo constructs a new PostgreSQL repository for CashJournalEntry aggregates.
func NewCashJournalRepo(db postgres.DBTX) *CashJournalRepo {
	return &CashJournalRepo{db: db}
}

// SaveCashJournalEntry inserts a PCG 530 Cash Journal entry into PostgreSQL.
func (r *CashJournalRepo) SaveCashJournalEntry(ctx context.Context, entry domain.CashJournalEntry) error {
	const query = `
		INSERT INTO cash_journal_entries (
			id, legal_entity_id, payment_id, account_code, counterpart_account_code,
			entry_type, label, debit_cents, credit_cents, receipt_number, entry_date, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12
		)
	`
	_, err := postgres.Conn(ctx, r.db).Exec(ctx, query,
		entry.ID,
		entry.LegalEntityID,
		entry.PaymentID,
		entry.AccountCode,
		entry.CounterpartAccountCode,
		string(entry.EntryType),
		entry.Label,
		int64(entry.DebitCents),
		int64(entry.CreditCents),
		entry.ReceiptNumber,
		entry.EntryDate,
		entry.CreatedAt,
	)
	if err != nil {
		if postgres.IsForeignKeyViolation(err) {
			return fmt.Errorf("%w: referenced record does not exist", domain.ErrValidation)
		}
		return fmt.Errorf("payment/postgres/cash_journal_repo: save cash journal entry: %w", err)
	}
	return nil
}

// ListCashJournalByEntity lists all PCG 530 Cash Journal entries for a legal entity ordered chronologically.
func (r *CashJournalRepo) ListCashJournalByEntity(ctx context.Context, entityID string) ([]domain.CashJournalEntry, error) {
	if !id.IsValidUUID(entityID) {
		return []domain.CashJournalEntry{}, nil
	}
	const query = `
		SELECT id, legal_entity_id, payment_id, account_code, counterpart_account_code,
		       entry_type, label, debit_cents, credit_cents, receipt_number, entry_date, created_at
		FROM cash_journal_entries
		WHERE legal_entity_id = $1
		ORDER BY entry_date ASC, created_at ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, entityID)
	if err != nil {
		return nil, fmt.Errorf("payment/postgres/cash_journal_repo: list cash journal: %w", err)
	}
	defer rows.Close()

	var entries []domain.CashJournalEntry
	for rows.Next() {
		var e domain.CashJournalEntry
		var entryTypeStr string
		var debit, credit int64
		err := rows.Scan(
			&e.ID,
			&e.LegalEntityID,
			&e.PaymentID,
			&e.AccountCode,
			&e.CounterpartAccountCode,
			&entryTypeStr,
			&e.Label,
			&debit,
			&credit,
			&e.ReceiptNumber,
			&e.EntryDate,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("payment/postgres/cash_journal_repo: scan cash journal entry: %w", err)
		}
		e.EntryType = domain.CashEntryType(entryTypeStr)
		e.DebitCents = money.AmountCents(debit)
		e.CreditCents = money.AmountCents(credit)
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("payment/postgres/cash_journal_repo: iterate cash journal entries: %w", err)
	}
	return entries, nil
}
