package memory

import (
	"context"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
)

// LotRepo is a thread-safe in-memory implementation of domain.LotRepository for tests.
type LotRepo struct {
	mu         sync.RWMutex
	lotsByProp map[string][]domain.Lot
}

// NewLotRepo constructs an empty thread-safe in-memory LotRepo.
func NewLotRepo() *LotRepo {
	return &LotRepo{
		lotsByProp: make(map[string][]domain.Lot),
	}
}

// CreateLot stores a new Lot in memory.
func (r *LotRepo) CreateLot(_ context.Context, lot domain.Lot) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lotsByProp[lot.PropertyID] = append(r.lotsByProp[lot.PropertyID], lot)
	return nil
}

// ListLotsByProperty returns all lots belonging to propertyID.
func (r *LotRepo) ListLotsByProperty(_ context.Context, propertyID string) ([]domain.Lot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := r.lotsByProp[propertyID]
	out := make([]domain.Lot, len(list))
	copy(out, list)
	return out, nil
}

// ListLotsByPropertyIDs returns all lots belonging to the given propertyIDs grouped by property_id.
func (r *LotRepo) ListLotsByPropertyIDs(_ context.Context, propertyIDs []string) (map[string][]domain.Lot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	grouped := make(map[string][]domain.Lot, len(propertyIDs))
	for _, propID := range propertyIDs {
		list := r.lotsByProp[propID]
		if len(list) == 0 {
			continue
		}
		out := make([]domain.Lot, len(list))
		copy(out, list)
		grouped[propID] = out
	}
	return grouped, nil
}
