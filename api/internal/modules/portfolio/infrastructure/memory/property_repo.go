package memory

import (
	"context"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
)

// PropertyRepo is a thread-safe in-memory implementation of domain.PropertyRepository for tests.
type PropertyRepo struct {
	mu         sync.RWMutex
	properties map[string]domain.Property
	propsByEnt map[string][]string
}

// NewPropertyRepo constructs an empty thread-safe in-memory PropertyRepo.
func NewPropertyRepo() *PropertyRepo {
	return &PropertyRepo{
		properties: make(map[string]domain.Property),
		propsByEnt: make(map[string][]string),
	}
}

// CreateProperty stores a new Property in memory.
func (r *PropertyRepo) CreateProperty(_ context.Context, property domain.Property) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.properties[property.ID] = property
	r.propsByEnt[property.LegalEntityID] = append(r.propsByEnt[property.LegalEntityID], property.ID)
	return nil
}

// FindPropertyByID retrieves a Property by ID from memory.
func (r *PropertyRepo) FindPropertyByID(_ context.Context, id string) (*domain.Property, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.properties[id]
	if !ok {
		return nil, domain.ErrPropertyNotFound
	}
	copyProp := p
	return &copyProp, nil
}

// ListPropertiesByEntity returns all properties belonging to entityID.
func (r *PropertyRepo) ListPropertiesByEntity(_ context.Context, entityID string) ([]domain.Property, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.propsByEnt[entityID]
	out := make([]domain.Property, 0, len(ids))
	for _, id := range ids {
		out = append(out, r.properties[id])
	}
	return out, nil
}
