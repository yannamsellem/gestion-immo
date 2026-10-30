package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
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
	trimmedEntity := strings.TrimSpace(entityID)
	if !id.IsValidUUID(trimmedEntity) {
		return fmt.Errorf("%w: invalid legal_entity_id", domain.ErrValidation)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := balanceKey(trimmedEntity, fiscalYear)
	existing := r.balanceLines[key]
	idxByLineID := make(map[string]int, len(existing))
	for i, l := range existing {
		idxByLineID[l.LineID] = i
	}
	for _, line := range lines {
		line.LegalEntityID = trimmedEntity
		line.FiscalYear = fiscalYear
		if pos, ok := idxByLineID[line.LineID]; ok {
			if strings.TrimSpace(line.ID) == "" {
				line.ID = existing[pos].ID
			}
			existing[pos] = line
		} else {
			if strings.TrimSpace(line.ID) == "" {
				newID, err := id.NewV7()
				if err != nil {
					return err
				}
				line.ID = newID
			}
			idxByLineID[line.LineID] = len(existing)
			existing = append(existing, line)
		}
	}
	r.balanceLines[key] = existing
	return nil
}

// ListAccountBalanceLines returns all AccountBalanceLines for an entity and fiscal year.
func (r *AccountBalanceRepo) ListAccountBalanceLines(_ context.Context, entityID string, fiscalYear int) ([]domain.AccountBalanceLine, error) {
	trimmedEntity := strings.TrimSpace(entityID)
	if !id.IsValidUUID(trimmedEntity) {
		return []domain.AccountBalanceLine{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := balanceKey(trimmedEntity, fiscalYear)
	list := r.balanceLines[key]
	out := make([]domain.AccountBalanceLine, len(list))
	copy(out, list)
	return out, nil
}
