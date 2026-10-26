package domain

import (
	"context"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// PaymentMethod classifies how a rent payment was received.
type PaymentMethod string

const (
	MethodCash         PaymentMethod = "CASH"
	MethodBankTransfer PaymentMethod = "BANK_TRANSFER"
	MethodCheck        PaymentMethod = "CHECK"
	MethodDirectDebit  PaymentMethod = "DIRECT_DEBIT"
)

// IsValid reports whether the PaymentMethod is supported.
func (m PaymentMethod) IsValid() bool {
	switch m {
	case MethodCash, MethodBankTransfer, MethodCheck, MethodDirectDebit:
		return true
	default:
		return false
	}
}

// RentAllocation records how a slice of a payment was applied to a specific monthly RentCall.
// It is a child entity owned by the Payment aggregate root.
type RentAllocation struct {
	ID           string            `json:"id"`
	PaymentID    string            `json:"payment_id"`
	RentCallID   string            `json:"rent_call_id"`
	PeriodMonth  string            `json:"period_month"`
	AppliedCents money.AmountCents `json:"applied_cents"`
	CreatedAt    time.Time         `json:"created_at"`
}

// Payment represents the Aggregate Root for a cash or bank rent payment received from a tenant for a lease,
// along with its child RentAllocations.
type Payment struct {
	ID               string
	LegalEntityID    string
	LeaseID          string
	TenantID         string
	AmountCents      money.AmountCents
	UnallocatedCents money.AmountCents
	Method           PaymentMethod
	Label            string
	ExternalRef      string
	ReceiptNumber    string
	PaymentDate      time.Time
	Allocations      []RentAllocation
	CreatedAt        time.Time
}

// PaymentRepository defines the persistence contract for the Payment aggregate root (including its RentAllocations).
type PaymentRepository interface {
	SavePayment(ctx context.Context, payment Payment) error
	FindPaymentByID(ctx context.Context, id string) (*Payment, error)
	ListPaymentsByLease(ctx context.Context, leaseID string) ([]Payment, error)
	ListPaymentsByEntity(ctx context.Context, entityID string) ([]Payment, error)
}
