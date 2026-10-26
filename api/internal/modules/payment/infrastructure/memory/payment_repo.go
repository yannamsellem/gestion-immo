package memory

import (
	"context"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
)

// PaymentRepo is a thread-safe in-memory implementation of domain.PaymentRepository for tests.
type PaymentRepo struct {
	mu       sync.RWMutex
	payments map[string]domain.Payment
	payOrder []string
}

// NewPaymentRepo constructs an empty thread-safe in-memory PaymentRepo.
func NewPaymentRepo() *PaymentRepo {
	return &PaymentRepo{
		payments: make(map[string]domain.Payment),
	}
}

// SavePayment stores a payment and its allocations in memory.
func (r *PaymentRepo) SavePayment(_ context.Context, payment domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	copyP := payment
	copyP.Allocations = append([]domain.RentAllocation(nil), payment.Allocations...)
	if _, exists := r.payments[payment.ID]; !exists {
		r.payOrder = append(r.payOrder, payment.ID)
	}
	r.payments[payment.ID] = copyP
	return nil
}

// FindPaymentByID retrieves a payment by ID from memory.
func (r *PaymentRepo) FindPaymentByID(_ context.Context, id string) (*domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.payments[id]
	if !ok {
		return nil, domain.ErrPaymentNotFound
	}
	copyP := p
	copyP.Allocations = append([]domain.RentAllocation(nil), p.Allocations...)
	return &copyP, nil
}

// ListPaymentsByLease lists all payments for leaseID in insertion order.
func (r *PaymentRepo) ListPaymentsByLease(_ context.Context, leaseID string) ([]domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.Payment
	for _, id := range r.payOrder {
		p := r.payments[id]
		if p.LeaseID == leaseID {
			copyP := p
			copyP.Allocations = append([]domain.RentAllocation(nil), p.Allocations...)
			out = append(out, copyP)
		}
	}
	return out, nil
}

// ListPaymentsByEntity lists all payments for entityID in insertion order.
func (r *PaymentRepo) ListPaymentsByEntity(_ context.Context, entityID string) ([]domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.Payment
	for _, id := range r.payOrder {
		p := r.payments[id]
		if p.LegalEntityID == entityID {
			copyP := p
			copyP.Allocations = append([]domain.RentAllocation(nil), p.Allocations...)
			out = append(out, copyP)
		}
	}
	return out, nil
}
