package memory

import (
	"context"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
)

// AnnualAccountReportRepo is a thread-safe in-memory implementation of domain.AnnualAccountReportRepository for tests.
type AnnualAccountReportRepo struct {
	mu      sync.RWMutex
	reports []domain.AnnualAccountReport
}

// NewAnnualAccountReportRepo constructs a new empty thread-safe in-memory AnnualAccountReportRepo.
func NewAnnualAccountReportRepo() *AnnualAccountReportRepo {
	return &AnnualAccountReportRepo{}
}

// SaveAnnualAccountReport stores an AnnualAccountReport in memory.
func (r *AnnualAccountReportRepo) SaveAnnualAccountReport(_ context.Context, report domain.AnnualAccountReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := report
	cp.DisabledLineIDs = append([]string(nil), report.DisabledLineIDs...)
	r.reports = append(r.reports, cp)
	return nil
}

// ListAnnualAccountReports returns all stored AnnualAccountReports for an entity and fiscal year.
func (r *AnnualAccountReportRepo) ListAnnualAccountReports(_ context.Context, entityID string, fiscalYear int) ([]domain.AnnualAccountReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []domain.AnnualAccountReport
	for _, rep := range r.reports {
		if rep.LegalEntityID == entityID && (fiscalYear <= 0 || rep.FiscalYear == fiscalYear) {
			cp := rep
			cp.DisabledLineIDs = append([]string(nil), rep.DisabledLineIDs...)
			out = append(out, cp)
		}
	}
	return out, nil
}
