package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
)

// RentCallRepo is a thread-safe in-memory implementation of domain.RentCallRepository for tests.
type RentCallRepo struct {
	mu        sync.RWMutex
	rentCalls map[string]domain.RentCall
	rcOrder   []string
}

// NewRentCallRepo constructs an empty thread-safe in-memory RentCallRepo.
func NewRentCallRepo() *RentCallRepo {
	return &RentCallRepo{
		rentCalls: make(map[string]domain.RentCall),
	}
}

// SaveRentCalls inserts or updates rent calls in memory.
func (r *RentCallRepo) SaveRentCalls(_ context.Context, calls []domain.RentCall) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, rc := range calls {
		for _, existingID := range r.rcOrder {
			existing := r.rentCalls[existingID]
			if existing.ID != rc.ID && existing.LeaseID == rc.LeaseID && existing.PeriodMonth == rc.PeriodMonth {
				return domain.ErrDuplicatePeriod
			}
		}
		if _, exists := r.rentCalls[rc.ID]; !exists {
			r.rcOrder = append(r.rcOrder, rc.ID)
		}
		r.rentCalls[rc.ID] = rc
	}
	return nil
}

// FindRentCallByLeaseAndPeriod finds a rent call matching (leaseID, periodMonth).
func (r *RentCallRepo) FindRentCallByLeaseAndPeriod(_ context.Context, leaseID, periodMonth string) (*domain.RentCall, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, id := range r.rcOrder {
		rc := r.rentCalls[id]
		if rc.LeaseID == leaseID && rc.PeriodMonth == periodMonth {
			copyRC := rc
			return &copyRC, nil
		}
	}
	return nil, domain.ErrRentCallNotFound
}

// ListRentCallsByEntity lists all rent calls belonging to entityID.
func (r *RentCallRepo) ListRentCallsByEntity(_ context.Context, entityID string) ([]domain.RentCall, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.RentCall
	for _, id := range r.rcOrder {
		rc := r.rentCalls[id]
		if rc.LegalEntityID == entityID {
			out = append(out, rc)
		}
	}
	return out, nil
}

// ListRentCallsByLease lists all rent calls belonging to leaseID.
func (r *RentCallRepo) ListRentCallsByLease(_ context.Context, leaseID string) ([]domain.RentCall, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.RentCall
	for _, id := range r.rcOrder {
		rc := r.rentCalls[id]
		if rc.LeaseID == leaseID {
			out = append(out, rc)
		}
	}
	return out, nil
}

// ListUnpaidRentCallsByLease returns all unpaid/partially paid rent calls for leaseID ordered oldest first.
func (r *RentCallRepo) ListUnpaidRentCallsByLease(_ context.Context, leaseID string) ([]domain.RentCall, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.RentCall
	for _, id := range r.rcOrder {
		rc := r.rentCalls[id]
		if rc.LeaseID == leaseID && rc.RemainingDueCents > 0 && rc.Status != domain.StatusPaid {
			out = append(out, rc)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].DueDate.Equal(out[j].DueDate) {
			return out[i].DueDate.Before(out[j].DueDate)
		}
		return out[i].PeriodMonth < out[j].PeriodMonth
	})
	return out, nil
}
