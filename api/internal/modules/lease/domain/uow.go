package domain

import "context"

// Repositories groups the per-aggregate repositories participating in a cross-aggregate Unit of Work transaction.
type Repositories struct {
	Tenants   TenantRepository
	Leases    LeaseRepository
	RentCalls RentCallRepository
}

// UnitOfWork coordinates atomic cross-aggregate transactions within the lease bounded context.
type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos Repositories) error) error
}
