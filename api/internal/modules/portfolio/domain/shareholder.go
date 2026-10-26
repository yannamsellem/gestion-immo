package domain

import (
	"context"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// Shareholder represents an associate or shareholder tracked for CERFA 2059-F / 2072 reporting,
// including exact share count, detention percentage in basis points, associate current account
// (compte courant d'associé - PCG 455), and annual remuneration paid.
type Shareholder struct {
	ID                    string
	LegalEntityID         string
	FullName              string
	SharesCount           int64
	DetentionBasisPoints  int64
	CurrentAccountCents   money.AmountCents
	RemunerationPaidCents money.AmountCents
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ComputeDetentionBasisPoints computes the ownership percentage in basis points (10000 = 100.00%)
// using half-up integer math: (sharesCount * 10000 + totalShares/2) / totalShares.
// Example: 6375 / 12875 = 4951 bps (49.51%, matching SARL INVESTISSIMO CERFA 2059-F).
func ComputeDetentionBasisPoints(sharesCount, totalShares int64) int64 {
	if totalShares <= 0 || sharesCount <= 0 {
		return 0
	}
	return (sharesCount*10000 + totalShares/2) / totalShares
}

// ShareholderRepository defines the persistence contract dedicated to the Shareholder aggregate.
type ShareholderRepository interface {
	UpsertShareholder(ctx context.Context, shareholder Shareholder) (*Shareholder, error)
	ListShareholdersByEntity(ctx context.Context, entityID string) ([]Shareholder, error)
}
