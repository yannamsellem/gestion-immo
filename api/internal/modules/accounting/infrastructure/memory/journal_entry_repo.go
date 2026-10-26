package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
)

// JournalEntryRepo is a thread-safe in-memory implementation of domain.JournalEntryRepository for tests.
type JournalEntryRepo struct {
	mu      sync.RWMutex
	entries []domain.JournalEntry
}

// NewJournalEntryRepo constructs a new empty thread-safe in-memory JournalEntryRepo.
func NewJournalEntryRepo() *JournalEntryRepo {
	return &JournalEntryRepo{}
}

// SaveJournalEntry stores a JournalEntry in memory.
func (r *JournalEntryRepo) SaveJournalEntry(_ context.Context, entry domain.JournalEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := entry
	cp.Lines = append([]domain.JournalLine(nil), entry.Lines...)
	r.entries = append(r.entries, cp)
	return nil
}

// ListJournalEntries returns all JournalEntries for an entity and fiscal year.
func (r *JournalEntryRepo) ListJournalEntries(_ context.Context, entityID string, fiscalYear int) ([]domain.JournalEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []domain.JournalEntry
	for _, e := range r.entries {
		if e.LegalEntityID == entityID && (fiscalYear <= 0 || e.FiscalYear == fiscalYear) {
			cp := e
			cp.Lines = append([]domain.JournalLine(nil), e.Lines...)
			out = append(out, cp)
		}
	}
	return out, nil
}

// ListFECEntries builds normative 18-column FEC entries from the recorded journal entries in memory.
func (r *JournalEntryRepo) ListFECEntries(_ context.Context, entityID string, fiscalYear int) ([]domain.FECEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []domain.FECEntry
	for _, e := range r.entries {
		if e.LegalEntityID != entityID || (fiscalYear > 0 && e.FiscalYear != fiscalYear) {
			continue
		}
		for _, l := range e.Lines {
			entryLabel := l.Label
			if entryLabel == "" {
				entryLabel = e.Label
			}
			out = append(out, domain.FECEntry{
				JournalCode:                e.JournalCode,
				JournalLabel:               e.JournalLabel,
				EntryNumber:                e.EntryNumber,
				EntryDate:                  e.EntryDate,
				AccountCode:                l.AccountCode,
				AccountLabel:               l.AccountLabel,
				SubAccountCode:             l.SubAccountCode,
				SubAccountLabel:            l.SubAccountLabel,
				DocumentRef:                e.DocumentRef,
				DocumentDate:               e.DocumentDate,
				EntryLabel:                 entryLabel,
				DebitCents:                 l.DebitCents,
				CreditCents:                l.CreditCents,
				ReconciliationCode:         l.ReconciliationCode,
				ReconciliationDate:         l.ReconciliationDate,
				ValidationDate:             e.ValidationDate,
				ForeignCurrencyAmountCents: l.ForeignCurrencyAmountCents,
				CurrencyCode:               l.CurrencyCode,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].EntryDate.Equal(out[j].EntryDate) {
			return out[i].EntryDate.Before(out[j].EntryDate)
		}
		return out[i].EntryNumber < out[j].EntryNumber
	})
	return out, nil
}
