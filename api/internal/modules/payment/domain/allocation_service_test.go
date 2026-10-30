// internal/modules/payment/domain/allocation_service_test.go
package domain_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	leaseapp "github.com/yannamsellem/gestion-immo/internal/modules/lease/application"
	leasedomain "github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	leasememory "github.com/yannamsellem/gestion-immo/internal/modules/lease/infrastructure/memory"
	paymentapp "github.com/yannamsellem/gestion-immo/internal/modules/payment/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
	paymentinfra "github.com/yannamsellem/gestion-immo/internal/modules/payment/infrastructure"
	paymentmemory "github.com/yannamsellem/gestion-immo/internal/modules/payment/infrastructure/memory"
	paymenthttp "github.com/yannamsellem/gestion-immo/internal/modules/payment/interfaces/http"
	paymentmcp "github.com/yannamsellem/gestion-immo/internal/modules/payment/interfaces/mcp"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
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

func TestPaymentModule_CashJournalCSVImportAndNegativePositions(t *testing.T) {
	ctx := context.Background()
	tenantRepo := leasememory.NewTenantRepo()
	leaseRepo := leasememory.NewLeaseRepo()
	rentCallRepo := leasememory.NewRentCallRepo()
	leaseUoW := leasememory.NewUnitOfWork(tenantRepo, leaseRepo, rentCallRepo)
	leaseSvc := leaseapp.NewService(tenantRepo, leaseRepo, rentCallRepo, leaseUoW)

	tenant, err := leaseSvc.CreateTenant(ctx, leaseapp.CreateTenantCmd{
		FullName: "Jean Dupont",
		Email:    "jean.dupont@example.fr",
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	entityID := id.MustNewV7()
	lotID := id.MustNewV7()
	lse, err := leaseSvc.CreateLease(ctx, leaseapp.CreateLeaseCmd{
		LegalEntityID:   entityID,
		LotID:           lotID,
		TenantID:        tenant.ID,
		LeaseType:       "RESIDENTIAL_UNFURNISHED",
		RentNetCents:    750_00,
		ChargesCents:    50_00,
		DueDayOfMonth:   5,
		GLIPolicyNumber: "GLI-2026-9988",
	})
	if err != nil {
		t.Fatalf("create lease: %v", err)
	}

	for _, period := range []string{"2026-01", "2026-02", "2026-03"} {
		if _, err := leaseSvc.GenerateMonthlyRentCalls(ctx, entityID, period); err != nil {
			t.Fatalf("generate rent calls %s: %v", period, err)
		}
	}

	paymentRepo := paymentmemory.NewPaymentRepo()
	cashJournalRepo := paymentmemory.NewCashJournalRepo()
	paymentUoW := paymentmemory.NewUnitOfWork(paymentRepo, cashJournalRepo)
	bankProvider := paymentinfra.NewDSP2Provider(false, "dsp2")
	csvImporter := paymentinfra.NewCSVImporter()
	svc := paymentapp.NewService(paymentRepo, cashJournalRepo, paymentUoW, leaseSvc, bankProvider, csvImporter)
	handler := paymenthttp.NewHandler(svc)

	// Verify DSP2 disabled by default
	if _, err := svc.SyncOpenBankingDSP2(ctx, entityID, time.Now()); !errors.Is(err, domain.ErrOpenBankingDisabled) {
		t.Fatalf("expected ErrOpenBankingDisabled, got %v", err)
	}

	// Record a cash payment for March explicitly -> posts to PCG 530 Cash Journal
	cashReceipt, err := svc.RecordCashPayment(ctx, paymentapp.RecordCashPaymentCmd{
		LeaseID:     lse.ID,
		AmountCents: 800_00,
		Label:       "Espèces loyer mars 2026",
		PaymentDate: "2026-03-10",
	})
	if err != nil {
		t.Fatalf("record cash payment: %v", err)
	}
	if cashReceipt.Method != domain.MethodCash || len(cashReceipt.Allocations) != 1 || cashReceipt.Allocations[0].PeriodMonth != "2026-03" {
		t.Fatalf("unexpected cash receipt allocations: %+v", cashReceipt)
	}

	journal, err := svc.ListCashJournalByEntity(ctx, entityID)
	if err != nil || len(journal) != 1 {
		t.Fatalf("expected 1 cash journal entry, got %d (%v)", len(journal), err)
	}
	if journal[0].AccountCode != "530" || journal[0].DebitCents != 800_00 {
		t.Fatalf("expected 530 debit 800_00, got %+v", journal[0])
	}

	// Check negative positions: Jan & Feb (1600 EUR) should still be unpaid and GLI eligible
	negPositions, err := svc.ListNegativePositions(ctx, entityID)
	if err != nil || len(negPositions) != 1 {
		t.Fatalf("expected 1 negative position, got %+v (%v)", negPositions, err)
	}
	if negPositions[0].TotalUnpaidCents != 1600_00 || !negPositions[0].GLIEligible || len(negPositions[0].UnpaidMonths) != 2 {
		t.Fatalf("unexpected negative position details: %+v", negPositions[0])
	}

	// Import CSV batch paying 600.00 EUR using decimal EUR column (no month -> FIFO allocates to 2026-01)
	csvData := "lease_id;amount;label;payment_date\n" + lse.ID + ";600,00;VIR SEPA JEAN DUPONT;2026-03-18\n"
	batchRes, err := svc.ImportPaymentsCSV(ctx, entityID, strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("import csv: %v", err)
	}
	if batchRes.ImportedCount != 1 || batchRes.TotalImportedCents != 600_00 {
		t.Fatalf("unexpected batch result: %+v", batchRes)
	}

	// Verify HTTP & MCP routes
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entities/"+entityID+"/negative-positions", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from negative-positions endpoint, got %d: %s", rec.Code, rec.Body.String())
	}
	var httpNeg []paymentapp.NegativePositionDTO
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&httpNeg); err != nil {
		t.Fatalf("decode negative positions response: %v", err)
	}
	if len(httpNeg) != 1 || httpNeg[0].TotalUnpaidCents != 1000_00 {
		t.Fatalf("expected 1000_00 remaining unpaid after CSV import, got %+v", httpNeg)
	}

	reg := mcpserver.NewRegistry(nil)
	paymentmcp.RegisterTools(reg, svc)
}
