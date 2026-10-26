package domain

import (
	"context"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// RentCallStatus represents the lifecycle state of a monthly rent call.
type RentCallStatus string

const (
	StatusPending RentCallStatus = "PENDING"
	StatusPartial RentCallStatus = "PARTIAL"
	StatusPaid    RentCallStatus = "PAID"
	StatusUnpaid  RentCallStatus = "UNPAID"
)

// IsValid reports whether the RentCallStatus is valid.
func (s RentCallStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusPartial, StatusPaid, StatusUnpaid:
		return true
	default:
		return false
	}
}

// RentCall is the aggregate representing a monthly rent call generated for a lease.
type RentCall struct {
	ID                string
	LeaseID           string
	LegalEntityID     string
	TenantID          string
	PeriodMonth       string
	RentNetCents      money.AmountCents
	VATCents          money.AmountCents
	ChargesCents      money.AmountCents
	TotalDueCents     money.AmountCents
	RemainingDueCents money.AmountCents
	Status            RentCallStatus
	DueDate           time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// RentCallRepository defines the persistence contract dedicated to the RentCall aggregate.
type RentCallRepository interface {
	SaveRentCalls(ctx context.Context, calls []RentCall) error
	FindRentCallByLeaseAndPeriod(ctx context.Context, leaseID, periodMonth string) (*RentCall, error)
	ListRentCallsByEntity(ctx context.Context, entityID string) ([]RentCall, error)
	ListRentCallsByLease(ctx context.Context, leaseID string) ([]RentCall, error)
	ListUnpaidRentCallsByLease(ctx context.Context, leaseID string) ([]RentCall, error)
}
