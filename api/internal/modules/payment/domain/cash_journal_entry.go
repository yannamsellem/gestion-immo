package domain

import (
	"context"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// CashEntryType classifies an entry in the PCG 530 Cash Journal (Journal de Caisse).
type CashEntryType string

const (
	CashEntryReceipt     CashEntryType = "CASH_RECEIPT"
	CashEntryBankDeposit CashEntryType = "BANK_DEPOSIT"
	CashEntryExpense     CashEntryType = "CASH_EXPENSE"
)

// CashJournalEntry represents the Aggregate Root for an immutable accounting entry in the PCG 530 Cash Journal (Caisse).
// Cash receipts debit 530 (and credit 411); cash deposits to bank credit 530 (and debit 580/512).
type CashJournalEntry struct {
	ID                     string
	LegalEntityID          string
	PaymentID              *string
	AccountCode            string
	CounterpartAccountCode string
	EntryType              CashEntryType
	Label                  string
	DebitCents             money.AmountCents
	CreditCents            money.AmountCents
	ReceiptNumber          string
	EntryDate              time.Time
	CreatedAt              time.Time
}

// CashJournalRepository defines the persistence contract for the CashJournalEntry aggregate root.
type CashJournalRepository interface {
	SaveCashJournalEntry(ctx context.Context, entry CashJournalEntry) error
	ListCashJournalByEntity(ctx context.Context, entityID string) ([]CashJournalEntry, error)
}
