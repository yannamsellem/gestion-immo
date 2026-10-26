package domain

import "context"

// Repositories groups the aggregate repositories of the inspection bounded context
// participating in a cross-aggregate Unit of Work transaction.
type Repositories struct {
	Templates   InspectionTemplateRepository
	Inspections InspectionRepository
	Incidents   IncidentRepository
}

// UnitOfWork coordinates atomic cross-aggregate transactions within the inspection bounded context.
type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos Repositories) error) error
}
