package application_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/modules/document/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/document/domain"
	docmemory "github.com/yannamsellem/gestion-immo/internal/modules/document/infrastructure/memory"
	dochttp "github.com/yannamsellem/gestion-immo/internal/modules/document/interfaces/http"
	docmcp "github.com/yannamsellem/gestion-immo/internal/modules/document/interfaces/mcp"
	leaseapp "github.com/yannamsellem/gestion-immo/internal/modules/lease/application"
	leasememory "github.com/yannamsellem/gestion-immo/internal/modules/lease/infrastructure/memory"
	paymentapp "github.com/yannamsellem/gestion-immo/internal/modules/payment/application"
	paymentinfra "github.com/yannamsellem/gestion-immo/internal/modules/payment/infrastructure"
	paymentmemory "github.com/yannamsellem/gestion-immo/internal/modules/payment/infrastructure/memory"
	portfolioapp "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/application"
	portfoliomemory "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/infrastructure/memory"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

type spyObjectStorage struct {
	r2.ObjectStorage
	uploads int
}

func (s *spyObjectStorage) Upload(ctx context.Context, key string, contentType string, body io.Reader) error {
	s.uploads++
	return s.ObjectStorage.Upload(ctx, key, contentType, body)
}

func TestAssembleGLIDossier_PopulatesDocxRendersLedgerAndBundlesR2Attachments(t *testing.T) {
	ctx := context.Background()

	entityRepo := portfoliomemory.NewLegalEntityRepo()
	shareholderRepo := portfoliomemory.NewShareholderRepo()
	propertyRepo := portfoliomemory.NewPropertyRepo()
	lotRepo := portfoliomemory.NewLotRepo()
	portfolioUoW := portfoliomemory.NewUnitOfWork(entityRepo, shareholderRepo, propertyRepo, lotRepo)
	portfolioSvc := portfolioapp.NewService(entityRepo, shareholderRepo, propertyRepo, lotRepo, portfolioUoW)

	tenantRepo := leasememory.NewTenantRepo()
	leaseRepo := leasememory.NewLeaseRepo()
	rentCallRepo := leasememory.NewRentCallRepo()
	leaseUoW := leasememory.NewUnitOfWork(tenantRepo, leaseRepo, rentCallRepo)
	leaseSvc := leaseapp.NewService(tenantRepo, leaseRepo, rentCallRepo, leaseUoW)

	paymentRepo := paymentmemory.NewPaymentRepo()
	cashJournalRepo := paymentmemory.NewCashJournalRepo()
	paymentUoW := paymentmemory.NewUnitOfWork(paymentRepo, cashJournalRepo)
	bankProvider := paymentinfra.NewDSP2Provider(false, "dsp2")
	csvImporter := paymentinfra.NewCSVImporter()
	paymentSvc := paymentapp.NewService(paymentRepo, cashJournalRepo, paymentUoW, leaseSvc, bankProvider, csvImporter)

	r2Store := r2.NewInMemoryStorage()
	pdfEngine := templating.NewInMemoryPdfEngine()
	docxEngine := templating.NewDocxEngine()

	docTemplateRepo := docmemory.NewDocumentTemplateRepo()
	storedFileRepo := docmemory.NewStoredFileRepo()
	gliDossierRepo := docmemory.NewGLIDossierRepo()
	docUoW := docmemory.NewUnitOfWork(docTemplateRepo, storedFileRepo, gliDossierRepo)
	docSvc := application.NewService(
		docTemplateRepo,
		storedFileRepo,
		gliDossierRepo,
		docUoW,
		r2Store,
		docxEngine,
		pdfEngine,
		portfolioSvc,
		leaseSvc,
		paymentSvc,
	)
	docHandler := dochttp.NewHandler(docSvc)

	// 1. Seed entity, property, lot, tenant, and lease with GLI policy
	entity, err := portfolioSvc.CreateLegalEntity(ctx, portfolioapp.CreateLegalEntityCmd{
		Name:              "SARL INVESTISSIMO",
		EntityType:        "SARL_IS",
		SIRET:             "44123456700019",
		Address:           "12 Rue Sainte, 13001 Marseille",
		ShareCapitalCents: 10000_00,
		TotalShares:       100,
	})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}

	prop, err := portfolioSvc.CreateProperty(ctx, portfolioapp.CreatePropertyCmd{
		LegalEntityID:      entity.ID,
		Label:              "Domaine des Caillols",
		PropertyType:       "BUILDING",
		AccountingCategory: "MERCHANDISE",
		Address:            "45 Avenue des Caillols, 13012 Marseille",
		AcquisitionCents:   250000_00,
	})
	if err != nil {
		t.Fatalf("create property: %v", err)
	}

	lot, err := portfolioSvc.CreateLot(ctx, portfolioapp.CreateLotCmd{
		PropertyID: prop.ID,
		Label:      "Appartement T3 - Lot A1",
		LotType:    "APARTMENT",
		SurfaceM2:  68,
	})
	if err != nil {
		t.Fatalf("create lot: %v", err)
	}

	tenant, err := leaseSvc.CreateTenant(ctx, leaseapp.CreateTenantCmd{
		FullName: "Jean Dupont",
		Email:    "jean.dupont@example.fr",
		Phone:    "0601020304",
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	lease, err := leaseSvc.CreateLease(ctx, leaseapp.CreateLeaseCmd{
		LegalEntityID:   entity.ID,
		LotID:           lot.ID,
		TenantID:        tenant.ID,
		LeaseType:       "RESIDENTIAL_UNFURNISHED",
		RentNetCents:    750_00,
		ChargesCents:    50_00,
		GLIPolicyNumber: "GLI-AXA-2026-9988",
		StartDate:       "2026-01-01",
	})
	if err != nil {
		t.Fatalf("create lease: %v", err)
	}

	// 2. Generate Jan, Feb, Mar 2026 rent calls (800 EUR each) and pay March explicitly
	for _, month := range []string{"2026-01", "2026-02", "2026-03"} {
		if _, err := leaseSvc.GenerateMonthlyRentCalls(ctx, entity.ID, month); err != nil {
			t.Fatalf("generate rent calls %s: %v", month, err)
		}
	}
	if _, err := paymentSvc.RecordBankPayment(ctx, paymentapp.RecordBankPaymentCmd{
		LeaseID:     lease.ID,
		AmountCents: 800_00,
		Method:      "BANK_TRANSFER",
		Label:       "Virement loyer mars 2026",
		PaymentDate: "2026-03-05",
	}); err != nil {
		t.Fatalf("record payment: %v", err)
	}

	// 3. Attach signed lease & entry inspection to R2
	if _, err := docSvc.AttachFileToR2(ctx, application.AttachFileCmd{
		LegalEntityID: entity.ID,
		LeaseID:       lease.ID,
		Category:      "LEASE_SIGNED",
		FileName:      "bail_signe_dupont.pdf",
		ContentType:   "application/pdf",
		Content:       []byte("%PDF-1.4 BAIL SIGNE JEAN DUPONT"),
	}); err != nil {
		t.Fatalf("attach lease pdf: %v", err)
	}

	// 4. Register a custom GLI .docx template and assemble the GLI dossier
	tplDocx := templating.BuildMinimalTestDocx(t,
		"DECLARATION GLI | Bailleur: {{societe.nom}} | Locataire: {{locataire.nom}} | Police: {{bail.gli_police}} | Total: {{impaye.total}}",
	)
	tpl, err := docSvc.RegisterTemplate(ctx, application.RegisterTemplateCmd{
		Name:         "Modèle Déclaration GLI AXA",
		TemplateType: "GLI_DECLARATION",
		Format:       "DOCX",
		DocxBytes:    tplDocx,
	})
	if err != nil {
		t.Fatalf("register template: %v", err)
	}

	dossier, err := docSvc.AssembleGLIDossier(ctx, lease.ID, tpl.ID)
	if err != nil {
		t.Fatalf("AssembleGLIDossier failed: %v", err)
	}

	if dossier.TotalUnpaidCents != 1600_00 {
		t.Fatalf("expected total unpaid 1600_00 cents, got %+v", dossier)
	}
	if len(dossier.UnpaidMonths) != 2 || dossier.UnpaidMonths[0] != "2026-01" || dossier.UnpaidMonths[1] != "2026-02" {
		t.Fatalf("expected unpaid months [2026-01 2026-02], got %v", dossier.UnpaidMonths)
	}

	// 5. Download ZIP from R2 and inspect contents
	rc, err := r2Store.Download(ctx, dossier.ArchiveR2Key)
	if err != nil {
		t.Fatalf("download GLI zip from R2: %v", err)
	}
	defer rc.Close()
	zipBytes, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read GLI zip: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("open GLI zip: %v", err)
	}

	entries := make(map[string][]byte)
	for _, f := range zr.File {
		frc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(frc)
		_ = frc.Close()
		entries[f.Name] = b
	}

	for _, expectedFile := range []string{
		"01_declaration_gli.docx",
		"02_extrait_compte_locataire.pdf",
		"03_mise_en_demeure.pdf",
		"pieces_jointes/bail_signe_dupont.pdf",
	} {
		if _, ok := entries[expectedFile]; !ok {
			t.Fatalf("missing expected file %q in GLI archive (got %v)", expectedFile, dossier.IncludedFiles)
		}
	}

	docxText := templating.ExtractTextFromDocx(t, bytes.NewReader(entries["01_declaration_gli.docx"]))
	for _, snippet := range []string{"SARL INVESTISSIMO", "Jean Dupont", "GLI-AXA-2026-9988", "1 600,00 €", "2026-01", "2026-02"} {
		if !strings.Contains(docxText, snippet) {
			t.Fatalf("expected %q in rendered 01_declaration_gli.docx, got: %s", snippet, docxText)
		}
	}

	// 6. Verify HTTP endpoint and MCP tools
	mux := http.NewServeMux()
	docHandler.RegisterRoutes(mux)

	reqListTpl := httptest.NewRequest(http.MethodGet, "/api/v1/document-templates", nil)
	recListTpl := httptest.NewRecorder()
	mux.ServeHTTP(recListTpl, reqListTpl)
	if recListTpl.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 from GET /api/v1/document-templates, got %d: %s", recListTpl.Code, recListTpl.Body.String())
	}
	var listedTemplates []application.DocumentTemplateDTO
	if err := json.Unmarshal(recListTpl.Body.Bytes(), &listedTemplates); err != nil || len(listedTemplates) == 0 {
		t.Fatalf("expected non-empty template list, got %+v (err=%v)", listedTemplates, err)
	}

	genBody, _ := json.Marshal(application.GenerateDocCmd{
		TemplateID:     tpl.ID,
		LegalEntityID:  entity.ID,
		LeaseID:        lease.ID,
		OutputFileName: "declaration_gli_custom.docx",
	})
	reqGen := httptest.NewRequest(http.MethodPost, "/api/v1/documents/generate", bytes.NewReader(genBody))
	recGen := httptest.NewRecorder()
	mux.ServeHTTP(recGen, reqGen)
	if recGen.Code != http.StatusCreated {
		t.Fatalf("expected HTTP 201 from POST /api/v1/documents/generate, got %d: %s", recGen.Code, recGen.Body.String())
	}

	reqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/leases/"+lease.ID+"/gli-dossier", nil)
	recHTTP := httptest.NewRecorder()
	mux.ServeHTTP(recHTTP, reqHTTP)
	if recHTTP.Code != http.StatusCreated {
		t.Fatalf("expected HTTP 201 from /gli-dossier, got %d: %s", recHTTP.Code, recHTTP.Body.String())
	}

	adminUserID := id.MustNewV7()
	reg := mcpserver.NewRegistry(func(_ context.Context, token string) (string, error) {
		return adminUserID, nil
	})
	docmcp.RegisterTools(reg, docSvc)

	mcpBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "generate_gli_dossier",
			"arguments": map[string]any{
				"lease_id":    lease.ID,
				"template_id": tpl.ID,
			},
		},
	})
	reqMCP := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(mcpBody))
	reqMCP.Header.Set("Authorization", "Bearer test_token")
	recMCP := httptest.NewRecorder()
	reg.ServeHTTP(recMCP, reqMCP)
	if recMCP.Code != http.StatusOK || !strings.Contains(recMCP.Body.String(), "160000") {
		t.Fatalf("unexpected MCP generate_gli_dossier response (%d): %s", recMCP.Code, recMCP.Body.String())
	}
}

