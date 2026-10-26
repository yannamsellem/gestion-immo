package domain

import (
	"context"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// GLIDossier represents the Aggregate Root for an assembled 1-click GLI (Garantie Loyers Impayés) claim archive.
type GLIDossier struct {
	ID               string
	LeaseID          string
	LegalEntityID    string
	TemplateID       string
	GLIPolicyNumber  string
	TotalUnpaidCents money.AmountCents
	UnpaidMonths     []string
	ArchiveR2Key     string
	IncludedFiles    []string
	CreatedAt        time.Time
}

// GLIDossierRepository defines the persistence contract for the GLIDossier aggregate root.
type GLIDossierRepository interface {
	SaveGLIDossier(ctx context.Context, dossier GLIDossier) error
	FindGLIDossierByID(ctx context.Context, id string) (*GLIDossier, error)
	ListGLIDossiersByLease(ctx context.Context, leaseID string) ([]GLIDossier, error)
}
