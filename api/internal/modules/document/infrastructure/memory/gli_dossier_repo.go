package memory

import (
	"context"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/document/domain"
)

// GLIDossierRepo is a thread-safe in-memory implementation of domain.GLIDossierRepository for tests.
type GLIDossierRepo struct {
	mu           sync.RWMutex
	gliDossiers  map[string]domain.GLIDossier
	dossierOrder []string
}

// NewGLIDossierRepo constructs an empty thread-safe in-memory GLIDossierRepo.
func NewGLIDossierRepo() *GLIDossierRepo {
	return &GLIDossierRepo{
		gliDossiers: make(map[string]domain.GLIDossier),
	}
}

// SaveGLIDossier stores a GLIDossier record in memory.
func (r *GLIDossierRepo) SaveGLIDossier(_ context.Context, dossier domain.GLIDossier) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	copyD := dossier
	copyD.UnpaidMonths = append([]string(nil), dossier.UnpaidMonths...)
	copyD.IncludedFiles = append([]string(nil), dossier.IncludedFiles...)
	if _, exists := r.gliDossiers[dossier.ID]; !exists {
		r.dossierOrder = append(r.dossierOrder, dossier.ID)
	}
	r.gliDossiers[dossier.ID] = copyD
	return nil
}

// FindGLIDossierByID retrieves a GLIDossier by ID from memory.
func (r *GLIDossierRepo) FindGLIDossierByID(_ context.Context, id string) (*domain.GLIDossier, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	d, ok := r.gliDossiers[id]
	if !ok {
		return nil, domain.ErrGLIDossierNotFound
	}
	copyD := d
	copyD.UnpaidMonths = append([]string(nil), d.UnpaidMonths...)
	copyD.IncludedFiles = append([]string(nil), d.IncludedFiles...)
	return &copyD, nil
}

// ListGLIDossiersByLease returns all GLIDossiers generated for leaseID.
func (r *GLIDossierRepo) ListGLIDossiersByLease(_ context.Context, leaseID string) ([]domain.GLIDossier, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.GLIDossier
	for _, id := range r.dossierOrder {
		d := r.gliDossiers[id]
		if d.LeaseID == leaseID {
			copyD := d
			copyD.UnpaidMonths = append([]string(nil), d.UnpaidMonths...)
			copyD.IncludedFiles = append([]string(nil), d.IncludedFiles...)
			out = append(out, copyD)
		}
	}
	return out, nil
}
