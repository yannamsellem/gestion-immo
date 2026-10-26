package domain

import "errors"

var (
	// ErrTemplateNotFound indicates the requested DocumentTemplate does not exist.
	ErrTemplateNotFound = errors.New("document: template not found")
	// ErrStoredFileNotFound indicates the requested StoredFile does not exist.
	ErrStoredFileNotFound = errors.New("document: stored file not found")
	// ErrGLIDossierNotFound indicates the requested GLIDossier does not exist.
	ErrGLIDossierNotFound = errors.New("document: GLI dossier not found")
	// ErrNoUnpaidRentForGLI indicates a GLI dossier cannot be assembled because the lease has no unpaid rent calls.
	ErrNoUnpaidRentForGLI = errors.New("document: lease has no unpaid rent calls for GLI dossier")
	// ErrValidation indicates a domain validation failure in the document bounded context.
	ErrValidation = errors.New("document: validation error")
)
