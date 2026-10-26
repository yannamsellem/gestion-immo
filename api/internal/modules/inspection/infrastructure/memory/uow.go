package memory

import (
	"context"

	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/domain"
)

// UnitOfWork implements domain.UnitOfWork for in-memory tests in the inspection module.
type UnitOfWork struct {
	repos domain.Repositories
}

// NewUnitOfWork constructs an in-memory UnitOfWork for the inspection module.
func NewUnitOfWork(
	templateRepo domain.InspectionTemplateRepository,
	inspectionRepo domain.InspectionRepository,
	incidentRepo domain.IncidentRepository,
) *UnitOfWork {
	return &UnitOfWork{
		repos: domain.Repositories{
			Templates:   templateRepo,
			Inspections: inspectionRepo,
			Incidents:   incidentRepo,
		},
	}
}

// WithinTransaction executes fn against the in-memory aggregate repositories.
func (u *UnitOfWork) WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos domain.Repositories) error) error {
	return fn(ctx, u.repos)
}
