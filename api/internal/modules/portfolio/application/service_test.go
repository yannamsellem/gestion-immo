package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/infrastructure/memory"
	portfoliohttp "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/interfaces/http"
	portfoliomcp "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/interfaces/mcp"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

func newTestPortfolioService() *application.Service {
	entityRepo := memory.NewLegalEntityRepo()
	shareholderRepo := memory.NewShareholderRepo()
	propertyRepo := memory.NewPropertyRepo()
	lotRepo := memory.NewLotRepo()
	uow := memory.NewUnitOfWork(entityRepo, shareholderRepo, propertyRepo, lotRepo)
	return application.NewService(entityRepo, shareholderRepo, propertyRepo, lotRepo, uow)
}

func TestCreateEntityAndMerchantOfGoodsPropertyWithWorkStock(t *testing.T) {
	svc := newTestPortfolioService()
	ctx := context.Background()

	ent, err := svc.CreateLegalEntity(ctx, application.CreateLegalEntityCmd{
		Name:              "SARL INVESTISSIMO",
		EntityType:        "SARL_IS",
		SIRET:             "40193236300021",
		NAFCode:           "6810Z",
		Address:           "3 RUE LAFAYETTE, 13001 MARSEILLE",
		ShareCapitalCents: money.AmountCents(196278_11),
		TotalShares:       12875,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sh, err := svc.UpsertShareholder(ctx, application.UpsertShareholderCmd{
		LegalEntityID:         ent.ID,
		FullName:              "AMSELLEM MAURICE",
		SharesCount:           6375,
		CurrentAccountCents:   money.AmountCents(404032_00),
		RemunerationPaidCents: money.AmountCents(10296_00),
	})
	if err != nil {
		t.Fatalf("unexpected shareholder error: %v", err)
	}
	if sh.DetentionBasisPoints != 4951 { // 6375 / 12875 = 49.51%
		t.Fatalf("expected 4951 bps (49.51%%), got %d", sh.DetentionBasisPoints)
	}

	prop, err := svc.CreateProperty(ctx, application.CreatePropertyCmd{
		LegalEntityID:      ent.ID,
		Label:              "APPART RUE SAINTE MARSEILLE 1er",
		PropertyType:       "APARTMENT",
		AccountingCategory: "MERCHANDISE",
		AcquisitionCents:   money.AmountCents(24284_00),
		WorkStockCents:     money.AmountCents(39835_00),
	})
	if err != nil || prop.TotalBookValueCents != money.AmountCents(64119_00) {
		t.Fatalf("expected TotalBookValueCents 64119_00, got %+v (err=%v)", prop, err)
	}
}

func TestPortfolioHierarchySummaryAndValidation(t *testing.T) {
	svc := newTestPortfolioService()
	ctx := context.Background()

	// Invalid entity type must return ErrValidation
	_, err := svc.CreateLegalEntity(ctx, application.CreateLegalEntityCmd{
		Name:       "Bad Entity",
		EntityType: "UNKNOWN_TYPE",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation for invalid entity_type, got %v", err)
	}

	ent, err := svc.CreateLegalEntity(ctx, application.CreateLegalEntityCmd{
		Name:              "SCI DE LA REPUBLIQUE",
		EntityType:        "SCI_IR",
		SIRET:             "38291022100018",
		NAFCode:           "6820B",
		Address:           "12 RUE DE LA REPUBLIQUE, 13001 MARSEILLE",
		ShareCapitalCents: money.AmountCents(1524_49),
		TotalShares:       100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Upsert shareholder and update existing shareholder
	_, err = svc.UpsertShareholder(ctx, application.UpsertShareholderCmd{
		LegalEntityID:       ent.ID,
		FullName:            "ASSOCIE A",
		SharesCount:         40,
		CurrentAccountCents: money.AmountCents(10000_00),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	shUpdated, err := svc.UpsertShareholder(ctx, application.UpsertShareholderCmd{
		LegalEntityID:       ent.ID,
		FullName:            "ASSOCIE A",
		SharesCount:         50,
		CurrentAccountCents: money.AmountCents(15000_00),
	})
	if err != nil || shUpdated.DetentionBasisPoints != 5000 {
		t.Fatalf("expected updated shareholder with 5000 bps, got %+v (err=%v)", shUpdated, err)
	}

	propImmo, err := svc.CreateProperty(ctx, application.CreatePropertyCmd{
		LegalEntityID:      ent.ID,
		Label:              "IMMEUBLE REPUBLIQUE",
		PropertyType:       "BUILDING",
		AccountingCategory: "FIXED_ASSET",
		AcquisitionCents:   money.AmountCents(350000_00),
		WorkStockCents:     money.AmountCents(25000_00),
	})
	if err != nil {
		t.Fatalf("unexpected property error: %v", err)
	}

	lot1, err := svc.CreateLot(ctx, application.CreateLotCmd{
		PropertyID:     propImmo.ID,
		Label:          "Lot 1 - RDC Commercial",
		LotType:        "COMMERCIAL_PREMISES",
		SurfaceM2:      85,
		OwnershipShare: 250,
		Floor:          "RDC",
	})
	if err != nil || lot1.SurfaceM2 != 85 {
		t.Fatalf("unexpected lot error: %+v (err=%v)", lot1, err)
	}

	summary, err := svc.GetEntityPortfolioSummary(ctx, ent.ID)
	if err != nil {
		t.Fatalf("unexpected summary error: %v", err)
	}
	if len(summary.Shareholders) != 1 || len(summary.Properties) != 1 || summary.TotalLotsCount != 1 {
		t.Fatalf("unexpected summary counts: %+v", summary)
	}
	if summary.TotalBookValueCents != money.AmountCents(375000_00) {
		t.Fatalf("expected TotalBookValueCents 375000_00, got %d", summary.TotalBookValueCents)
	}
}

func TestUpsertShareholder_RejectsExceedingTotalShares(t *testing.T) {
	svc := newTestPortfolioService()
	ctx := context.Background()

	ent, err := svc.CreateLegalEntity(ctx, application.CreateLegalEntityCmd{
		Name:        "SCI CAP TEST",
		EntityType:  "SCI_IR",
		TotalShares: 100,
	})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}

	// First shareholder gets 60 shares
	if _, err := svc.UpsertShareholder(ctx, application.UpsertShareholderCmd{
		LegalEntityID: ent.ID,
		FullName:      "ASSOCIE 1",
		SharesCount:   60,
	}); err != nil {
		t.Fatalf("upsert shareholder 1: %v", err)
	}

	// Second shareholder requesting 50 shares (60 + 50 = 110 > 100) must fail
	if _, err := svc.UpsertShareholder(ctx, application.UpsertShareholderCmd{
		LegalEntityID: ent.ID,
		FullName:      "ASSOCIE 2",
		SharesCount:   50,
	}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation when exceeding total_shares across shareholders, got %v", err)
	}

	// Updating first shareholder to 80 shares (0 + 80 <= 100) must succeed
	updated, err := svc.UpsertShareholder(ctx, application.UpsertShareholderCmd{
		LegalEntityID: ent.ID,
		FullName:      "ASSOCIE 1",
		SharesCount:   80,
	})
	if err != nil || updated.SharesCount != 80 {
		t.Fatalf("expected updating existing shareholder to 80 shares to succeed, got %+v (err=%v)", updated, err)
	}

	// Second shareholder requesting 20 shares (80 + 20 == 100) must succeed
	if _, err := svc.UpsertShareholder(ctx, application.UpsertShareholderCmd{
		LegalEntityID: ent.ID,
		FullName:      "ASSOCIE 2",
		SharesCount:   20,
	}); err != nil {
		t.Fatalf("expected 20 shares to fit within remaining total_shares, got %v", err)
	}
}

func TestPortfolioHTTPAndMCPInterfaces(t *testing.T) {
	svc := newTestPortfolioService()
	handler := portfoliohttp.NewHandler(svc)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// 1. POST /api/v1/entities
	createEntBody := `{"name":"SCI LES OLIVIERS","entity_type":"SCI_IS","siret":"51234567800011","total_shares":1000}`
	reqEnt := httptest.NewRequest(http.MethodPost, "/api/v1/entities", strings.NewReader(createEntBody))
	recEnt := httptest.NewRecorder()
	mux.ServeHTTP(recEnt, reqEnt)
	if recEnt.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", recEnt.Code, recEnt.Body.String())
	}
	var createdEnt application.LegalEntityDTO
	if err := json.Unmarshal(recEnt.Body.Bytes(), &createdEnt); err != nil {
		t.Fatalf("unmarshal entity: %v", err)
	}

	// 2. POST /api/v1/entities/{id}/shareholders
	reqSh := httptest.NewRequest(http.MethodPost, "/api/v1/entities/"+createdEnt.ID+"/shareholders", strings.NewReader(`{"full_name":"JEAN DUPONT","shares_count":250}`))
	recSh := httptest.NewRecorder()
	mux.ServeHTTP(recSh, reqSh)
	if recSh.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", recSh.Code, recSh.Body.String())
	}

	// 3. POST /api/v1/entities/{id}/properties
	reqProp := httptest.NewRequest(http.MethodPost, "/api/v1/entities/"+createdEnt.ID+"/properties", strings.NewReader(`{"label":"VILLA OLIVIERS","property_type":"HOUSE","accounting_category":"FIXED_ASSET","acquisition_cents":20000000,"work_stock_cents":5000000}`))
	recProp := httptest.NewRecorder()
	mux.ServeHTTP(recProp, reqProp)
	if recProp.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", recProp.Code, recProp.Body.String())
	}
	var createdProp application.PropertyDTO
	if err := json.Unmarshal(recProp.Body.Bytes(), &createdProp); err != nil {
		t.Fatalf("unmarshal property: %v", err)
	}

	// 4. POST /api/v1/properties/{id}/lots
	reqLot := httptest.NewRequest(http.MethodPost, "/api/v1/properties/"+createdProp.ID+"/lots", strings.NewReader(`{"label":"Lot Principal","lot_type":"RESIDENTIAL","surface_m2":120}`))
	recLot := httptest.NewRecorder()
	mux.ServeHTTP(recLot, reqLot)
	if recLot.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", recLot.Code, recLot.Body.String())
	}

	// 5. GET /api/v1/entities/{id}
	reqSum := httptest.NewRequest(http.MethodGet, "/api/v1/entities/"+createdEnt.ID, nil)
	recSum := httptest.NewRecorder()
	mux.ServeHTTP(recSum, reqSum)
	if recSum.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", recSum.Code, recSum.Body.String())
	}

	// 6. Verify all 7 MCP tools are registered and callable
	reg := mcpserver.NewRegistry(func(_ context.Context, token string) (string, error) {
		if token == "valid" {
			return "usr_1", nil
		}
		return "", errors.New("unauthorized")
	})
	portfoliomcp.RegisterTools(reg, svc)

	tools := reg.Tools()
	expectedTools := []string{
		"create_legal_entity",
		"upsert_shareholder",
		"list_entities",
		"get_entity_summary",
		"create_property",
		"create_lot",
		"list_properties",
	}
	if len(tools) != len(expectedTools) {
		t.Fatalf("expected %d MCP tools, got %d", len(expectedTools), len(tools))
	}
	for i, name := range expectedTools {
		if tools[i].Name != name {
			t.Fatalf("expected tool[%d] = %q, got %q", i, name, tools[i].Name)
		}
	}

	callBody := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_entities","arguments":{}}}`
	reqMCP := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(callBody))
	reqMCP.Header.Set("Authorization", "Bearer valid")
	recMCP := httptest.NewRecorder()
	reg.ServeHTTP(recMCP, reqMCP)
	if recMCP.Code != http.StatusOK || !strings.Contains(recMCP.Body.String(), "SCI LES OLIVIERS") {
		t.Fatalf("unexpected MCP response (%d): %s", recMCP.Code, recMCP.Body.String())
	}
}
