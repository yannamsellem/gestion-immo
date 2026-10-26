package memory

import (
	"context"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
)

// UnitOfWork implements domain.UnitOfWork for in-memory tests in the accounting module.
type UnitOfWork struct {
	repos domain.Repositories
}

// NewUnitOfWork constructs an in-memory UnitOfWork for the accounting module.
func NewUnitOfWork(
	journalEntries domain.JournalEntryRepository,
	accountBalances domain.AccountBalanceRepository,
	annualReports domain.AnnualAccountReportRepository,
) *UnitOfWork {
	return &UnitOfWork{
		repos: domain.Repositories{
			JournalEntries:  journalEntries,
			AccountBalances: accountBalances,
			AnnualReports:   annualReports,
		},
	}
}

// WithinTransaction executes fn against the in-memory aggregate repositories.
func (u *UnitOfWork) WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos domain.Repositories) error) error {
	return fn(ctx, u.repos)
}
