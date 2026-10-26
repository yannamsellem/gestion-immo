package infrastructure

import (
	"context"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
)

// BankProvider aliases domain.BankProvider for convenience in infrastructure callers.
type BankProvider = domain.BankProvider

// DSP2Provider is an optional Open Banking adapter that remains disabled unless explicitly configured.
type DSP2Provider struct {
	enabled      bool
	providerName string
	staticRows   []domain.BankTransactionRow
}

// NewDSP2Provider constructs an optional DSP2Provider. When enabled is false, FetchTransactions returns domain.ErrOpenBankingDisabled.
func NewDSP2Provider(enabled bool, providerName string) *DSP2Provider {
	return &DSP2Provider{
		enabled:      enabled,
		providerName: providerName,
	}
}

// Enabled reports whether Open Banking DSP2 synchronization is active.
func (p *DSP2Provider) Enabled() bool {
	return p != nil && p.enabled
}

// ProviderName returns the configured Open Banking DSP2 aggregator name (e.g., "bridge", "powens", "gocardless").
func (p *DSP2Provider) ProviderName() string {
	if p == nil {
		return ""
	}
	return p.providerName
}

// SetStaticTransactions seeds deterministic bank transactions for tests when DSP2 is enabled.
func (p *DSP2Provider) SetStaticTransactions(rows []domain.BankTransactionRow) {
	p.staticRows = append([]domain.BankTransactionRow(nil), rows...)
}

// FetchTransactions retrieves bank transactions since the given timestamp when Open Banking DSP2 is enabled.
func (p *DSP2Provider) FetchTransactions(_ context.Context, _ string, since time.Time) ([]domain.BankTransactionRow, error) {
	if p == nil || !p.enabled {
		return nil, domain.ErrOpenBankingDisabled
	}
	var out []domain.BankTransactionRow
	for _, row := range p.staticRows {
		if row.PaymentDate.IsZero() || !row.PaymentDate.Before(since) {
			out = append(out, row)
		}
	}
	return out, nil
}
