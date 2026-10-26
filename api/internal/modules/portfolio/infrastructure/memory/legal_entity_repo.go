package memory

import (
	"context"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
)

// LegalEntityRepo is a thread-safe in-memory implementation of domain.LegalEntityRepository for tests.
type LegalEntityRepo struct {
	mu          sync.RWMutex
	entities    map[string]domain.LegalEntity
	entityOrder []string
}

// NewLegalEntityRepo constructs an empty thread-safe in-memory LegalEntityRepo.
func NewLegalEntityRepo() *LegalEntityRepo {
	return &LegalEntityRepo{
		entities: make(map[string]domain.LegalEntity),
	}
}

// CreateLegalEntity stores a new LegalEntity in memory.
func (r *LegalEntityRepo) CreateLegalEntity(_ context.Context, entity domain.LegalEntity) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entity.SIRET != "" {
		for _, existing := range r.entities {
			if existing.SIRET == entity.SIRET {
				return domain.ErrDuplicateSIRET
			}
		}
	}
	r.entities[entity.ID] = entity
	r.entityOrder = append(r.entityOrder, entity.ID)
	return nil
}

// FindLegalEntityByID retrieves a LegalEntity by ID from memory.
func (r *LegalEntityRepo) FindLegalEntityByID(_ context.Context, id string) (*domain.LegalEntity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ent, ok := r.entities[id]
	if !ok {
		return nil, domain.ErrLegalEntityNotFound
	}
	copyEnt := ent
	return &copyEnt, nil
}

// ListLegalEntities returns all LegalEntities in insertion order.
func (r *LegalEntityRepo) ListLegalEntities(_ context.Context) ([]domain.LegalEntity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.LegalEntity, 0, len(r.entityOrder))
	for _, id := range r.entityOrder {
		out = append(out, r.entities[id])
	}
	return out, nil
}
