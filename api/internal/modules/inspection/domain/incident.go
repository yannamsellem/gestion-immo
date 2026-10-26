package domain

import (
	"context"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// IncidentStatus tracks the lifecycle of a property/lease incident or insurance claim.
type IncidentStatus string

const (
	IncidentStatusOpen           IncidentStatus = "OPEN"
	IncidentStatusInsuranceClaim IncidentStatus = "INSURANCE_CLAIM"
	IncidentStatusWorkInProgress IncidentStatus = "WORK_IN_PROGRESS"
	IncidentStatusClosed         IncidentStatus = "CLOSED"
)

// IsValid reports whether s is a valid IncidentStatus.
func (s IncidentStatus) IsValid() bool {
	switch s {
	case IncidentStatusOpen, IncidentStatusInsuranceClaim, IncidentStatusWorkInProgress, IncidentStatusClosed:
		return true
	default:
		return false
	}
}

// Incident represents the Aggregate Root for a maintenance incident, damage report, or insurance claim
// linked to a property, lot, or lease.
type Incident struct {
	ID                   string
	PropertyID           string
	LotID                string
	LeaseID              string
	Title                string
	Status               IncidentStatus
	InsuranceClaimNumber string
	EstimatedCostCents   money.AmountCents
	Observations         []string
	PhotoR2Keys          []string
	OccurredAt           time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// IncidentRepository defines the persistence contract for the Incident aggregate root.
type IncidentRepository interface {
	SaveIncident(ctx context.Context, inc Incident) error
	FindIncidentByID(ctx context.Context, id string) (*Incident, error)
	ListIncidents(ctx context.Context, propertyID, leaseID string) ([]Incident, error)
}
