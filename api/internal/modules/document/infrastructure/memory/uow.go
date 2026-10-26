package memory

import (
	"context"

	"github.com/yannamsellem/gestion-immo/internal/modules/document/domain"
)

// UnitOfWork implements domain.UnitOfWork for in-memory tests in the document module.
type UnitOfWork struct {
	repos domain.Repositories
}

// NewUnitOfWork constructs an in-memory UnitOfWork for the document module.
func NewUnitOfWork(
	templateRepo domain.DocumentTemplateRepository,
	storedFileRepo domain.StoredFileRepository,
	gliDossierRepo domain.GLIDossierRepository,
) *UnitOfWork {
	return &UnitOfWork{
		repos: domain.Repositories{
			Templates:   templateRepo,
			StoredFiles: storedFileRepo,
			GLIDossiers: gliDossierRepo,
		},
	}
}

// WithinTransaction executes fn against the in-memory aggregate repositories.
func (u *UnitOfWork) WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos domain.Repositories) error) error {
	return fn(ctx, u.repos)
}
