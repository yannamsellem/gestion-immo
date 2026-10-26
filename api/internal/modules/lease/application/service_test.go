package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/lease/infrastructure/memory"
	leasehttp "github.com/yannamsellem/gestion-immo/internal/modules/lease/interfaces/http"
	leasemcp "github.com/yannamsellem/gestion-immo/internal/modules/lease/interfaces/mcp"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

func newTestLeaseService() *application.Service {
	tenantRepo := memory.NewTenantRepo()
	leaseRepo := memory.NewLeaseRepo()
	rentCallRepo := memory.NewRentCallRepo()
	uow := memory.NewUnitOfWork(tenantRepo, leaseRepo, rentCallRepo)
	return application.NewService(tenantRepo, leaseRepo, rentCallRepo, uow)
}

func TestCreateCommercialRentCall_VATExactCentsNoFloatDrift(t *testing.T) {
	svc := newTestLeaseService()
	ctx := context.Background()

	tenant, err := svc.CreateTenant(ctx, application.CreateTenantCmd{
		FullName: "BOULANGERIE DU VIEUX PORT",
		Email:    "contact@boulangerie.fr",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.CreateLease(ctx, application.CreateLeaseCmd{
		LegalEntityID:        "ent_1",
		LotID:                "lot_comm_1",
		TenantID:             tenant.ID,
		LeaseType:            "COMMERCIAL_3_6_9",
		RentNetCents:         money.AmountCents(1912_67),
		ChargesCents:         money.AmountCents(796_50),
		SubjectToVAT:         true,
		VATRateBasisPoints:   2000, // 20.00%
		SecurityDepositCents: money.AmountCents(3825_34),
		DueDayOfMonth:        5,
		GLIPolicyNumber:      "GLI-AXA-99881",
	})
	if err != nil {
		t.Fatal(err)
	}

	calls, err := svc.GenerateMonthlyRentCalls(ctx, "ent_1", "2026-01")
	if err != nil || len(calls) != 1 {
		t.Fatalf("expected 1 rent call, got %d (err=%v)", len(calls), err)
	}
	rc := calls[0]
	// VAT 20% on Rent (1912.67) = 382.534 -> 382_53 cents; Total TTC = 1912_67 + 382_53 + 796_50 = 3091_70
	if rc.VATCents != money.AmountCents(382_53) || rc.TotalDueCents != money.AmountCents(3091_70) || rc.RemainingDueCents != money.AmountCents(3091_70) {
		t.Fatalf("unexpected rent call amounts: %+v", rc)
	}
}

func TestResidentialRentCall_NoVATAndArchiveTenant(t *testing.T) {
	svc := newTestLeaseService()
	ctx := context.Background()

	t1, err := svc.CreateTenant(ctx, application.CreateTenantCmd{
		FullName: "M. JEAN DUPONT",
		Email:    "jean.dupont@example.fr",
	})
	if err != nil {
		t.Fatal(err)
	}
	t2, err := svc.CreateTenant(ctx, application.CreateTenantCmd{
		FullName: "MME MARIE CURIE",
		Email:    "marie.curie@example.fr",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.ArchiveTenant(ctx, t2.ID); err != nil {
		t.Fatalf("archive tenant failed: %v", err)
	}

	presentOnly, err := svc.ListTenants(ctx, false)
	if err != nil || len(presentOnly) != 1 || presentOnly[0].ID != t1.ID {
		t.Fatalf("expected 1 present tenant, got %+v (err=%v)", presentOnly, err)
	}

	allTenants, err := svc.ListTenants(ctx, true)
	if err != nil || len(allTenants) != 2 {
		t.Fatalf("expected 2 total tenants including archived, got %d (err=%v)", len(allTenants), err)
	}

	lse, err := svc.CreateLease(ctx, application.CreateLeaseCmd{
		LegalEntityID:        "ent_sci_1",
		LotID:                "lot_res_1",
		TenantID:             t1.ID,
		LeaseType:            "RESIDENTIAL_UNFURNISHED",
		RentNetCents:         money.AmountCents(850_00),
		ChargesCents:         money.AmountCents(75_00),
		SubjectToVAT:         false,
		SecurityDepositCents: money.AmountCents(850_00),
		DueDayOfMonth:        1,
		GLIPolicyNumber:      "GLI-ALLIANZ-1001",
		GuarantorName:        "PIERRE DUPONT",
	})
	if err != nil {
		t.Fatal(err)
	}

	calls, err := svc.GenerateMonthlyRentCalls(ctx, "ent_sci_1", "2026-02")
	if err != nil || len(calls) != 1 {
		t.Fatalf("expected 1 residential rent call, got %d (err=%v)", len(calls), err)
	}
	if calls[0].VATCents != 0 || calls[0].TotalDueCents != money.AmountCents(925_00) || calls[0].Status != domain.StatusPending {
		t.Fatalf("unexpected residential rent call: %+v", calls[0])
	}

	_, err = svc.GenerateMonthlyRentCalls(ctx, "ent_sci_1", "2026-01")
	if err != nil {
		t.Fatal(err)
	}

	unpaid, err := svc.ListUnpaidRentCallsByLease(ctx, lse.ID)
	if err != nil || len(unpaid) != 2 {
		t.Fatalf("expected 2 unpaid rent calls, got %d (err=%v)", len(unpaid), err)
	}
	if unpaid[0].PeriodMonth != "2026-01" || unpaid[1].PeriodMonth != "2026-02" {
		t.Fatalf("expected oldest unpaid rent call first (2026-01 then 2026-02), got %s then %s", unpaid[0].PeriodMonth, unpaid[1].PeriodMonth)
	}
}

func TestGenerateMonthlyRentCalls_RespectsLeaseStartAndEndDates(t *testing.T) {
	svc := newTestLeaseService()
	ctx := context.Background()

	tenant, err := svc.CreateTenant(ctx, application.CreateTenantCmd{
		FullName: "CLAIRE MARTIN",
		Email:    "claire@example.fr",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.CreateLease(ctx, application.CreateLeaseCmd{
		LegalEntityID: "ent_bounds",
		LotID:         "lot_bounds",
		TenantID:      tenant.ID,
		LeaseType:     "RESIDENTIAL_UNFURNISHED",
		RentNetCents:  money.AmountCents(700_00),
		ChargesCents:  money.AmountCents(50_00),
		StartDate:     "2026-03-01",
		EndDate:       "2026-05-31",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Period before StartDate (2026-02 < 2026-03) -> 0 rent calls
	beforeCalls, err := svc.GenerateMonthlyRentCalls(ctx, "ent_bounds", "2026-02")
	if err != nil || len(beforeCalls) != 0 {
		t.Fatalf("expected 0 rent calls before lease start_date, got %d (err=%v)", len(beforeCalls), err)
	}

	// Period within [StartDate, EndDate] (2026-04) -> 1 rent call
	inBoundsCalls, err := svc.GenerateMonthlyRentCalls(ctx, "ent_bounds", "2026-04")
	if err != nil || len(inBoundsCalls) != 1 {
		t.Fatalf("expected 1 rent call within lease bounds, got %d (err=%v)", len(inBoundsCalls), err)
	}

	// Period after EndDate (2026-06 > 2026-05) -> 0 rent calls
	afterCalls, err := svc.GenerateMonthlyRentCalls(ctx, "ent_bounds", "2026-06")
	if err != nil || len(afterCalls) != 0 {
		t.Fatalf("expected 0 rent calls after lease end_date, got %d (err=%v)", len(afterCalls), err)
	}
}

func TestLeaseModule_HTTPAndMCPIntegration(t *testing.T) {
	svc := newTestLeaseService()
	handler := leasehttp.NewHandler(svc)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	reg := mcpserver.NewRegistry(func(_ context.Context, token string) (string, error) {
		return "user_1", nil
	})
	leasemcp.RegisterTools(reg, svc)

	// 1. Create tenant via HTTP POST /api/v1/tenants
	tenantBody := `{"full_name":"SARL OPTIQUE SOLEIL","email":"contact@optique.fr"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", bytes.NewBufferString(tenantBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/tenants status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var createdTenant application.TenantDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &createdTenant); err != nil {
		t.Fatal(err)
	}

	// 2. Create lease via HTTP POST /api/v1/leases
	leasePayload, _ := json.Marshal(application.CreateLeaseCmd{
		LegalEntityID:      "ent_99",
		LotID:              "lot_99",
		TenantID:           createdTenant.ID,
		LeaseType:          "COMMERCIAL_3_6_9",
		RentNetCents:       money.AmountCents(2000_00),
		ChargesCents:       money.AmountCents(150_00),
		SubjectToVAT:       true,
		VATRateBasisPoints: 2000,
		DueDayOfMonth:      5,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/leases", bytes.NewReader(leasePayload))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/leases status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// 3. Generate rent calls via HTTP POST /api/v1/entities/{id}/rent-calls/generate
	req = httptest.NewRequest(http.MethodPost, "/api/v1/entities/ent_99/rent-calls/generate", bytes.NewBufferString(`{"period_month":"2026-03"}`))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/entities/ent_99/rent-calls/generate status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var generated []application.RentCallDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &generated); err != nil || len(generated) != 1 {
		t.Fatalf("expected 1 generated rent call, got %d (err=%v)", len(generated), err)
	}
	if generated[0].TotalDueCents != money.AmountCents(2550_00) {
		t.Fatalf("expected 2550_00 TTC, got %d", generated[0].TotalDueCents)
	}

	// 4. Verify MCP tools list includes create_tenant, create_lease, list_leases
	tools := reg.Tools()
	if len(tools) != 3 {
		t.Fatalf("expected 3 MCP tools registered, got %d", len(tools))
	}
}
