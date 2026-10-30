package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/yannamsellem/gestion-immo/cmd/bootstrap"
	"github.com/yannamsellem/gestion-immo/cmd/server"
	"github.com/yannamsellem/gestion-immo/cmd/token"
	"github.com/yannamsellem/gestion-immo/internal/modules/accounting"
	"github.com/yannamsellem/gestion-immo/internal/modules/lease"
	leaseapp "github.com/yannamsellem/gestion-immo/internal/modules/lease/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment"
	paymentapp "github.com/yannamsellem/gestion-immo/internal/modules/payment/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio"
	"github.com/yannamsellem/gestion-immo/internal/platform/config"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

func TestCLIApp_ExposesRequiredSubcommandsAndFlags(t *testing.T) {
	var outBuf bytes.Buffer
	app := &cli.Command{
		Name:   "gestion-immo",
		Usage:  "Gestion Immobilière & Comptable — Go Modular Monolith + Streamable HTTP MCP Server",
		Writer: &outBuf,
		Commands: []*cli.Command{
			server.Command(),
			bootstrap.Command(),
			token.Command(),
		},
	}

	for _, expectedCmd := range []string{"server", "bootstrap", "token"} {
		if app.Command(expectedCmd) == nil {
			t.Fatalf("missing top-level CLI command %q", expectedCmd)
		}
	}

	bootstrapCmd := app.Command("bootstrap")
	if bootstrapCmd.Command("super-admin") == nil || bootstrapCmd.Command("approve") == nil || bootstrapCmd.Command("reject") == nil || bootstrapCmd.Command("pending") == nil {
		t.Fatal("bootstrap command must expose 'super-admin', 'approve', 'reject', and 'pending' subcommands")
	}

	tokenCmd := app.Command("token")
	if tokenCmd.Command("create") == nil {
		t.Fatal("token command must expose 'create' subcommand")
	}

	// Verify bootstrap super-admin validates required flags before opening any DB connection
	if err := app.Run(context.Background(), []string{"gestion-immo", "bootstrap", "super-admin", "--email", "yann@example.com"}); err == nil {
		t.Fatal("expected bootstrap super-admin without required flags to fail")
	}
}

func TestServer_ExportOpenAPIFlag_WritesValidOpenAPI31SpecWithoutDatabase(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "openapi.json")
	app := &cli.Command{
		Name: "gestion-immo",
		Commands: []*cli.Command{
			server.Command(),
		},
	}

	if err := app.Run(context.Background(), []string{"gestion-immo", "server", "--export-openapi", outPath}); err != nil {
		t.Fatalf("server --export-openapi failed: %v", err)
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exported openapi.json: %v", err)
	}

	var spec struct {
		OpenAPI    string                    `json:"openapi"`
		Paths      map[string]map[string]any `json:"paths"`
		Components struct {
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("unmarshal exported openapi.json: %v", err)
	}
	if !strings.HasPrefix(spec.OpenAPI, "3.1") {
		t.Fatalf("expected OpenAPI 3.1.x, got %q", spec.OpenAPI)
	}

	requiredPaths := []string{
		"/api/v1/auth/login",
		"/api/v1/entities",
		"/api/v1/leases",
		"/api/v1/payments/cash",
		"/api/v1/documents/generate",
		"/api/v1/inspections",
		"/api/v1/entities/{id}/fec",
	}
	for _, p := range requiredPaths {
		if _, ok := spec.Paths[p]; !ok {
			t.Errorf("missing expected OpenAPI path %q", p)
		}
	}
}

func TestServerHTTPHandler_EnforcesJWTAuthOnBusinessRoutes(t *testing.T) {
	handler, err := server.BuildHTTPHandler(nil, config.Config{})
	if err != nil {
		t.Fatalf("BuildHTTPHandler failed: %v", err)
	}

	reqHealth := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recHealth := httptest.NewRecorder()
	handler.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Fatalf("expected GET /healthz to return 200 OK, got %d", recHealth.Code)
	}

	reqEntities := httptest.NewRequest(http.MethodGet, "/api/v1/entities", nil)
	recEntities := httptest.NewRecorder()
	handler.ServeHTTP(recEntities, reqEntities)
	if recEntities.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated GET /api/v1/entities to return 401 Unauthorized, got %d: %s", recEntities.Code, recEntities.Body.String())
	}
}

func TestAutomaticAccountingPosting_RentCallAndPayment(t *testing.T) {
	ctx := context.Background()
	portfolioMod := portfolio.NewModule(nil)
	leaseMod := lease.NewModule(nil)
	paymentMod := payment.NewModule(nil, leaseMod.Service(), false)
	accountingMod := accounting.NewModule(nil, r2.NewInMemoryStorage(), templating.NewInMemoryPdfEngine(), portfolioMod.Service())

	leaseMod.Service().WithAccountingRecorder(accountingMod.Service())
	paymentMod.Service().WithAccountingRecorder(accountingMod.Service())

	tenant, err := leaseMod.Service().CreateTenant(ctx, leaseapp.CreateTenantCmd{
		FullName: "DUPONT LOCATAIRE",
		Email:    "dupont@example.fr",
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	lse, err := leaseMod.Service().CreateLease(ctx, leaseapp.CreateLeaseCmd{
		LegalEntityID: "ent_main_auto",
		LotID:         "lot_1",
		TenantID:      tenant.ID,
		LeaseType:     "RESIDENTIAL_UNFURNISHED",
		RentNetCents:  money.AmountCents(900_00),
		ChargesCents:  money.AmountCents(80_00),
		DueDayOfMonth: 5,
		StartDate:     "2026-01-01",
	})
	if err != nil {
		t.Fatalf("create lease: %v", err)
	}

	calls, err := leaseMod.Service().GenerateMonthlyRentCalls(ctx, "ent_main_auto", "2026-01")
	if err != nil || len(calls) != 1 {
		t.Fatalf("generate rent calls: len=%d err=%v", len(calls), err)
	}

	if _, err := paymentMod.Service().RecordCashPayment(ctx, paymentapp.RecordCashPaymentCmd{
		LeaseID:     lse.ID,
		AmountCents: money.AmountCents(980_00),
		Label:       "Loyer janvier 2026 espèces",
		PaymentDate: "2026-01-05",
	}); err != nil {
		t.Fatalf("record cash payment: %v", err)
	}

	entries, err := accountingMod.Service().ListJournalEntries(ctx, "ent_main_auto", 2026)
	if err != nil || len(entries) != 2 {
		t.Fatalf("expected 2 automatic journal entries (VE + CA), got %d (err=%v)", len(entries), err)
	}

	fecBytes, err := accountingMod.Service().ExportEntityFEC(ctx, "ent_main_auto", 2026)
	if err != nil {
		t.Fatalf("export fec: %v", err)
	}
	fecStr := string(fecBytes)
	for _, code := range []string{"411000", "706100", "708000", "530000"} {
		if !strings.Contains(fecStr, code) {
			t.Fatalf("expected FEC export to contain %s, got:\n%s", code, fecStr)
		}
	}
}
