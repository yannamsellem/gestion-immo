package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
)

// AccountBalanceRepo is a thread-safe in-memory implementation of domain.AccountBalanceRepository for tests.
type AccountBalanceRepo struct {
	mu           sync.RWMutex
	balanceLines map[string][]domain.AccountBalanceLine
}

// NewAccountBalanceRepo constructs a new empty thread-safe in-memory AccountBalanceRepo.
func NewAccountBalanceRepo() *AccountBalanceRepo {
	return &AccountBalanceRepo{
		balanceLines: make(map[string][]domain.AccountBalanceLine),
	}
}

func balanceKey(entityID string, fiscalYear int) string {
	return fmt.Sprintf("%s::%d", strings.TrimSpace(entityID), fiscalYear)
}

// UpsertAccountBalanceLines inserts or updates account balance lines in memory.
func (r *AccountBalanceRepo) UpsertAccountBalanceLines(_ context.Context, entityID string, fiscalYear int, lines []domain.AccountBalanceLine) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := balanceKey(entityID, fiscalYear)
	existing := r.balanceLines[key]
	idxByLineID := make(map[string]int, len(existing))
	for i, l := range existing {
		idxByLineID[l.LineID] = i
	}
	for _, line := range lines {
		line.LegalEntityID = entityID
		line.FiscalYear = fiscalYear
		if pos, ok := idxByLineID[line.LineID]; ok {
			existing[pos] = line
		} else {
			idxByLineID[line.LineID] = len(existing)
			existing = append(existing, line)
		}
	}
	r.balanceLines[key] = existing
	return nil
}

// ListAccountBalanceLines returns all AccountBalanceLines for an entity and fiscal year.
func (r *AccountBalanceRepo) ListAccountBalanceLines(_ context.Context, entityID string, fiscalYear int) ([]domain.AccountBalanceLine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := balanceKey(entityID, fiscalYear)
	list := r.balanceLines[key]
	out := make([]domain.AccountBalanceLine, len(list))
	copy(out, list)
	return out, nil
}
