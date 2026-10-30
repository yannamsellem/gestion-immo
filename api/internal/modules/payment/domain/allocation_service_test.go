// internal/modules/payment/domain/allocation_service_test.go
package domain_test

import (
	"testing"
	"time"

	leasedomain "github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

func TestAllocatePayment_FIFOWhenNoMonthSpecified(t *testing.T) {
	svc := domain.NewAllocationService()
	rcJan := id.MustNewV7()
	rcFeb := id.MustNewV7()
	rcMar := id.MustNewV7()
	unpaid := []leasedomain.RentCall{
		{ID: rcJan, PeriodMonth: "2026-01", TotalDueCents: 800_00, RemainingDueCents: 800_00, Status: leasedomain.StatusUnpaid},
		{ID: rcFeb, PeriodMonth: "2026-02", TotalDueCents: 800_00, RemainingDueCents: 800_00, Status: leasedomain.StatusUnpaid},
		{ID: rcMar, PeriodMonth: "2026-03", TotalDueCents: 800_00, RemainingDueCents: 800_00, Status: leasedomain.StatusUnpaid},
	}

	allocs, updated := svc.AllocatePayment(unpaid, money.AmountCents(800_00), "VIR SEPA M DUPONT", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC))
	if len(allocs) != 1 || allocs[0].RentCallID != rcJan || allocs[0].AppliedCents != 800_00 {
		t.Fatalf("expected FIFO allocation to %s, got %+v", rcJan, allocs)
	}
	if updated[0].Status != leasedomain.StatusPaid || updated[1].Status != leasedomain.StatusUnpaid || updated[2].Status != leasedomain.StatusUnpaid {
		t.Fatalf("expected Jan PAID and Feb/Mar UNPAID, got %+v", updated)
	}
}

func TestAllocatePayment_ExplicitMonthWithOverpaymentSpillsToFIFO(t *testing.T) {
	svc := domain.NewAllocationService()
	rcJan := id.MustNewV7()
	rcFeb := id.MustNewV7()
	rcMar := id.MustNewV7()
	unpaid := []leasedomain.RentCall{
		{ID: rcJan, PeriodMonth: "2026-01", TotalDueCents: 800_00, RemainingDueCents: 800_00, Status: leasedomain.StatusUnpaid},
		{ID: rcFeb, PeriodMonth: "2026-02", TotalDueCents: 800_00, RemainingDueCents: 800_00, Status: leasedomain.StatusUnpaid},
		{ID: rcMar, PeriodMonth: "2026-03", TotalDueCents: 800_00, RemainingDueCents: 800_00, Status: leasedomain.StatusUnpaid},
	}

	// Pays 800 EUR specifying "mars" -> March is paid, Jan & Feb remain unpaid
	allocs, updated := svc.AllocatePayment(unpaid, money.AmountCents(800_00), "Virement loyer mars", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC))
	if len(allocs) != 1 || allocs[0].RentCallID != rcMar {
		t.Fatalf("expected allocation to %s, got %+v", rcMar, allocs)
	}
	if updated[2].Status != leasedomain.StatusPaid || updated[0].Status != leasedomain.StatusUnpaid || updated[1].Status != leasedomain.StatusUnpaid {
		t.Fatalf("expected Mar PAID and Jan/Feb UNPAID, got %+v", updated)
	}

	// Overpayment case: pays 1000 EUR specifying "Loyer Mars 2026" -> 800 to Mar (PAID), remaining 200 to Jan via FIFO (PARTIAL, 600 remaining)
	allocs2, updated2 := svc.AllocatePayment(unpaid, money.AmountCents(1000_00), "Virement Loyer Mars 2026", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC))
	if len(allocs2) != 2 || allocs2[0].RentCallID != rcMar || allocs2[1].RentCallID != rcJan || allocs2[1].AppliedCents != 200_00 {
		t.Fatalf("expected Mar then Jan spillover, got %+v", allocs2)
	}
	if updated2[2].Status != leasedomain.StatusPaid || updated2[0].Status != leasedomain.StatusPartial || updated2[0].RemainingDueCents != 600_00 {
		t.Fatalf("unexpected statuses after spillover: %+v", updated2)
	}
}

func TestDetectTargetMonth_FrenchAccentsAndSlashFormat(t *testing.T) {
	svc := domain.NewAllocationService()

	cases := []struct {
		label        string
		fallbackYear int
		wantMonth    string
		wantOK       bool
	}{
		{"Virement loyer février", 2026, "2026-02", true},
		{"Loyer Août 2025", 2026, "2025-08", true},
		{"Reglement decembre", 2026, "2026-12", true},
		{"Loyer 04/2026", 2025, "2026-04", true},
		{"Loyer 2026-11", 2025, "2026-11", true},
		{"VIR SEPA DOMAINE MAISON", 2026, "", false},
	}

	for _, tc := range cases {
		got, ok := svc.DetectTargetMonth(tc.label, tc.fallbackYear)
		if ok != tc.wantOK || got != tc.wantMonth {
			t.Fatalf("DetectTargetMonth(%q, %d) = (%q, %v), want (%q, %v)", tc.label, tc.fallbackYear, got, ok, tc.wantMonth, tc.wantOK)
		}
	}
}
