package domain

import "context"

// Repositories groups the aggregate repositories of the document bounded context
// participating in a cross-aggregate Unit of Work transaction.
type Repositories struct {
	Templates   DocumentTemplateRepository
	StoredFiles StoredFileRepository
	GLIDossiers GLIDossierRepository
}

// UnitOfWork coordinates atomic cross-aggregate transactions within the document bounded context
// (e.g., saving both a GLIDossier aggregate and its corresponding StoredFile metadata atomically).
type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, repos Repositories) error) error
}
