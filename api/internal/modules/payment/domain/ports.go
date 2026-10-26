package domain

import (
	"context"
	"io"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// BankTransactionRow represents a normalized bank statement transaction parsed from CSV, OFX, or DSP2.
type BankTransactionRow struct {
	LeaseID     string
	AmountCents money.AmountCents
	Label       string
	PaymentDate time.Time
	ExternalRef string
}

// StatementImporter defines the port for parsing bank statement exports (CSV/OFX).
type StatementImporter interface {
	ParseTransactions(r io.Reader) ([]BankTransactionRow, error)
}

// BankProvider defines the optional Open Banking DSP2 synchronization contract (Bridge, Powens, GoCardless).
type BankProvider interface {
	Enabled() bool
	FetchTransactions(ctx context.Context, entityID string, since time.Time) ([]BankTransactionRow, error)
}
