package domain

import (
	"context"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// LeaseType classifies a lease agreement under French property law.
type LeaseType string

const (
	LeaseTypeResidentialUnfurnished LeaseType = "RESIDENTIAL_UNFURNISHED"
	LeaseTypeResidentialFurnished   LeaseType = "RESIDENTIAL_FURNISHED"
	LeaseTypeCommercial369          LeaseType = "COMMERCIAL_3_6_9"
	LeaseTypeProfessional           LeaseType = "PROFESSIONAL"
	LeaseTypeParking                LeaseType = "PARKING"
	LeaseTypeShortTerm              LeaseType = "SHORT_TERM"
)

// IsValid reports whether the LeaseType is a supported lease classification.
func (t LeaseType) IsValid() bool {
	switch t {
	case LeaseTypeResidentialUnfurnished,
		LeaseTypeResidentialFurnished,
		LeaseTypeCommercial369,
		LeaseTypeProfessional,
		LeaseTypeParking,
		LeaseTypeShortTerm:
		return true
	default:
		return false
	}
}

// Lease is the aggregate root representing a residential or commercial lease contract linking a LegalEntity, Lot, and Tenant,
// including optional VAT (TVA sur loyers commerciaux) and GLI (Garantie Loyers Impayés) policy metadata.
type Lease struct {
	ID                   string
	LegalEntityID        string
	LotID                string
	TenantID             string
	LeaseType            LeaseType
	RentNetCents         money.AmountCents
	ChargesCents         money.AmountCents
	SubjectToVAT         bool
	VATRateBasisPoints   int64
	SecurityDepositCents money.AmountCents
	DueDayOfMonth        int
	GLIPolicyNumber      string
	GuarantorName        string
	StartDate            time.Time
	EndDate              *time.Time
	Active               bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// ComputeRentCallAmounts computes the exact VAT in cents and Total TTC in cents for a monthly rent call.
// When SubjectToVAT is true, VAT is computed on RentNetCents using exact integer half-up rounding via
// money.ComputeVATCents(RentNetCents, VATRateBasisPoints).
func (l Lease) ComputeRentCallAmounts() (vatCents money.AmountCents, totalDueCents money.AmountCents) {
	if l.SubjectToVAT && l.VATRateBasisPoints > 0 {
		vatCents = money.ComputeVATCents(l.RentNetCents, l.VATRateBasisPoints)
	}
	totalDueCents = l.RentNetCents.Add(vatCents).Add(l.ChargesCents)
	return vatCents, totalDueCents
}

// LeaseRepository defines the persistence contract dedicated to the Lease aggregate.
type LeaseRepository interface {
	CreateLease(ctx context.Context, lease Lease) error
	FindLeaseByID(ctx context.Context, id string) (*Lease, error)
	ListLeases(ctx context.Context) ([]Lease, error)
	ListLeasesByEntity(ctx context.Context, entityID string) ([]Lease, error)
}
