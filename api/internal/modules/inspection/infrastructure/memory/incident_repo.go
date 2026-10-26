package memory

import (
	"context"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/domain"
)

// IncidentRepo is a thread-safe in-memory implementation of domain.IncidentRepository for tests.
type IncidentRepo struct {
	mu        sync.RWMutex
	incidents map[string]domain.Incident
	incOrder  []string
}

// NewIncidentRepo constructs an empty thread-safe in-memory IncidentRepo.
func NewIncidentRepo() *IncidentRepo {
	return &IncidentRepo{
		incidents: make(map[string]domain.Incident),
	}
}

// SaveIncident stores an Incident in memory.
func (r *IncidentRepo) SaveIncident(_ context.Context, inc domain.Incident) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	copyInc := cloneIncident(inc)
	if _, exists := r.incidents[inc.ID]; !exists {
		r.incOrder = append(r.incOrder, inc.ID)
	}
	r.incidents[inc.ID] = copyInc
	return nil
}

// FindIncidentByID retrieves an Incident by ID from memory.
func (r *IncidentRepo) FindIncidentByID(_ context.Context, id string) (*domain.Incident, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inc, ok := r.incidents[id]
	if !ok {
		return nil, domain.ErrIncidentNotFound
	}
	copyInc := cloneIncident(inc)
	return &copyInc, nil
}

// ListIncidents returns all Incidents matching optional propertyID and leaseID filters.
func (r *IncidentRepo) ListIncidents(_ context.Context, propertyID, leaseID string) ([]domain.Incident, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.Incident
	for _, id := range r.incOrder {
		inc := r.incidents[id]
		if propertyID != "" && inc.PropertyID != propertyID {
			continue
		}
		if leaseID != "" && inc.LeaseID != leaseID {
			continue
		}
		out = append(out, cloneIncident(inc))
	}
	return out, nil
}

func cloneIncident(inc domain.Incident) domain.Incident {
	copyInc := inc
	copyInc.Observations = append([]string(nil), inc.Observations...)
	copyInc.PhotoR2Keys = append([]string(nil), inc.PhotoR2Keys...)
	return copyInc
}
