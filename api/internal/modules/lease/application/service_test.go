package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/lease/infrastructure/memory"
	leasehttp "github.com/yannamsellem/gestion-immo/internal/modules/lease/interfaces/http"
	leasemcp "github.com/yannamsellem/gestion-immo/internal/modules/lease/interfaces/mcp"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
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
	entityID := id.MustNewV7()
	lotID := id.MustNewV7()

	tenant, err := svc.CreateTenant(ctx, application.CreateTenantCmd{
		FullName: "BOULANGERIE DU VIEUX PORT",
		Email:    "contact@boulangerie.fr",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.CreateLease(ctx, application.CreateLeaseCmd{
		LegalEntityID:        entityID,
		LotID:                lotID,
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

	calls, err := svc.GenerateMonthlyRentCalls(ctx, entityID, "2026-01")
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
	entityID := id.MustNewV7()
	lotID := id.MustNewV7()

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
		LegalEntityID:        entityID,
		LotID:                lotID,
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

	calls, err := svc.GenerateMonthlyRentCalls(ctx, entityID, "2026-02")
	if err != nil || len(calls) != 1 {
		t.Fatalf("expected 1 residential rent call, got %d (err=%v)", len(calls), err)
	}
	if calls[0].VATCents != 0 || calls[0].TotalDueCents != money.AmountCents(925_00) || calls[0].Status != domain.StatusPending {
		t.Fatalf("unexpected residential rent call: %+v", calls[0])
	}

	_, err = svc.GenerateMonthlyRentCalls(ctx, entityID, "2026-01")
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
	entityID := id.MustNewV7()
	lotID := id.MustNewV7()

	tenant, err := svc.CreateTenant(ctx, application.CreateTenantCmd{
		FullName: "CLAIRE MARTIN",
		Email:    "claire@example.fr",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.CreateLease(ctx, application.CreateLeaseCmd{
		LegalEntityID: entityID,
		LotID:         lotID,
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
	beforeCalls, err := svc.GenerateMonthlyRentCalls(ctx, entityID, "2026-02")
	if err != nil || len(beforeCalls) != 0 {
		t.Fatalf("expected 0 rent calls before lease start_date, got %d (err=%v)", len(beforeCalls), err)
	}

	// Period within [StartDate, EndDate] (2026-04) -> 1 rent call
	inBoundsCalls, err := svc.GenerateMonthlyRentCalls(ctx, entityID, "2026-04")
	if err != nil || len(inBoundsCalls) != 1 {
		t.Fatalf("expected 1 rent call within lease bounds, got %d (err=%v)", len(inBoundsCalls), err)
	}

	// Period after EndDate (2026-06 > 2026-05) -> 0 rent calls
	afterCalls, err := svc.GenerateMonthlyRentCalls(ctx, entityID, "2026-06")
	if err != nil || len(afterCalls) != 0 {
		t.Fatalf("expected 0 rent calls after lease end_date, got %d (err=%v)", len(afterCalls), err)
	}
}

func TestLeaseModule_HTTPAndMCPIntegration(t *testing.T) {
	svc := newTestLeaseService()
	handler := leasehttp.NewHandler(svc)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	mcpUserID := id.MustNewV7()
	entityID := id.MustNewV7()
	lot99ID := id.MustNewV7()
	lot100ID := id.MustNewV7()

	reg := mcpserver.NewRegistry(func(_ context.Context, _ string) (string, error) {
		return mcpUserID, nil
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
		LegalEntityID:      entityID,
		LotID:              lot99ID,
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
	req = httptest.NewRequest(http.MethodPost, "/api/v1/entities/"+entityID+"/rent-calls/generate", bytes.NewBufferString(`{"period_month":"2026-03"}`))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/entities/%s/rent-calls/generate status = %d, body = %s", entityID, rec.Code, rec.Body.String())
	}
	var generated []application.RentCallDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &generated); err != nil || len(generated) != 1 {
		t.Fatalf("expected 1 generated rent call, got %d (err=%v)", len(generated), err)
	}
	if generated[0].TotalDueCents != money.AmountCents(2550_00) {
		t.Fatalf("expected 2550_00 TTC, got %d", generated[0].TotalDueCents)
	}

	// 4. Verify MCP tools list and execute create_tenant, create_lease, list_leases, generate_monthly_rent_calls
	tools := reg.Tools()
	if len(tools) != 4 {
		t.Fatalf("expected 4 MCP tools registered, got %d", len(tools))
	}

	callMCPTool := func(rpcID int, name string, args map[string]any) *httptest.ResponseRecorder {
		t.Helper()
		payload, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      rpcID,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      name,
				"arguments": args,
			},
		})
		r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
		r.Header.Set("Authorization", "Bearer test_token")
		w := httptest.NewRecorder()
		reg.ServeHTTP(w, r)
		return w
	}

	recMCPTenant := callMCPTool(1, "create_tenant", map[string]any{
		"full_name": "MCP TENANT SAS",
		"email":     "mcp@tenant.fr",
	})
	if recMCPTenant.Code != http.StatusOK || !bytes.Contains(recMCPTenant.Body.Bytes(), []byte("MCP TENANT SAS")) {
		t.Fatalf("create_tenant MCP failed (%d): %s", recMCPTenant.Code, recMCPTenant.Body.String())
	}

	recMCPLease := callMCPTool(2, "create_lease", map[string]any{
		"legal_entity_id": entityID,
		"lot_id":          lot100ID,
		"tenant_id":       createdTenant.ID,
		"lease_type":      "RESIDENTIAL_UNFURNISHED",
		"rent_net_cents":  950_00,
	})
	if recMCPLease.Code != http.StatusOK || !bytes.Contains(recMCPLease.Body.Bytes(), []byte(lot100ID)) {
		t.Fatalf("create_lease MCP failed (%d): %s", recMCPLease.Code, recMCPLease.Body.String())
	}

	recMCPList := callMCPTool(3, "list_leases", map[string]any{
		"legal_entity_id": entityID,
	})
	if recMCPList.Code != http.StatusOK || !bytes.Contains(recMCPList.Body.Bytes(), []byte(lot99ID)) || !bytes.Contains(recMCPList.Body.Bytes(), []byte(lot100ID)) {
		t.Fatalf("list_leases MCP failed (%d): %s", recMCPList.Code, recMCPList.Body.String())
	}

	recMCPRentCalls := callMCPTool(4, "generate_monthly_rent_calls", map[string]any{
		"legal_entity_id": entityID,
		"period_month":    "2026-04",
	})
	if recMCPRentCalls.Code != http.StatusOK || !bytes.Contains(recMCPRentCalls.Body.Bytes(), []byte("2026-04")) {
		t.Fatalf("generate_monthly_rent_calls MCP failed (%d): %s", recMCPRentCalls.Code, recMCPRentCalls.Body.String())
	}
}

func TestLease_UUIDv7IdentitiesAndBoundaryValidation(t *testing.T) {
	svc := newTestLeaseService()
	ctx := context.Background()
	entityID := id.MustNewV7()
	lotID := id.MustNewV7()

	tenant, err := svc.CreateTenant(ctx, application.CreateTenantCmd{
		FullName: "LOCATAIRE UUIDV7",
		Email:    "uuid@example.fr",
	})
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	if !id.IsValidUUID(tenant.ID) || tenant.ID[14] != '7' {
		t.Fatalf("expected UUIDv7 TenantDTO.ID, got %q", tenant.ID)
	}

	lease, err := svc.CreateLease(ctx, application.CreateLeaseCmd{
		LegalEntityID: entityID,
		LotID:         lotID,
		TenantID:      tenant.ID,
		LeaseType:     "RESIDENTIAL_UNFURNISHED",
		RentNetCents:  money.AmountCents(800_00),
	})
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	if !id.IsValidUUID(lease.ID) || lease.ID[14] != '7' {
		t.Fatalf("expected UUIDv7 LeaseDTO.ID, got %q", lease.ID)
	}

	calls, err := svc.GenerateMonthlyRentCalls(ctx, entityID, "2026-05")
	if err != nil || len(calls) != 1 {
		t.Fatalf("GenerateMonthlyRentCalls: len=%d err=%v", len(calls), err)
	}
	if !id.IsValidUUID(calls[0].ID) || calls[0].ID[14] != '7' {
		t.Fatalf("expected UUIDv7 RentCallDTO.ID, got %q", calls[0].ID)
	}

	if _, err := svc.CreateLease(ctx, application.CreateLeaseCmd{
		LegalEntityID: "not-a-uuid",
		LotID:         lotID,
		TenantID:      tenant.ID,
		LeaseType:     "RESIDENTIAL_UNFURNISHED",
		RentNetCents:  money.AmountCents(800_00),
	}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed LegalEntityID, got %v", err)
	}
	if _, err := svc.CreateLease(ctx, application.CreateLeaseCmd{
		LegalEntityID: entityID,
		LotID:         "not-a-uuid",
		TenantID:      tenant.ID,
		LeaseType:     "RESIDENTIAL_UNFURNISHED",
		RentNetCents:  money.AmountCents(800_00),
	}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed LotID, got %v", err)
	}
	if _, err := svc.CreateLease(ctx, application.CreateLeaseCmd{
		LegalEntityID: entityID,
		LotID:         lotID,
		TenantID:      "not-a-uuid",
		LeaseType:     "RESIDENTIAL_UNFURNISHED",
		RentNetCents:  money.AmountCents(800_00),
	}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed TenantID, got %v", err)
	}
	if _, err := svc.GenerateMonthlyRentCalls(ctx, "not-a-uuid", "2026-05"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed entityID in GenerateMonthlyRentCalls, got %v", err)
	}
	if _, err := svc.ListLeasesByEntity(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed entityID in ListLeasesByEntity, got %v", err)
	}
	if err := svc.ArchiveTenant(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrTenantNotFound) {
		t.Fatalf("expected ErrTenantNotFound on non-UUID in ArchiveTenant, got %v", err)
	}
}
