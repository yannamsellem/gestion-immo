package domain

import (
	"context"
	"strings"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// JournalLine represents a single debit or credit line inside a double-entry JournalEntry.
// It is a child entity owned by the JournalEntry aggregate root.
type JournalLine struct {
	ID                         string            `json:"id"`
	JournalEntryID             string            `json:"journal_entry_id"`
	LegalEntityID              string            `json:"legal_entity_id"`
	FiscalYear                 int               `json:"fiscal_year"`
	LineID                     string            `json:"line_id,omitempty"`
	AccountCode                string            `json:"account_code"`
	AccountLabel               string            `json:"account_label"`
	SubAccountCode             string            `json:"sub_account_code,omitempty"`
	SubAccountLabel            string            `json:"sub_account_label,omitempty"`
	Section                    string            `json:"section,omitempty"`
	Label                      string            `json:"label"`
	DebitCents                 money.AmountCents `json:"debit_cents"`
	CreditCents                money.AmountCents `json:"credit_cents"`
	NetNMinus1Cents            money.AmountCents `json:"net_n_minus_1_cents,omitempty"`
	ReconciliationCode         string            `json:"reconciliation_code,omitempty"`
	ReconciliationDate         string            `json:"reconciliation_date,omitempty"`
	ForeignCurrencyAmountCents money.AmountCents `json:"foreign_currency_amount_cents,omitempty"`
	CurrencyCode               string            `json:"currency_code,omitempty"`
	CreatedAt                  time.Time         `json:"created_at"`
}

// JournalEntry represents the Aggregate Root for a balanced PCG accounting journal entry.
type JournalEntry struct {
	ID             string        `json:"id"`
	LegalEntityID  string        `json:"legal_entity_id"`
	FiscalYear     int           `json:"fiscal_year"`
	JournalCode    string        `json:"journal_code"`
	JournalLabel   string        `json:"journal_label"`
	EntryNumber    string        `json:"entry_number"`
	EntryDate      time.Time     `json:"entry_date"`
	DocumentRef    string        `json:"document_ref"`
	DocumentDate   time.Time     `json:"document_date"`
	Label          string        `json:"label"`
	ValidationDate time.Time     `json:"validation_date"`
	Lines          []JournalLine `json:"lines"`
	CreatedAt      time.Time     `json:"created_at"`
}

// Validate checks that a JournalEntry has at least two lines, non-negative amounts, and balanced total debits and credits.
func (e JournalEntry) Validate() error {
	if strings.TrimSpace(e.LegalEntityID) == "" {
		return ErrValidation
	}
	if len(e.Lines) < 2 {
		return ErrValidation
	}
	var totalDebit, totalCredit money.AmountCents
	for _, l := range e.Lines {
		if strings.TrimSpace(l.AccountCode) == "" {
			return ErrValidation
		}
		if l.DebitCents < 0 || l.CreditCents < 0 {
			return ErrValidation
		}
		if l.DebitCents == 0 && l.CreditCents == 0 {
			return ErrValidation
		}
		totalDebit = totalDebit.Add(l.DebitCents)
		totalCredit = totalCredit.Add(l.CreditCents)
	}
	if totalDebit != totalCredit {
		return ErrUnbalancedJournalEntry
	}
	return nil
}

// FECEntry represents a single normative DGFiP FEC (Article A.47 A-1 du LPF) line with all 18 columns.
type FECEntry struct {
	JournalCode                string            `json:"journal_code"`
	JournalLabel               string            `json:"journal_label"`
	EntryNumber                string            `json:"entry_number"`
	EntryDate                  time.Time         `json:"entry_date"`
	AccountCode                string            `json:"account_code"`
	AccountLabel               string            `json:"account_label"`
	SubAccountCode             string            `json:"sub_account_code"`
	SubAccountLabel            string            `json:"sub_account_label"`
	DocumentRef                string            `json:"document_ref"`
	DocumentDate               time.Time         `json:"document_date"`
	EntryLabel                 string            `json:"entry_label"`
	DebitCents                 money.AmountCents `json:"debit_cents"`
	CreditCents                money.AmountCents `json:"credit_cents"`
	ReconciliationCode         string            `json:"reconciliation_code"`
	ReconciliationDate         string            `json:"reconciliation_date"`
	ValidationDate             time.Time         `json:"validation_date"`
	ForeignCurrencyAmountCents money.AmountCents `json:"foreign_currency_amount_cents"`
	CurrencyCode               string            `json:"currency_code"`
}

// JournalEntryRepository defines the persistence contract for the JournalEntry aggregate root (including its JournalLines).
type JournalEntryRepository interface {
	SaveJournalEntry(ctx context.Context, entry JournalEntry) error
	ListJournalEntries(ctx context.Context, entityID string, fiscalYear int) ([]JournalEntry, error)
	ListFECEntries(ctx context.Context, entityID string, fiscalYear int) ([]FECEntry, error)
}

// FECExporterPort defines the contract for exporting normative 18-column DGFiP FEC files (Article A.47 A-1 du LPF).
type FECExporterPort interface {
	ExportFEC(entries []FECEntry) ([]byte, error)
}
