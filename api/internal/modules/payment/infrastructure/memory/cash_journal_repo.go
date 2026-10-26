package memory

import (
	"context"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
)

// CashJournalRepo is a thread-safe in-memory implementation of domain.CashJournalRepository for tests.
type CashJournalRepo struct {
	mu          sync.RWMutex
	cashEntries []domain.CashJournalEntry
}

// NewCashJournalRepo constructs an empty thread-safe in-memory CashJournalRepo.
func NewCashJournalRepo() *CashJournalRepo {
	return &CashJournalRepo{}
}

// SaveCashJournalEntry stores a PCG 530 Cash Journal entry in memory.
func (r *CashJournalRepo) SaveCashJournalEntry(_ context.Context, entry domain.CashJournalEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cashEntries = append(r.cashEntries, entry)
	return nil
}

// ListCashJournalByEntity lists all PCG 530 Cash Journal entries for entityID.
func (r *CashJournalRepo) ListCashJournalByEntity(_ context.Context, entityID string) ([]domain.CashJournalEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.CashJournalEntry
	for _, e := range r.cashEntries {
		if e.LegalEntityID == entityID {
			out = append(out, e)
		}
	}
	return out, nil
}
