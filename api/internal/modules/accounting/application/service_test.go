package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/infrastructure"
	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/infrastructure/memory"
	acchttp "github.com/yannamsellem/gestion-immo/internal/modules/accounting/interfaces/http"
	accmcp "github.com/yannamsellem/gestion-immo/internal/modules/accounting/interfaces/mcp"
	portfolioapp "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/application"
	portfoliomem "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/infrastructure/memory"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

type spyAccountBalanceRepo struct {
	domain.AccountBalanceRepository
	lastUpserted []domain.AccountBalanceLine
}

func (s *spyAccountBalanceRepo) UpsertAccountBalanceLines(ctx context.Context, entityID string, fiscalYear int, lines []domain.AccountBalanceLine) error {
	s.lastUpserted = append([]domain.AccountBalanceLine(nil), lines...)
	return s.AccountBalanceRepository.UpsertAccountBalanceLines(ctx, entityID, fiscalYear, lines)
}

func newTestPortfolioService() *portfolioapp.Service {
	entityRepo := portfoliomem.NewLegalEntityRepo()
	shareholderRepo := portfoliomem.NewShareholderRepo()
	propertyRepo := portfoliomem.NewPropertyRepo()
	lotRepo := portfoliomem.NewLotRepo()
	uow := portfoliomem.NewUnitOfWork(entityRepo, shareholderRepo, propertyRepo, lotRepo)
	return portfolioapp.NewService(entityRepo, shareholderRepo, propertyRepo, lotRepo, uow)
}

func newTestAccountingService(portfolioSvc application.PortfolioPort) (*application.Service, *spyAccountBalanceRepo, *r2.InMemoryStorage) {
	journalEntries := memory.NewJournalEntryRepo()
	accountBalances := &spyAccountBalanceRepo{AccountBalanceRepository: memory.NewAccountBalanceRepo()}
	annualReports := memory.NewAnnualAccountReportRepo()
	uow := memory.NewUnitOfWork(journalEntries, accountBalances, annualReports)
	storage := r2.NewInMemoryStorage()
	pdfEngine := templating.NewInMemoryPdfEngine()
	svc := application.NewService(
		journalEntries,
		accountBalances,
		annualReports,
		uow,
		storage,
		pdfEngine,
		infrastructure.NewFECExporter(),
		infrastructure.NewAnnualAccountsPDFRenderer(pdfEngine),
		portfolioSvc,
	)
	return svc, accountBalances, storage
}