func TestDocumentService_SecurityAndDuplicateGLIAttachments(t *testing.T) {
	ctx := context.Background()

	entityRepo := portfoliomemory.NewLegalEntityRepo()
	shareholderRepo := portfoliomemory.NewShareholderRepo()
	propertyRepo := portfoliomemory.NewPropertyRepo()
	lotRepo := portfoliomemory.NewLotRepo()
	portfolioUoW := portfoliomemory.NewUnitOfWork(entityRepo, shareholderRepo, propertyRepo, lotRepo)
	portfolioSvc := portfolioapp.NewService(entityRepo, shareholderRepo, propertyRepo, lotRepo, portfolioUoW)

	tenantRepo := leasememory.NewTenantRepo()
	leaseRepo := leasememory.NewLeaseRepo()
	rentCallRepo := leasememory.NewRentCallRepo()
	leaseUoW := leasememory.NewUnitOfWork(tenantRepo, leaseRepo, rentCallRepo)
	leaseSvc := leaseapp.NewService(tenantRepo, leaseRepo, rentCallRepo, leaseUoW)

	r2Store := r2.NewInMemoryStorage()
	pdfEngine := templating.NewInMemoryPdfEngine()
	docxEngine := templating.NewDocxEngine()

	docTemplateRepo := docmemory.NewDocumentTemplateRepo()
	storedFileRepo := docmemory.NewStoredFileRepo()
	gliDossierRepo := docmemory.NewGLIDossierRepo()
	docUoW := docmemory.NewUnitOfWork(docTemplateRepo, storedFileRepo, gliDossierRepo)
	docSvc := application.NewService(
		docTemplateRepo,
		storedFileRepo,
		gliDossierRepo,
		docUoW,
		r2Store,
		docxEngine,
		pdfEngine,
		portfolioSvc,
		leaseSvc,
		nil,
	)

	// 1. RegisterTemplate rejects HTML containing <script> or <iframe> with domain.ErrValidation
	for _, unsafeHTML := range []string{
		"<html><body><script>alert(1)</script></body></html>",
		"<html><body><IFRAME src=\"http://postgres:5432\"></IFRAME></body></html>",
	} {
		_, err := docSvc.RegisterTemplate(ctx, application.RegisterTemplateCmd{
			Name:         "Unsafe Template",
			TemplateType: "CUSTOM",
			Format:       "HTML",
			HTMLBody:     unsafeHTML,
		})
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("expected domain.ErrValidation for unsafe HTML %q, got %v", unsafeHTML, err)
		}
	}

	// 2. AttachFileToR2 rejects lease_id containing "../" or "a/b" before uploading to R2
	for _, invalidLeaseID := range []string{"../", "../secret", "a/b"} {
		_, err := docSvc.AttachFileToR2(ctx, application.AttachFileCmd{
			LeaseID:     invalidLeaseID,
			Category:    "ATTACHMENT",
			FileName:    "test.pdf",
			ContentType: "application/pdf",
			Content:     []byte("%PDF-1.4 test"),
		})
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("expected domain.ErrValidation for invalid lease_id %q, got %v", invalidLeaseID, err)
		}
	}

	// 3. AssembleGLIDossier deduplicates two lease attachments both named "quittance.pdf"
	entity, err := portfolioSvc.CreateLegalEntity(ctx, portfolioapp.CreateLegalEntityCmd{
		Name:              "SCI AZUR",
		EntityType:        "SCI_IR",
		SIRET:             "55123456700011",
		ShareCapitalCents: 1000_00,
		TotalShares:       100,
	})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}
	prop, err := portfolioSvc.CreateProperty(ctx, portfolioapp.CreatePropertyCmd{
		LegalEntityID:      entity.ID,
		Label:              "Immeuble Azur",
		PropertyType:       "BUILDING",
		AccountingCategory: "FIXED_ASSET",
		AcquisitionCents:   100000_00,
	})
	if err != nil {
		t.Fatalf("create property: %v", err)
	}
	lot, err := portfolioSvc.CreateLot(ctx, portfolioapp.CreateLotCmd{
		PropertyID: prop.ID,
		Label:      "Lot 1",
		LotType:    "APARTMENT",
		SurfaceM2:  45,
	})
	if err != nil {
		t.Fatalf("create lot: %v", err)
	}
	tenant, err := leaseSvc.CreateTenant(ctx, leaseapp.CreateTenantCmd{
		FullName: "Marie Martin",
		Email:    "marie@example.fr",
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	lease, err := leaseSvc.CreateLease(ctx, leaseapp.CreateLeaseCmd{
		LegalEntityID:   entity.ID,
		LotID:           lot.ID,
		TenantID:        tenant.ID,
		LeaseType:       "RESIDENTIAL_UNFURNISHED",
		RentNetCents:    600_00,
		ChargesCents:    50_00,
		GLIPolicyNumber: "GLI-AZUR-01",
		StartDate:       "2026-01-01",
	})
	if err != nil {
		t.Fatalf("create lease: %v", err)
	}
	if _, err := leaseSvc.GenerateMonthlyRentCalls(ctx, entity.ID, "2026-01"); err != nil {
		t.Fatalf("generate rent calls: %v", err)
	}

	for _, body := range []string{"%PDF-1.4 QUITTANCE 1", "%PDF-1.4 QUITTANCE 2"} {
		if _, err := docSvc.AttachFileToR2(ctx, application.AttachFileCmd{
			LegalEntityID: entity.ID,
			LeaseID:       lease.ID,
			Category:      "QUITTANCE",
			FileName:      "quittance.pdf",
			ContentType:   "application/pdf",
			Content:       []byte(body),
		}); err != nil {
			t.Fatalf("attach quittance.pdf: %v", err)
		}
	}

	dossier, err := docSvc.AssembleGLIDossier(ctx, lease.ID, "")
	if err != nil {
		t.Fatalf("AssembleGLIDossier: %v", err)
	}

	rc, err := r2Store.Download(ctx, dossier.ArchiveR2Key)
	if err != nil {
		t.Fatalf("download GLI zip: %v", err)
	}
	defer rc.Close()
	zipBytes, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read GLI zip: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("open GLI zip: %v", err)
	}

	entrySet := make(map[string]bool)
	for _, f := range zr.File {
		entrySet[f.Name] = true
	}
	for _, expected := range []string{"pieces_jointes/quittance.pdf", "pieces_jointes/quittance_2.pdf"} {
		if !entrySet[expected] {
			t.Fatalf("expected %q in GLI archive, got %v", expected, dossier.IncludedFiles)
		}
	}
}

