package domain

import "context"

// Repositories groups the aggregate repositories of the accounting bounded context
// participating in a cross-aggregate Unit of Work transaction.
type Repositories struct {
	JournalEntries  JournalEntryRepository
	AccountBalances AccountBalanceRepository
	AnnualReports   AnnualAccountReportRepository
}

// UnitOfWork coordinates atomic cross-aggregate transactions within the accounting bounded context
// (e.g., saving a balanced JournalEntry and updating AccountBalanceLines atomically).
type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos Repositories) error) error
}