func TestRecordJournalEntry_RejectsUnbalancedAndSingleLineEntries(t *testing.T) {
	svc, _, _ := newTestAccountingService(nil)
	ctx := context.Background()

	// Fewer than 2 lines must return ErrValidation
	_, err := svc.RecordJournalEntry(ctx, application.RecordJournalEntryCmd{
		LegalEntityID: "ent_val_1",
		FiscalYear:    2025,
		Label:         "Single line entry",
		Lines: []application.JournalLineInput{
			{AccountCode: "512000", DebitCents: 1000_00},
		},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation for < 2 lines, got %v", err)
	}

	// Unbalanced debit != credit must return ErrUnbalancedJournalEntry
	_, err = svc.RecordJournalEntry(ctx, application.RecordJournalEntryCmd{
		LegalEntityID: "ent_val_1",
		FiscalYear:    2025,
		Label:         "Unbalanced entry",
		Lines: []application.JournalLineInput{
			{AccountCode: "512000", DebitCents: 1200_00},
			{AccountCode: "706100", CreditCents: 1000_00},
		},
	})
	if !errors.Is(err, domain.ErrUnbalancedJournalEntry) {
		t.Fatalf("expected ErrUnbalancedJournalEntry, got %v", err)
	}
}

func TestRecordJournalEntry_UpdatesSeededAccountByAccountCodeAndDeduplicatesUpserts(t *testing.T) {
	svc, balancesSpy, _ := newTestAccountingService(nil)
	ctx := context.Background()
	entityID := "ent_sync_1"
	fiscalYear := 2025

	// Seed an existing AccountBalanceLine with LineID "rev_locaux" on AccountCode "706100"
	err := svc.UpsertAccountBalanceLines(ctx, entityID, fiscalYear, []domain.AccountBalanceLine{
		{
			ID:              entityID + "_rev_locaux",
			LegalEntityID:   entityID,
			FiscalYear:      fiscalYear,
			LineID:          "rev_locaux",
			AccountCode:     "706100",
			Label:           "LOCATIONS LOCAUX",
			Section:         string(domain.SectionOperatingRevenue),
			NetNCents:       money.AmountCents(100000_00),
			NetNMinus1Cents: money.AmountCents(90000_00),
		},
	})
	if err != nil {
		t.Fatalf("seed balance lines: %v", err)
	}

	// Post a journal entry with two 706100 lines (without explicit LineID) and one 512000 line
	_, err = svc.RecordJournalEntry(ctx, application.RecordJournalEntryCmd{
		LegalEntityID: entityID,
		FiscalYear:    fiscalYear,
		JournalCode:   "VT",
		EntryDate:     "2025-03-15",
		Label:         "Facturation loyers mars",
		Lines: []application.JournalLineInput{
			{AccountCode: "512000", AccountLabel: "Banque", DebitCents: 8000_00},
			{AccountCode: "706100", AccountLabel: "Loyer Lot 1", CreditCents: 5000_00},
			{AccountCode: "706100", AccountLabel: "Loyer Lot 2", CreditCents: 3000_00},
		},
	})
	if err != nil {
		t.Fatalf("record journal entry: %v", err)
	}

	// Verify UpsertAccountBalanceLines received exactly 2 deduplicated lines ("acct_512000" and "rev_locaux")
	if len(balancesSpy.lastUpserted) != 2 {
		t.Fatalf("expected 2 deduplicated upserted balance lines, got %d: %+v", len(balancesSpy.lastUpserted), balancesSpy.lastUpserted)
	}

	allLines, err := balancesSpy.ListAccountBalanceLines(ctx, entityID, fiscalYear)
	if err != nil {
		t.Fatalf("list balance lines: %v", err)
	}
	if len(allLines) != 2 {
		t.Fatalf("expected 2 total balance lines (no duplicate acct_706100), got %d: %+v", len(allLines), allLines)
	}

	var foundRevLocaux bool
	for _, l := range allLines {
		if l.LineID == "acct_706100" {
			t.Fatalf("unexpected duplicate balance line acct_706100 created alongside rev_locaux: %+v", l)
		}
		if l.LineID == "rev_locaux" {
			foundRevLocaux = true
			if l.NetNCents != money.AmountCents(108000_00) {
				t.Fatalf("expected rev_locaux NetNCents 108000_00, got %d", l.NetNCents)
			}
		}
	}
	if !foundRevLocaux {
		t.Fatal("expected rev_locaux balance line to be preserved")
	}
}

func TestPreviewFinancialStatements_DisabledLinesRecalculateTotals(t *testing.T) {
	svc, _, _ := newTestAccountingService(nil)
	ctx := context.Background()
	entityID := "ent_preview_1"
	fiscalYear := 2025

	if err := svc.UpsertAccountBalanceLines(ctx, entityID, fiscalYear, []domain.AccountBalanceLine{
		{LineID: "rev_locaux", AccountCode: "706100", Label: "LOCATIONS", Section: "OPERATING_REVENUE", NetNCents: 150000_00, NetNMinus1Cents: 140000_00},
		{LineID: "rev_comm", AccountCode: "706200", Label: "LOYERS COMMERCIAUX", Section: "OPERATING_REVENUE", NetNCents: 50000_00, NetNMinus1Cents: 40000_00},
		{LineID: "chg_loc", AccountCode: "614000", Label: "CHARGES", Section: "EXTERNAL_EXPENSES", NetNCents: 60000_00, NetNMinus1Cents: 50000_00},
	}); err != nil {
		t.Fatalf("upsert balance lines: %v", err)
	}

	full, err := svc.PreviewFinancialStatements(ctx, entityID, fiscalYear, nil)
	if err != nil {
		t.Fatalf("preview full: %v", err)
	}
	if full.TotalOperatingRevenueNCents != 200000_00 || full.NetIncomeNCents != 140000_00 {
		t.Fatalf("unexpected full preview totals: rev=%d net=%d", full.TotalOperatingRevenueNCents, full.NetIncomeNCents)
	}

	filtered, err := svc.PreviewFinancialStatements(ctx, entityID, fiscalYear, []string{"rev_comm"})
	if err != nil {
		t.Fatalf("preview filtered: %v", err)
	}
	if filtered.TotalOperatingRevenueNCents != 150000_00 || filtered.NetIncomeNCents != 90000_00 {
		t.Fatalf("unexpected filtered preview totals: rev=%d net=%d", filtered.TotalOperatingRevenueNCents, filtered.NetIncomeNCents)
	}
}

func TestGenerateAnnualAccountsPDF_UsesLivePortfolioMetadataAndUploadsToR2(t *testing.T) {
	portfolioSvc := newTestPortfolioService()
	svc, _, storage := newTestAccountingService(portfolioSvc)
	ctx := context.Background()

	ent, err := portfolioSvc.CreateLegalEntity(ctx, portfolioapp.CreateLegalEntityCmd{
		Name:              "SCI PARC MONCEAU",
		EntityType:        "SCI_IS",
		SIRET:             "89911122200015",
		NAFCode:           "6820B",
		Address:           "45 AVENUE DE MESSINE, 75008 PARIS",
		ShareCapitalCents: money.AmountCents(500000_00),
		TotalShares:       1000,
	})
	if err != nil {
		t.Fatalf("create legal entity: %v", err)
	}

	if _, err := portfolioSvc.UpsertShareholder(ctx, portfolioapp.UpsertShareholderCmd{
		LegalEntityID:         ent.ID,
		FullName:              "CLAIRE DE LA TOUR",
		SharesCount:           750,
		CurrentAccountCents:   money.AmountCents(120000_00),
		RemunerationPaidCents: money.AmountCents(18000_00),
	}); err != nil {
		t.Fatalf("upsert shareholder: %v", err)
	}

	stored, err := svc.GenerateAnnualAccountsPDF(ctx, ent.ID, 2025, nil)
	if err != nil {
		t.Fatalf("generate annual accounts pdf: %v", err)
	}
	if stored.R2Key == "" || stored.SizeBytes <= 0 || stored.DownloadURL == "" {
		t.Fatalf("expected populated StoredFileDTO with R2Key, SizeBytes, and DownloadURL, got %+v", stored)
	}

	rc, err := storage.Download(ctx, stored.R2Key)
	if err != nil {
		t.Fatalf("download generated pdf from R2: %v", err)
	}
	defer rc.Close()
	pdfBytes, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read pdf bytes: %v", err)
	}
	pdfText := string(pdfBytes)

	for _, expected := range []string{"SCI PARC MONCEAU", "89911122200015", "45 AVENUE DE MESSINE, 75008 PARIS", "CLAIRE DE LA TOUR"} {
		if !strings.Contains(pdfText, expected) {
			t.Fatalf("expected generated PDF to contain live portfolio metadata %q, got: %s", expected, pdfText)
		}
	}
	if strings.Contains(pdfText, "SARL INVESTISSIMO") || strings.Contains(pdfText, "38012345600018") {
		t.Fatalf("expected generated PDF not to use hardcoded SARL INVESTISSIMO metadata when entity exists in portfolioSvc: %s", pdfText)
	}
}