func TestDocument_GLITemplateResolutionAndScopeValidation(t *testing.T) {
	ctx := context.Background()

	entityRepo := portfoliomemory.NewLegalEntityRepo()
	shareholderRepo := portfoliomemory.NewShareholderRepo()
	propertyRepo := portfoliomemory.NewPropertyRepo()
	lotRepo := portfoliomemory.NewLotRepo()
	portfolioUoW := portfoliomemory.NewUnitOfWork(entityRepo, shareholderRepo, propertyRepo, lotRepo)
	portfolioSvc := portfolioapp.NewService(entityRepo, shareholderRepo, propertyRepo, lotRepo, portfolioUoW)

	tenantRepo := leasememory.NewTenantRepo()
	leaseRepo := leasememory.NewLeaseRepo()
	rentCallRepo := leasememory.NewRentCallRepo()
	leaseUoW := leasememory.NewUnitOfWork(tenantRepo, leaseRepo, rentCallRepo)
	leaseSvc := leaseapp.NewService(tenantRepo, leaseRepo, rentCallRepo, leaseUoW)

	spyStore := &spyObjectStorage{ObjectStorage: r2.NewInMemoryStorage()}
	pdfEngine := templating.NewInMemoryPdfEngine()
	docxEngine := templating.NewDocxEngine()

	docTemplateRepo := docmemory.NewDocumentTemplateRepo()
	storedFileRepo := docmemory.NewStoredFileRepo()
	gliDossierRepo := docmemory.NewGLIDossierRepo()
	docUoW := docmemory.NewUnitOfWork(docTemplateRepo, storedFileRepo, gliDossierRepo)
	docSvc := application.NewService(
		docTemplateRepo,
		storedFileRepo,
		gliDossierRepo,
		docUoW,
		spyStore,
		docxEngine,
		pdfEngine,
		portfolioSvc,
		leaseSvc,
		nil,
	)
	docHandler := dochttp.NewHandler(docSvc)

	// 1. AttachFileToR2 rejects any non-UUID LeaseID or LegalEntityID before uploading to R2
	for _, badID := range []string{"../secret", "a/b", "lease_1"} {
		if _, err := docSvc.AttachFileToR2(ctx, application.AttachFileCmd{
			LeaseID:     badID,
			Category:    "ATTACHMENT",
			FileName:    "test.pdf",
			ContentType: "application/pdf",
			Content:     []byte("%PDF-1.4 test"),
		}); !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("expected ErrValidation for LeaseID %q, got %v", badID, err)
		}
		if _, err := docSvc.AttachFileToR2(ctx, application.AttachFileCmd{
			LegalEntityID: badID,
			Category:      "ATTACHMENT",
			FileName:      "test.pdf",
			ContentType:   "application/pdf",
			Content:       []byte("%PDF-1.4 test"),
		}); !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("expected ErrValidation for LegalEntityID %q, got %v", badID, err)
		}
	}
	if spyStore.uploads != 0 {
		t.Fatalf("expected 0 R2 uploads on rejected scope IDs, got %d", spyStore.uploads)
	}

	entity, err := portfolioSvc.CreateLegalEntity(ctx, portfolioapp.CreateLegalEntityCmd{
		Name:              "SCI UUIDV7",
		EntityType:        "SCI_IS",
		SIRET:             "66123456700015",
		ShareCapitalCents: 1000_00,
		TotalShares:       100,
	})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}
	prop, err := portfolioSvc.CreateProperty(ctx, portfolioapp.CreatePropertyCmd{
		LegalEntityID:      entity.ID,
		Label:              "Immeuble V7",
		PropertyType:       "BUILDING",
		AccountingCategory: "FIXED_ASSET",
		AcquisitionCents:   100000_00,
	})
	if err != nil {
		t.Fatalf("create property: %v", err)
	}
	lot, err := portfolioSvc.CreateLot(ctx, portfolioapp.CreateLotCmd{
		PropertyID: prop.ID,
		Label:      "Lot V7",
		LotType:    "APARTMENT",
		SurfaceM2:  50,
	})
	if err != nil {
		t.Fatalf("create lot: %v", err)
	}
	tenant, err := leaseSvc.CreateTenant(ctx, leaseapp.CreateTenantCmd{
		FullName: "Paul Martin",
		Email:    "paul@example.fr",
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	lease, err := leaseSvc.CreateLease(ctx, leaseapp.CreateLeaseCmd{
		LegalEntityID:   entity.ID,
		LotID:           lot.ID,
		TenantID:        tenant.ID,
		LeaseType:       "RESIDENTIAL_UNFURNISHED",
		RentNetCents:    700_00,
		ChargesCents:    50_00,
		GLIPolicyNumber: "GLI-V7-01",
		StartDate:       "2026-01-01",
	})
	if err != nil {
		t.Fatalf("create lease: %v", err)
	}
	if _, err := leaseSvc.GenerateMonthlyRentCalls(ctx, entity.ID, "2026-01"); err != nil {
		t.Fatalf("generate rent calls: %v", err)
	}

	mux := http.NewServeMux()
	docHandler.RegisterRoutes(mux)

	// 2. POST /api/v1/leases/{id}/gli-dossier with {"template_id": ""} -> 201 Created with TemplateID == ""
	reqDefault := httptest.NewRequest(http.MethodPost, "/api/v1/leases/"+lease.ID+"/gli-dossier", strings.NewReader(`{"template_id":""}`))
	reqDefault.Header.Set("Content-Type", "application/json")
	recDefault := httptest.NewRecorder()
	mux.ServeHTTP(recDefault, reqDefault)
	if recDefault.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created from /gli-dossier with empty template_id, got %d: %s", recDefault.Code, recDefault.Body.String())
	}
	var defaultDossier application.GLIDossierArchiveDTO
	if err := json.Unmarshal(recDefault.Body.Bytes(), &defaultDossier); err != nil {
		t.Fatalf("unmarshal GLI dossier: %v", err)
	}
	if defaultDossier.TemplateID != "" {
		t.Fatalf("expected empty TemplateID when built-in default GLI template is used, got %q", defaultDossier.TemplateID)
	}

	// 3. POST /api/v1/leases/{id}/gli-dossier with valid-syntax unknown template_id -> 404 Not Found
	unknownTplID := id.MustNewV7()
	reqMissing := httptest.NewRequest(http.MethodPost, "/api/v1/leases/"+lease.ID+"/gli-dossier", strings.NewReader(`{"template_id":"`+unknownTplID+`"}`))
	reqMissing.Header.Set("Content-Type", "application/json")
	recMissing := httptest.NewRecorder()
	mux.ServeHTTP(recMissing, reqMissing)
	if recMissing.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for unknown template_id UUID, got %d: %s", recMissing.Code, recMissing.Body.String())
	}
}
