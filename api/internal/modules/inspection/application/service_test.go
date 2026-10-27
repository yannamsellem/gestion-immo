package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/infrastructure/memory"
	insphttp "github.com/yannamsellem/gestion-immo/internal/modules/inspection/interfaces/http"
	inspmcp "github.com/yannamsellem/gestion-immo/internal/modules/inspection/interfaces/mcp"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

func TestGenerateExitInspection_WithoutStructuredEntryInspectionSucceeds(t *testing.T) {
	templates := memory.NewInspectionTemplateRepo()
	inspections := memory.NewInspectionRepo()
	incidents := memory.NewIncidentRepo()
	uow := memory.NewUnitOfWork(templates, inspections, incidents)
	svc := application.NewService(templates, inspections, incidents, uow, r2.NewInMemoryStorage(), templating.NewInMemoryPdfEngine())
	ctx := context.Background()

	tpl, err := svc.CreateTemplate(ctx, application.CreateInspectionTemplateCmd{
		Name:  "Appartement T2 Standard",
		Rooms: []string{"Séjour", "Cuisine", "Chambre", "Salle de bain"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Exit inspection created for a legacy lease that has NO structured entry inspection (EntryInspectionID == nil)
	exitInsp, err := svc.CreateInspection(ctx, application.CreateInspectionCmd{
		LeaseID:           "lease_legacy_1",
		TemplateID:        tpl.ID,
		Type:              "MOVE_OUT",
		EntryInspectionID: nil,
	})
	if err != nil {
		t.Fatalf("unexpected error creating exit inspection without entry: %v", err)
	}

	updated, err := svc.UpsertInspectionItems(ctx, exitInsp.ID, []application.InspectionItemInput{
		{RoomName: "Séjour", ElementName: "Parquet", Condition: "POOR", Observation: "Rayures profondes près de la baie vitrée", PhotoR2Keys: []string{"edl/lease_legacy_1/sejour_1.jpg"}},
	})
	if err != nil || len(updated.Items) != 1 {
		t.Fatalf("expected 1 inspection item, got %+v (err=%v)", updated, err)
	}

	report, err := svc.GenerateInspectionReport(ctx, exitInsp.ID)
	if err != nil || report.HasEntryComparison {
		t.Fatalf("expected standalone report with HasEntryComparison=false, got %+v (err=%v)", report, err)
	}
}

func TestGenerateExitInspection_WithStructuredEntryComparisonAndIncidents(t *testing.T) {
	ctx := context.Background()
	r2Store := r2.NewInMemoryStorage()
	pdfEngine := templating.NewInMemoryPdfEngine()

	templates := memory.NewInspectionTemplateRepo()
	inspections := memory.NewInspectionRepo()
	incidents := memory.NewIncidentRepo()
	uow := memory.NewUnitOfWork(templates, inspections, incidents)
	svc := application.NewService(templates, inspections, incidents, uow, r2Store, pdfEngine)
	handler := insphttp.NewHandler(svc)

	tpl, err := svc.CreateTemplate(ctx, application.CreateInspectionTemplateCmd{
		Name:  "Appartement T3 Haussmannien",
		Rooms: []string{"Séjour", "Cuisine", "Chambre 1", "Chambre 2", "Salle de bain"},
	})
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	entryInsp, err := svc.CreateInspection(ctx, application.CreateInspectionCmd{
		LeaseID:        "lease_t3_2025",
		TemplateID:     tpl.ID,
		Type:           string(domain.InspectionTypeMoveIn),
		InspectionDate: "2025-01-15",
	})
	if err != nil {
		t.Fatalf("create entry inspection: %v", err)
	}

	if _, err := svc.UpsertInspectionItems(ctx, entryInsp.ID, []application.InspectionItemInput{
		{RoomName: "Séjour", ElementName: "Parquet", Condition: "NEW", Observation: "Parquet vitrifié neuf"},
		{RoomName: "Cuisine", ElementName: "Plan de travail", Condition: "GOOD", Observation: "Aucune trace"},
		{RoomName: "Salle de bain", ElementName: "Robinetterie", Condition: "FAIR_WEAR", Observation: "Léger calcaire"},
	}); err != nil {
		t.Fatalf("upsert entry items: %v", err)
	}

	entryID := entryInsp.ID

	// Reject entry_inspection_id on MOVE_IN inspection
	if _, err := svc.CreateInspection(ctx, application.CreateInspectionCmd{
		LeaseID:           "lease_t3_2025",
		TemplateID:        tpl.ID,
		Type:              string(domain.InspectionTypeMoveIn),
		EntryInspectionID: &entryID,
	}); err == nil {
		t.Fatalf("expected error when setting entry_inspection_id on MOVE_IN inspection")
	}

	// Reject entry_inspection_id belonging to a different lease
	if _, err := svc.CreateInspection(ctx, application.CreateInspectionCmd{
		LeaseID:           "lease_other_999",
		TemplateID:        tpl.ID,
		Type:              string(domain.InspectionTypeMoveOut),
		EntryInspectionID: &entryID,
	}); err == nil {
		t.Fatalf("expected error when entry_inspection_id belongs to a different lease")
	}

	exitInsp, err := svc.CreateInspection(ctx, application.CreateInspectionCmd{
		LeaseID:           "lease_t3_2025",
		TemplateID:        tpl.ID,
		Type:              string(domain.InspectionTypeMoveOut),
		EntryInspectionID: &entryID,
		InspectionDate:    "2026-10-20",
	})
	if err != nil {
		t.Fatalf("create exit inspection: %v", err)
	}

	if _, err := svc.UpsertInspectionItems(ctx, exitInsp.ID, []application.InspectionItemInput{
		{RoomName: "Séjour", ElementName: "Parquet", Condition: "POOR", Observation: "Brûlure et lames fendues", PhotoR2Keys: []string{"edl/lease_t3_2025/parquet_1.jpg", "edl/lease_t3_2025/parquet_2.jpg"}},
		{RoomName: "Cuisine", ElementName: "Plan de travail", Condition: "GOOD", Observation: "RAS"},
		{RoomName: "Salle de bain", ElementName: "Robinetterie", Condition: "OUT_OF_SERVICE", Observation: "Mitigeur cassé", PhotoR2Keys: []string{"edl/lease_t3_2025/robinetterie_1.jpg"}},
	}); err != nil {
		t.Fatalf("upsert exit items: %v", err)
	}

	report, err := svc.GenerateInspectionReport(ctx, exitInsp.ID)
	if err != nil {
		t.Fatalf("generate exit report: %v", err)
	}
	if !report.HasEntryComparison {
		t.Fatalf("expected HasEntryComparison=true, got false")
	}
	if len(report.Comparisons) != 3 {
		t.Fatalf("expected 3 element comparisons, got %d", len(report.Comparisons))
	}
	if len(report.Degradations) != 2 {
		t.Fatalf("expected 2 degradations (Parquet, Robinetterie), got %+v", report.Degradations)
	}

	rc, err := r2Store.Download(ctx, report.PDFR2Key)
	if err != nil {
		t.Fatalf("download generated PDF from R2: %v", err)
	}
	pdfBytes, _ := io.ReadAll(rc)
	_ = rc.Close()
	if !strings.HasPrefix(string(pdfBytes), "%PDF-1.4") {
		t.Fatalf("expected valid PDF header in R2, got: %s", string(pdfBytes))
	}
	for _, expected := range []string{"Photos", "edl/lease_t3_2025/parquet_1.jpg, edl/lease_t3_2025/parquet_2.jpg", "edl/lease_t3_2025/robinetterie_1.jpg"} {
		if !strings.Contains(string(pdfBytes), expected) {
			t.Fatalf("expected rendered inspection report HTML/PDF to contain %q, got: %s", expected, string(pdfBytes))
		}
	}

	// Verify HTTP endpoints & MCP tools for incidents and inspections
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	itemsPayload, _ := json.Marshal([]application.InspectionItemInput{
		{RoomName: "Chambre 1", ElementName: "Fenêtre", Condition: "GOOD", Observation: "Double vitrage OK"},
	})
	reqUpsertItems := httptest.NewRequest(http.MethodPut, "/api/v1/inspections/"+exitInsp.ID+"/items", bytes.NewReader(itemsPayload))
	recUpsertItems := httptest.NewRecorder()
	mux.ServeHTTP(recUpsertItems, reqUpsertItems)
	if recUpsertItems.Code != http.StatusOK {
		t.Fatalf("expected 200 from PUT /api/v1/inspections/{id}/items, got %d: %s", recUpsertItems.Code, recUpsertItems.Body.String())
	}

	reqGenPDF := httptest.NewRequest(http.MethodPost, "/api/v1/inspections/"+exitInsp.ID+"/pdf", nil)
	recGenPDF := httptest.NewRecorder()
	mux.ServeHTTP(recGenPDF, reqGenPDF)
	if recGenPDF.Code != http.StatusCreated {
		t.Fatalf("expected 201 from POST /api/v1/inspections/{id}/pdf, got %d: %s", recGenPDF.Code, recGenPDF.Body.String())
	}

	incPayload, _ := json.Marshal(application.CreateIncidentCmd{
		PropertyID:         "prop_1",
		LotID:              "lot_1",
		LeaseID:            "lease_t3_2025",
		Title:              "Dégât des eaux salle de bain",
		EstimatedCostCents: 1250_00,
		Observations:       []string{"Fuite joint colonne montante"},
		PhotoR2Keys:        []string{"incidents/prop_1/fuite_1.jpg"},
	})
	reqCreateInc := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", bytes.NewReader(incPayload))
	recCreateInc := httptest.NewRecorder()
	mux.ServeHTTP(recCreateInc, reqCreateInc)
	if recCreateInc.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating incident, got %d: %s", recCreateInc.Code, recCreateInc.Body.String())
	}

	var createdInc application.IncidentDTO
	if err := json.Unmarshal(recCreateInc.Body.Bytes(), &createdInc); err != nil {
		t.Fatal(err)
	}
	if createdInc.Status != domain.IncidentStatusOpen || createdInc.EstimatedCostCents != 1250_00 {
		t.Fatalf("unexpected incident DTO: %+v", createdInc)
	}

	// Update incident via MCP tool update_incident
	reg := mcpserver.NewRegistry(func(_ context.Context, _ string) (string, error) {
		return "usr_admin", nil
	})
	inspmcp.RegisterTools(reg, svc)

	claimRef := "SIN-AXA-2026-7741"
	newCost := money.AmountCents(1480_00)
	mcpBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "update_incident",
			"arguments": map[string]any{
				"incident_id":            createdInc.ID,
				"status":                 "INSURANCE_CLAIM",
				"insurance_claim_number": claimRef,
				"estimated_cost_cents":   newCost,
				"add_observation":        "Expert mandaté pour le 28/10/2026",
			},
		},
	})
	reqMCP := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(mcpBody))
	reqMCP.Header.Set("Authorization", "Bearer test_token")
	recMCP := httptest.NewRecorder()
	reg.ServeHTTP(recMCP, reqMCP)
	if recMCP.Code != http.StatusOK || !strings.Contains(recMCP.Body.String(), "SIN-AXA-2026-7741") {
		t.Fatalf("unexpected MCP update_incident response (%d): %s", recMCP.Code, recMCP.Body.String())
	}
}