func TestExportEntityFEC_ProducesNormative18ColumnTSV(t *testing.T) {
	svc, _, _ := newTestAccountingService(nil)
	ctx := context.Background()
	entityID := "ent_fec_1"

	_, err := svc.RecordJournalEntry(ctx, application.RecordJournalEntryCmd{
		LegalEntityID:  entityID,
		FiscalYear:     2025,
		JournalCode:    "VT",
		JournalLabel:   "Journal des Ventes",
		EntryNumber:    "VT-2025-0001",
		EntryDate:      "2025-01-10",
		DocumentRef:    "QUITT-001",
		DocumentDate:   "2025-01-10",
		ValidationDate: "2025-01-10",
		Label:          "Loyer Janvier 2025",
		Lines: []application.JournalLineInput{
			{AccountCode: "411000", AccountLabel: "Clients Locataires", SubAccountCode: "LOC01", SubAccountLabel: "Dupont", DebitCents: 1250_50},
			{AccountCode: "706100", AccountLabel: "Locations", CreditCents: 1250_50},
		},
	})
	if err != nil {
		t.Fatalf("record journal entry: %v", err)
	}

	fecBytes, err := svc.ExportEntityFEC(ctx, entityID, 2025)
	if err != nil {
		t.Fatalf("export fec: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(fecBytes)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 1 header + 2 entry lines = 3 lines, got %d: %s", len(lines), string(fecBytes))
	}
	headerCols := strings.Split(lines[0], "\t")
	if len(headerCols) != 18 || headerCols[0] != "JournalCode" || headerCols[17] != "Idevise" {
		t.Fatalf("expected 18 normative FEC header columns, got %d: %v", len(headerCols), headerCols)
	}
	row1Cols := strings.Split(lines[1], "\t")
	if len(row1Cols) != 18 || row1Cols[11] != "1250,50" || row1Cols[12] != "0,00" {
		t.Fatalf("unexpected FEC row 1 columns: %v", row1Cols)
	}
}

