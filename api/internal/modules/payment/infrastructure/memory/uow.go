package memory

import (
	"context"

	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
)

// UnitOfWork implements domain.UnitOfWork for in-memory tests in the payment module.
type UnitOfWork struct {
	repos domain.Repositories
}

// NewUnitOfWork constructs an in-memory UnitOfWork for the payment module.
func NewUnitOfWork(
	paymentRepo domain.PaymentRepository,
	cashJournalRepo domain.CashJournalRepository,
) *UnitOfWork {
	return &UnitOfWork{
		repos: domain.Repositories{
			Payments:    paymentRepo,
			CashJournal: cashJournalRepo,
		},
	}
}

// WithinTransaction executes fn against the in-memory aggregate repositories.
func (u *UnitOfWork) WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos domain.Repositories) error) error {
	return fn(ctx, u.repos)
}
