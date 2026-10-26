package memory

import (
	"context"
	"strings"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
)

// ShareholderRepo is a thread-safe in-memory implementation of domain.ShareholderRepository for tests.
type ShareholderRepo struct {
	mu           sync.RWMutex
	shareholders map[string][]domain.Shareholder
}

// NewShareholderRepo constructs an empty thread-safe in-memory ShareholderRepo.
func NewShareholderRepo() *ShareholderRepo {
	return &ShareholderRepo{
		shareholders: make(map[string][]domain.Shareholder),
	}
}

// UpsertShareholder creates or updates a shareholder matched by (LegalEntityID, FullName).
func (r *ShareholderRepo) UpsertShareholder(_ context.Context, shareholder domain.Shareholder) (*domain.Shareholder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	list := r.shareholders[shareholder.LegalEntityID]
	for i, existing := range list {
		if strings.EqualFold(existing.FullName, shareholder.FullName) {
			existing.SharesCount = shareholder.SharesCount
			existing.DetentionBasisPoints = shareholder.DetentionBasisPoints
			existing.CurrentAccountCents = shareholder.CurrentAccountCents
			existing.RemunerationPaidCents = shareholder.RemunerationPaidCents
			existing.UpdatedAt = shareholder.UpdatedAt
			list[i] = existing
			r.shareholders[shareholder.LegalEntityID] = list
			copySh := existing
			return &copySh, nil
		}
	}

	r.shareholders[shareholder.LegalEntityID] = append(list, shareholder)
	copySh := shareholder
	return &copySh, nil
}

// ListShareholdersByEntity returns all shareholders for a legal entity.
func (r *ShareholderRepo) ListShareholdersByEntity(_ context.Context, entityID string) ([]domain.Shareholder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := r.shareholders[entityID]
	out := make([]domain.Shareholder, len(list))
	copy(out, list)
	return out, nil
}
