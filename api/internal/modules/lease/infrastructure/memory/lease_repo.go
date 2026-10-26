package memory

import (
	"context"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
)

// LeaseRepo is a thread-safe in-memory implementation of domain.LeaseRepository for tests.
type LeaseRepo struct {
	mu         sync.RWMutex
	leases     map[string]domain.Lease
	leaseOrder []string
}

// NewLeaseRepo constructs an empty thread-safe in-memory LeaseRepo.
func NewLeaseRepo() *LeaseRepo {
	return &LeaseRepo{
		leases: make(map[string]domain.Lease),
	}
}

// CreateLease stores a new Lease in memory.
func (r *LeaseRepo) CreateLease(_ context.Context, lease domain.Lease) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.leases[lease.ID] = lease
	r.leaseOrder = append(r.leaseOrder, lease.ID)
	return nil
}

// FindLeaseByID retrieves a Lease by ID from memory.
func (r *LeaseRepo) FindLeaseByID(_ context.Context, id string) (*domain.Lease, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	l, ok := r.leases[id]
	if !ok {
		return nil, domain.ErrLeaseNotFound
	}
	copyL := l
	return &copyL, nil
}

// ListLeases returns all Leases in insertion order.
func (r *LeaseRepo) ListLeases(_ context.Context) ([]domain.Lease, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.Lease, 0, len(r.leaseOrder))
	for _, id := range r.leaseOrder {
		out = append(out, r.leases[id])
	}
	return out, nil
}

// ListLeasesByEntity returns all Leases belonging to entityID.
func (r *LeaseRepo) ListLeasesByEntity(_ context.Context, entityID string) ([]domain.Lease, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.Lease
	for _, id := range r.leaseOrder {
		l := r.leases[id]
		if l.LegalEntityID == entityID {
			out = append(out, l)
		}
	}
	return out, nil
}