func TestAccountingHTTPAndMCPInterfaces(t *testing.T) {
	portfolioSvc := newTestPortfolioService()
	svc, _, _ := newTestAccountingService(portfolioSvc)
	ctx := context.Background()

	ent, err := portfolioSvc.CreateLegalEntity(ctx, portfolioapp.CreateLegalEntityCmd{
		Name:              "SCI VENDOME",
		EntityType:        "SCI_IS",
		SIRET:             "52198765400019",
		NAFCode:           "6820B",
		Address:           "10 PLACE VENDOME, 75001 PARIS",
		ShareCapitalCents: money.AmountCents(100000_00),
		TotalShares:       100,
	})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}

	handler := acchttp.NewHandler(svc)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// 1. PUT /api/v1/entities/{id}/statements/lines
	putLinesBody := `{"fiscal_year":2025,"lines":[{"line_id":"rev_locaux","account_code":"706100","label":"LOCATIONS LOCAUX","section":"OPERATING_REVENUE","net_n_cents":10000000,"net_n_minus_1_cents":9500000},{"line_id":"chg_loc","account_code":"614000","label":"CHARGES LOCATIVES","section":"EXTERNAL_EXPENSES","net_n_cents":3000000,"net_n_minus_1_cents":2500000}]}`
	reqPut := httptest.NewRequest(http.MethodPut, "/api/v1/entities/"+ent.ID+"/statements/lines", strings.NewReader(putLinesBody))
	recPut := httptest.NewRecorder()
	mux.ServeHTTP(recPut, reqPut)
	if recPut.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content on PUT statements/lines, got %d: %s", recPut.Code, recPut.Body.String())
	}

	// 2. POST /api/v1/entities/{id}/journal-entries
	postEntryBody := `{"fiscal_year":2025,"journal_code":"BQ","entry_date":"2025-02-01","label":"Encaissement loyer fevrier","lines":[{"account_code":"512000","account_label":"Banque","debit_cents":200000},{"account_code":"706100","account_label":"Locations","credit_cents":200000}]}`
	reqPostEntry := httptest.NewRequest(http.MethodPost, "/api/v1/entities/"+ent.ID+"/journal-entries", strings.NewReader(postEntryBody))
	recPostEntry := httptest.NewRecorder()
	mux.ServeHTTP(recPostEntry, reqPostEntry)
	if recPostEntry.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on POST journal-entries, got %d: %s", recPostEntry.Code, recPostEntry.Body.String())
	}

	// 3. GET /api/v1/entities/{id}/journal-entries
	reqGetEntries := httptest.NewRequest(http.MethodGet, "/api/v1/entities/"+ent.ID+"/journal-entries?fiscal_year=2025", nil)
	recGetEntries := httptest.NewRecorder()
	mux.ServeHTTP(recGetEntries, reqGetEntries)
	if recGetEntries.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET journal-entries, got %d: %s", recGetEntries.Code, recGetEntries.Body.String())
	}
	var entriesList []application.JournalEntryDTO
	if err := json.Unmarshal(recGetEntries.Body.Bytes(), &entriesList); err != nil || len(entriesList) != 1 {
		t.Fatalf("expected 1 journal entry from GET journal-entries, got %d (err=%v)", len(entriesList), err)
	}

	// 4. POST /api/v1/entities/{id}/statements/preview
	reqPreview := httptest.NewRequest(http.MethodPost, "/api/v1/entities/"+ent.ID+"/statements/preview", strings.NewReader(`{"fiscal_year":2025,"disabled_line_ids":["chg_loc"]}`))
	recPreview := httptest.NewRecorder()
	mux.ServeHTTP(recPreview, reqPreview)
	if recPreview.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on POST statements/preview, got %d: %s", recPreview.Code, recPreview.Body.String())
	}
	var previewDTO domain.ComputedFinancialStatementsDTO
	if err := json.Unmarshal(recPreview.Body.Bytes(), &previewDTO); err != nil {
		t.Fatalf("unmarshal preview response: %v", err)
	}
	// 100 000,00 + 2 000,00 = 102 000,00 € revenue; chg_loc is disabled so NetIncomeNCents == 102 000,00 €
	if previewDTO.TotalOperatingRevenueNCents != 102000_00 || previewDTO.NetIncomeNCents != 102000_00 {
		t.Fatalf("expected TotalOperatingRevenueNCents and NetIncomeNCents 102000_00, got %d / %d", previewDTO.TotalOperatingRevenueNCents, previewDTO.NetIncomeNCents)
	}

	// 5. POST /api/v1/entities/{id}/statements/pdf
	reqPDF := httptest.NewRequest(http.MethodPost, "/api/v1/entities/"+ent.ID+"/statements/pdf", strings.NewReader(`{"fiscal_year":2025}`))
	recPDF := httptest.NewRecorder()
	mux.ServeHTTP(recPDF, reqPDF)
	if recPDF.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on POST statements/pdf, got %d: %s", recPDF.Code, recPDF.Body.String())
	}

	// 6. GET /api/v1/entities/{id}/fec
	reqFEC := httptest.NewRequest(http.MethodGet, "/api/v1/entities/"+ent.ID+"/fec?fiscal_year=2025", nil)
	recFEC := httptest.NewRecorder()
	mux.ServeHTTP(recFEC, reqFEC)
	if recFEC.Code != http.StatusOK || !strings.Contains(recFEC.Body.String(), "JournalCode\tJournalLib") {
		t.Fatalf("expected 200 OK with FEC header on GET fec, got %d: %s", recFEC.Code, recFEC.Body.String())
	}

	// 7. Verify all 6 MCP tools (`record_journal_entry`, `record_expense_invoice`, `get_trial_balance`, `preview_financial_statement`, `generate_annual_accounts_pdf`, `export_fec`)
	reg := mcpserver.NewRegistry(func(_ context.Context, token string) (string, error) {
		if token == "valid" {
			return "usr_1", nil
		}
		return "", errors.New("unauthorized")
	})
	accmcp.RegisterTools(reg, svc)

	tools := reg.Tools()
	expectedTools := []string{
		"record_journal_entry",
		"record_expense_invoice",
		"get_trial_balance",
		"preview_financial_statement",
		"generate_annual_accounts_pdf",
		"export_fec",
	}
	if len(tools) != len(expectedTools) {
		t.Fatalf("expected %d MCP tools, got %d", len(expectedTools), len(tools))
	}
	for i, name := range expectedTools {
		if tools[i].Name != name {
			t.Fatalf("expected tool[%d] = %q, got %q", i, name, tools[i].Name)
		}
	}

	for _, tc := range []struct {
		toolName string
		argsJSON string
		expect   string
	}{
		{
			toolName: "record_journal_entry",
			argsJSON: `{"legal_entity_id":"` + ent.ID + `","fiscal_year":2025,"journal_code":"OD","entry_date":"2025-03-01","label":"Ecriture OD MCP","lines":[{"account_code":"512000","debit_cents":50000},{"account_code":"706100","credit_cents":50000}]}`,
			expect:   "Ecriture OD MCP",
		},
		{
			toolName: "record_expense_invoice",
			argsJSON: `{"legal_entity_id":"` + ent.ID + `","supplier_name":"PLOMBERIE EXPRESS","expense_account_code":"615200","label":"Reparation fuite","invoice_ref":"FAC-2025-42","invoice_date":"2025-03-10","amount_ht_cents":100000,"vat_cents":20000}`,
			expect:   "445660",
		},
		{
			toolName: "get_trial_balance",
			argsJSON: `{"legal_entity_id":"` + ent.ID + `","fiscal_year":2025}`,
			expect:   "total_operating_revenue_n_cents",
		},
		{
			toolName: "preview_financial_statement",
			argsJSON: `{"legal_entity_id":"` + ent.ID + `","fiscal_year":2025}`,
			expect:   "total_operating_revenue_n_cents",
		},
		{
			toolName: "generate_annual_accounts_pdf",
			argsJSON: `{"legal_entity_id":"` + ent.ID + `","fiscal_year":2025}`,
			expect:   "ANNUAL_ACCOUNTS_TAX_RETURN",
		},
		{
			toolName: "export_fec",
			argsJSON: `{"legal_entity_id":"` + ent.ID + `","fiscal_year":2025}`,
			expect:   "JournalCode",
		},
	} {
		callBody := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + tc.toolName + `","arguments":` + tc.argsJSON + `}}`
		reqMCP := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(callBody))
		reqMCP.Header.Set("Authorization", "Bearer valid")
		recMCP := httptest.NewRecorder()
		reg.ServeHTTP(recMCP, reqMCP)
		if recMCP.Code != http.StatusOK || !strings.Contains(recMCP.Body.String(), tc.expect) {
			t.Fatalf("unexpected MCP response for %s (%d): %s", tc.toolName, recMCP.Code, recMCP.Body.String())
		}
	}
}
