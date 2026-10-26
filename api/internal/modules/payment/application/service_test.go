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

	leaseapp "github.com/yannamsellem/gestion-immo/internal/modules/lease/application"
	leasememory "github.com/yannamsellem/gestion-immo/internal/modules/lease/infrastructure/memory"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
	paymentinfra "github.com/yannamsellem/gestion-immo/internal/modules/payment/infrastructure"
	paymentmemory "github.com/yannamsellem/gestion-immo/internal/modules/payment/infrastructure/memory"
	paymenthttp "github.com/yannamsellem/gestion-immo/internal/modules/payment/interfaces/http"
	paymentmcp "github.com/yannamsellem/gestion-immo/internal/modules/payment/interfaces/mcp"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

func newTestPaymentSuite() (*application.Service, *leaseapp.Service) {
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

	paymentSvc := application.NewService(paymentRepo, cashJournalRepo, paymentUoW, leaseSvc, bankProvider, csvImporter)
	return paymentSvc, leaseSvc
}

func seedTestLeaseWithRentCalls(t *testing.T, leaseSvc *leaseapp.Service, entityID string, periods ...string) leaseapp.LeaseDTO {
	t.Helper()
	ctx := context.Background()

	tenant, err := leaseSvc.CreateTenant(ctx, leaseapp.CreateTenantCmd{
		FullName: "JEAN DUPONT",
		Email:    "jean.dupont@example.fr",
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	lse, err := leaseSvc.CreateLease(ctx, leaseapp.CreateLeaseCmd{
		LegalEntityID:   entityID,
		LotID:           "lot_101",
		TenantID:        tenant.ID,
		LeaseType:       "RESIDENTIAL_UNFURNISHED",
		RentNetCents:    money.AmountCents(750_00),
		ChargesCents:    money.AmountCents(50_00),
		DueDayOfMonth:   5,
		GLIPolicyNumber: "GLI-2026-9988",
		StartDate:       "2026-01-01",
	})
	if err != nil {
		t.Fatalf("create lease: %v", err)
	}

	for _, p := range periods {
		if _, err := leaseSvc.GenerateMonthlyRentCalls(ctx, entityID, p); err != nil {
			t.Fatalf("generate rent calls %s: %v", p, err)
		}
	}
	return lse
}

func TestImportPaymentsCSV_WholeEuroParsingAndExternalRefIdempotency(t *testing.T) {
	paymentSvc, leaseSvc := newTestPaymentSuite()
	ctx := context.Background()

	lse := seedTestLeaseWithRentCalls(t, leaseSvc, "ent_csv_1", "2026-01", "2026-02")

	csvContent := strings.Join([]string{
		"lease_id,amount_eur,label,payment_date,external_ref",
		lse.ID + ",800,Loyer janvier 2026,2026-01-05,REF-JAN-001",
		lse.ID + ",800 €,Loyer février 2026,2026-02-05,REF-FEB-001",
	}, "\n")

	res, err := paymentSvc.ImportPaymentsCSV(ctx, "ent_csv_1", strings.NewReader(csvContent))
	if err != nil {
		t.Fatalf("ImportPaymentsCSV: %v", err)
	}
	if res.ImportedCount != 2 || res.TotalImportedCents != money.AmountCents(1600_00) {
		t.Fatalf("expected 2 payments totaling 1600_00 cents, got count=%d total=%d", res.ImportedCount, res.TotalImportedCents)
	}
	for i, p := range res.Payments {
		if p.AmountCents != money.AmountCents(800_00) || p.UnallocatedCents != 0 {
			t.Fatalf("payment[%d] expected 800_00 cents allocated, got %+v", i, p)
		}
	}

	unpaid, err := leaseSvc.ListUnpaidRentCallsByLease(ctx, lse.ID)
	if err != nil || len(unpaid) != 0 {
		t.Fatalf("expected 0 unpaid rent calls after whole-euro CSV import, got %d (err=%v)", len(unpaid), err)
	}

	// Re-importing the same external_ref values must be idempotent (0 new payments imported)
	res2, err := paymentSvc.ImportPaymentsCSV(ctx, "ent_csv_1", strings.NewReader(csvContent))
	if err != nil {
		t.Fatalf("re-import CSV: %v", err)
	}
	if res2.ImportedCount != 0 || res2.TotalImportedCents != 0 {
		t.Fatalf("expected idempotent re-import (0 imported), got %+v", res2)
	}
}

func TestImportPaymentsCSV_AtomicRollbackOnUnresolvableRow(t *testing.T) {
	paymentSvc, leaseSvc := newTestPaymentSuite()
	ctx := context.Background()

	lse := seedTestLeaseWithRentCalls(t, leaseSvc, "ent_csv_atomic", "2026-01", "2026-02")

	// Row 1 has a valid lease_id; Row 2 has an empty lease_id and an unknown tenant label
	csvContent := strings.Join([]string{
		"lease_id,amount_eur,label,payment_date,external_ref",
		lse.ID + ",800,Loyer janvier 2026,2026-01-05,REF-ROW-1",
		",800,VIR SEPA UNKNOWN TENANT,2026-02-05,REF-ROW-2",
	}, "\n")

	_, err := paymentSvc.ImportPaymentsCSV(ctx, "ent_csv_atomic", strings.NewReader(csvContent))
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation when a row has an unresolvable lease_id, got %v", err)
	}

	// Verify Row 1 was NOT persisted and rent calls remain untouched
	payments, err := paymentSvc.ListPaymentsByLease(ctx, lse.ID)
	if err != nil || len(payments) != 0 {
		t.Fatalf("expected 0 persisted payments after failed batch, got %d (err=%v)", len(payments), err)
	}
	unpaid, err := leaseSvc.ListUnpaidRentCallsByLease(ctx, lse.ID)
	if err != nil || len(unpaid) != 2 || unpaid[0].RemainingDueCents != money.AmountCents(800_00) {
		t.Fatalf("expected both rent calls to remain unpaid at 800_00, got %+v (err=%v)", unpaid, err)
	}
}

func TestRecordCashPaymentAndDeposit_CashJournalEntries(t *testing.T) {
	paymentSvc, leaseSvc := newTestPaymentSuite()
	ctx := context.Background()

	lse := seedTestLeaseWithRentCalls(t, leaseSvc, "ent_cash_1", "2026-01")

	cashReceipt, err := paymentSvc.RecordCashPayment(ctx, application.RecordCashPaymentCmd{
		LeaseID:       lse.ID,
		AmountCents:   money.AmountCents(800_00),
		Label:         "Encaissement espèces janvier 2026",
		ReceiptNumber: "CAISSE-20260105-001",
		PaymentDate:   "2026-01-05",
	})
	if err != nil {
		t.Fatalf("RecordCashPayment: %v", err)
	}
	if cashReceipt.Method != domain.MethodCash || cashReceipt.CashJournalEntryID == "" {
		t.Fatalf("expected CASH receipt with CashJournalEntryID, got %+v", cashReceipt)
	}

	depositEntry, err := paymentSvc.RecordCashDepositToBank(ctx, application.RecordCashDepositCmd{
		LegalEntityID: "ent_cash_1",
		AmountCents:   money.AmountCents(800_00),
		Label:         "Remise espèces banque (530 -> 580 -> 512)",
		EntryDate:     "2026-01-06",
	})
	if err != nil {
		t.Fatalf("RecordCashDepositToBank: %v", err)
	}
	if depositEntry.AccountCode != "530" || depositEntry.CounterpartAccountCode != "580" || depositEntry.CreditCents != money.AmountCents(800_00) {
		t.Fatalf("unexpected cash deposit entry: %+v", depositEntry)
	}

	entries, err := paymentSvc.ListCashJournalByEntity(ctx, "ent_cash_1")
	if err != nil || len(entries) != 2 {
		t.Fatalf("expected 2 cash journal entries, got %d (err=%v)", len(entries), err)
	}
	if entries[0].EntryType != domain.CashEntryReceipt || entries[0].DebitCents != money.AmountCents(800_00) {
		t.Fatalf("unexpected first cash journal entry: %+v", entries[0])
	}
	if entries[1].EntryType != domain.CashEntryBankDeposit || entries[1].CreditCents != money.AmountCents(800_00) {
		t.Fatalf("unexpected second cash journal entry: %+v", entries[1])
	}
}

func TestImportPaymentsOFX_BankStatement(t *testing.T) {
	paymentSvc, leaseSvc := newTestPaymentSuite()
	ctx := context.Background()

	lse := seedTestLeaseWithRentCalls(t, leaseSvc, "ent_ofx_1", "2026-01")

	ofxPayload := `<OFX>
<BANKMSGSRSV1>
<STMTTRN>
<TRNTYPE>CREDIT
<DTPOSTED>20260107120000
<TRNAMT>800.00
<FITID>OFX-FITID-20260107
<CHECKNUM>` + lse.ID + `
<NAME>JEAN DUPONT
<MEMO>Loyer janvier 2026
</STMTTRN>
</BANKMSGSRSV1>
</OFX>`

	res, err := paymentSvc.ImportPaymentsCSV(ctx, "ent_ofx_1", strings.NewReader(ofxPayload))
	if err != nil {
		t.Fatalf("ImportPaymentsCSV (OFX): %v", err)
	}
	if res.ImportedCount != 1 || res.TotalImportedCents != money.AmountCents(800_00) {
		t.Fatalf("expected 1 OFX payment of 800_00, got %+v", res)
	}
	if res.Payments[0].ExternalRef != "OFX-FITID-20260107" {
		t.Fatalf("expected ExternalRef OFX-FITID-20260107, got %q", res.Payments[0].ExternalRef)
	}
}

func TestPaymentModule_AllHTTPAndMCPEndpoints(t *testing.T) {
	paymentSvc, leaseSvc := newTestPaymentSuite()
	lse := seedTestLeaseWithRentCalls(t, leaseSvc, "ent_http_1", "2026-01", "2026-02", "2026-03", "2026-04")

	handler := paymenthttp.NewHandler(paymentSvc)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// 1. POST /api/v1/payments/cash
	reqCash := httptest.NewRequest(http.MethodPost, "/api/v1/payments/cash", strings.NewReader(
		`{"lease_id":"`+lse.ID+`","amount_cents":80000,"label":"Loyer janvier 2026","payment_date":"2026-01-05"}`,
	))
	recCash := httptest.NewRecorder()
	mux.ServeHTTP(recCash, reqCash)
	if recCash.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/payments/cash status = %d, body = %s", recCash.Code, recCash.Body.String())
	}

	// 2. POST /api/v1/payments/bank
	reqBank := httptest.NewRequest(http.MethodPost, "/api/v1/payments/bank", strings.NewReader(
		`{"lease_id":"`+lse.ID+`","amount_cents":80000,"method":"BANK_TRANSFER","label":"Loyer février 2026","payment_date":"2026-02-05"}`,
	))
	recBank := httptest.NewRecorder()
	mux.ServeHTTP(recBank, reqBank)
	if recBank.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/payments/bank status = %d, body = %s", recBank.Code, recBank.Body.String())
	}

	// 3. POST /api/v1/payments/cash-deposit
	reqDep := httptest.NewRequest(http.MethodPost, "/api/v1/payments/cash-deposit", strings.NewReader(
		`{"legal_entity_id":"ent_http_1","amount_cents":80000,"label":"Remise banque","entry_date":"2026-01-06"}`,
	))
	recDep := httptest.NewRecorder()
	mux.ServeHTTP(recDep, reqDep)
	if recDep.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/payments/cash-deposit status = %d, body = %s", recDep.Code, recDep.Body.String())
	}

	// 4. POST /api/v1/payments/import-csv
	csvBody := "lease_id,amount_eur,label,payment_date,external_ref\n" + lse.ID + ",800,Loyer mars 2026,2026-03-05,REF-MAR-1\n"
	reqCSV := httptest.NewRequest(http.MethodPost, "/api/v1/payments/import-csv?entity_id=ent_http_1", strings.NewReader(csvBody))
	reqCSV.Header.Set("Content-Type", "text/csv")
	recCSV := httptest.NewRecorder()
	mux.ServeHTTP(recCSV, reqCSV)
	if recCSV.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/payments/import-csv status = %d, body = %s", recCSV.Code, recCSV.Body.String())
	}

	// 5. GET /api/v1/entities/{id}/negative-positions (2026-04 remains unpaid = 800_00)
	reqNeg := httptest.NewRequest(http.MethodGet, "/api/v1/entities/ent_http_1/negative-positions", nil)
	recNeg := httptest.NewRecorder()
	mux.ServeHTTP(recNeg, reqNeg)
	if recNeg.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/entities/ent_http_1/negative-positions status = %d, body = %s", recNeg.Code, recNeg.Body.String())
	}
	var negPositions []application.NegativePositionDTO
	if err := json.Unmarshal(recNeg.Body.Bytes(), &negPositions); err != nil {
		t.Fatalf("decode negative positions: %v", err)
	}
	if len(negPositions) != 1 || negPositions[0].TotalUnpaidCents != money.AmountCents(800_00) {
		t.Fatalf("expected 1 negative position of 800_00, got %+v", negPositions)
	}

	// 6. GET /api/v1/entities/{id}/cash-journal
	reqJournal := httptest.NewRequest(http.MethodGet, "/api/v1/entities/ent_http_1/cash-journal", nil)
	recJournal := httptest.NewRecorder()
	mux.ServeHTTP(recJournal, reqJournal)
	if recJournal.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/entities/ent_http_1/cash-journal status = %d, body = %s", recJournal.Code, recJournal.Body.String())
	}
	var journalEntries []application.CashJournalEntryDTO
	if err := json.Unmarshal(recJournal.Body.Bytes(), &journalEntries); err != nil || len(journalEntries) != 2 {
		t.Fatalf("expected 2 cash journal entries, got %d (err=%v)", len(journalEntries), err)
	}

	// 7. Verify all 3 MCP tools (record_cash_payment, record_bank_payment, list_negative_positions)
	reg := mcpserver.NewRegistry(func(_ context.Context, token string) (string, error) {
		return "usr_1", nil
	})
	paymentmcp.RegisterTools(reg, paymentSvc)

	tools := reg.Tools()
	expectedTools := []string{"record_cash_payment", "record_bank_payment", "list_negative_positions"}
	if len(tools) != len(expectedTools) {
		t.Fatalf("expected %d MCP tools, got %d", len(expectedTools), len(tools))
	}
	for i, want := range expectedTools {
		if tools[i].Name != want {
			t.Fatalf("expected tool[%d] = %q, got %q", i, want, tools[i].Name)
		}
	}

	// Call list_negative_positions and record_bank_payment via MCP JSON-RPC
	mcpListBody := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_negative_positions","arguments":{"legal_entity_id":"ent_http_1"}}}`
	reqMCPList := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(mcpListBody))
	reqMCPList.Header.Set("Authorization", "Bearer valid")
	recMCPList := httptest.NewRecorder()
	reg.ServeHTTP(recMCPList, reqMCPList)
	if recMCPList.Code != http.StatusOK || !strings.Contains(recMCPList.Body.String(), "2026-04") {
		t.Fatalf("unexpected MCP list_negative_positions response (%d): %s", recMCPList.Code, recMCPList.Body.String())
	}

	mcpPayBody := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"record_bank_payment","arguments":{"lease_id":"` + lse.ID + `","amount_cents":80000,"label":"Loyer avril 2026"}}}`
	reqMCPPay := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(mcpPayBody))
	reqMCPPay.Header.Set("Authorization", "Bearer valid")
	recMCPPay := httptest.NewRecorder()
	reg.ServeHTTP(recMCPPay, reqMCPPay)
	if recMCPPay.Code != http.StatusOK {
		t.Fatalf("unexpected MCP record_bank_payment response (%d): %s", recMCPPay.Code, recMCPPay.Body.String())
	}
}
