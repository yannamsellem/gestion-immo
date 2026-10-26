package domain

import "context"

// Repositories groups the aggregate repositories of the payment bounded context
// participating in a cross-aggregate Unit of Work transaction.
type Repositories struct {
	Payments    PaymentRepository
	CashJournal CashJournalRepository
}

// UnitOfWork coordinates atomic cross-aggregate transactions within the payment bounded context
// (e.g., recording a cash payment and its PCG 530 Cash Journal entry atomically).
type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos Repositories) error) error
}
