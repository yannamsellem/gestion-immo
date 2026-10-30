package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	accountingapp "github.com/yannamsellem/gestion-immo/internal/modules/accounting/application"
	accountinginfra "github.com/yannamsellem/gestion-immo/internal/modules/accounting/infrastructure"
	accountingmemory "github.com/yannamsellem/gestion-immo/internal/modules/accounting/infrastructure/memory"
	leaseapp "github.com/yannamsellem/gestion-immo/internal/modules/lease/application"
	leasememory "github.com/yannamsellem/gestion-immo/internal/modules/lease/infrastructure/memory"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
	paymentinfra "github.com/yannamsellem/gestion-immo/internal/modules/payment/infrastructure"
	paymentmemory "github.com/yannamsellem/gestion-immo/internal/modules/payment/infrastructure/memory"
	paymenthttp "github.com/yannamsellem/gestion-immo/internal/modules/payment/interfaces/http"
	paymentmcp "github.com/yannamsellem/gestion-immo/internal/modules/payment/interfaces/mcp"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
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
		LotID:           id.MustNewV7(),
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
	entityID := id.MustNewV7()

	lse := seedTestLeaseWithRentCalls(t, leaseSvc, entityID, "2026-01", "2026-02")

	csvContent := strings.Join([]string{
		"lease_id,amount_eur,label,payment_date,external_ref",
		lse.ID + ",800,Loyer janvier 2026,2026-01-05,REF-JAN-001",
		lse.ID + ",800 €,Loyer février 2026,2026-02-05,REF-FEB-001",
	}, "\n")

	res, err := paymentSvc.ImportPaymentsCSV(ctx, entityID, strings.NewReader(csvContent))
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
	res2, err := paymentSvc.ImportPaymentsCSV(ctx, entityID, strings.NewReader(csvContent))
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
	entityID := id.MustNewV7()

	lse := seedTestLeaseWithRentCalls(t, leaseSvc, entityID, "2026-01", "2026-02")

	// Row 1 has a valid lease_id; Row 2 has an empty lease_id and an unknown tenant label
	csvContent := strings.Join([]string{
		"lease_id,amount_eur,label,payment_date,external_ref",
		lse.ID + ",800,Loyer janvier 2026,2026-01-05,REF-ROW-1",
		",800,VIR SEPA UNKNOWN TENANT,2026-02-05,REF-ROW-2",
	}, "\n")

	_, err := paymentSvc.ImportPaymentsCSV(ctx, entityID, strings.NewReader(csvContent))
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
	entityID := id.MustNewV7()

	lse := seedTestLeaseWithRentCalls(t, leaseSvc, entityID, "2026-01")

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
		LegalEntityID: entityID,
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

	entries, err := paymentSvc.ListCashJournalByEntity(ctx, entityID)
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
	entityID := id.MustNewV7()

	lse := seedTestLeaseWithRentCalls(t, leaseSvc, entityID, "2026-01")

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

	res, err := paymentSvc.ImportPaymentsCSV(ctx, entityID, strings.NewReader(ofxPayload))
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
	entityID := id.MustNewV7()
	mcpUserID := id.MustNewV7()
	lse := seedTestLeaseWithRentCalls(t, leaseSvc, entityID, "2026-01", "2026-02", "2026-03", "2026-04")

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
		`{"legal_entity_id":"`+entityID+`","amount_cents":80000,"label":"Remise banque","entry_date":"2026-01-06"}`,
	))
	recDep := httptest.NewRecorder()
	mux.ServeHTTP(recDep, reqDep)
	if recDep.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/payments/cash-deposit status = %d, body = %s", recDep.Code, recDep.Body.String())
	}

	// 4. POST /api/v1/payments/import-csv
	csvBody := "lease_id,amount_eur,label,payment_date,external_ref\n" + lse.ID + ",800,Loyer mars 2026,2026-03-05,REF-MAR-1\n"
	reqCSV := httptest.NewRequest(http.MethodPost, "/api/v1/payments/import-csv?entity_id="+entityID, strings.NewReader(csvBody))
	reqCSV.Header.Set("Content-Type", "text/csv")
	recCSV := httptest.NewRecorder()
	mux.ServeHTTP(recCSV, reqCSV)
	if recCSV.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/payments/import-csv status = %d, body = %s", recCSV.Code, recCSV.Body.String())
	}

	// 5. GET /api/v1/entities/{id}/negative-positions (2026-04 remains unpaid = 800_00)
	reqNeg := httptest.NewRequest(http.MethodGet, "/api/v1/entities/"+entityID+"/negative-positions", nil)
	recNeg := httptest.NewRecorder()
	mux.ServeHTTP(recNeg, reqNeg)
	if recNeg.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/entities/%s/negative-positions status = %d, body = %s", entityID, recNeg.Code, recNeg.Body.String())
	}
	var negPositions []application.NegativePositionDTO
	if err := json.Unmarshal(recNeg.Body.Bytes(), &negPositions); err != nil {
		t.Fatalf("decode negative positions: %v", err)
	}
	if len(negPositions) != 1 || negPositions[0].TotalUnpaidCents != money.AmountCents(800_00) {
		t.Fatalf("expected 1 negative position of 800_00, got %+v", negPositions)
	}

	// 6. GET /api/v1/entities/{id}/cash-journal
	reqJournal := httptest.NewRequest(http.MethodGet, "/api/v1/entities/"+entityID+"/cash-journal", nil)
	recJournal := httptest.NewRecorder()
	mux.ServeHTTP(recJournal, reqJournal)
	if recJournal.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/entities/%s/cash-journal status = %d, body = %s", entityID, recJournal.Code, recJournal.Body.String())
	}
	var journalEntries []application.CashJournalEntryDTO
	if err := json.Unmarshal(recJournal.Body.Bytes(), &journalEntries); err != nil || len(journalEntries) != 2 {
		t.Fatalf("expected 2 cash journal entries, got %d (err=%v)", len(journalEntries), err)
	}

	// 7. Verify all 5 MCP tools (record_cash_payment, record_bank_payment, list_negative_positions, import_payments_batch, record_cash_deposit)
	reg := mcpserver.NewRegistry(func(_ context.Context, _ string) (string, error) {
		return mcpUserID, nil
	})
	paymentmcp.RegisterTools(reg, paymentSvc)

	tools := reg.Tools()
	expectedTools := []string{
		"record_cash_payment",
		"record_bank_payment",
		"list_negative_positions",
		"import_payments_batch",
		"record_cash_deposit",
	}
	if len(tools) != len(expectedTools) {
		t.Fatalf("expected %d MCP tools, got %d", len(expectedTools), len(tools))
	}
	for i, want := range expectedTools {
		if tools[i].Name != want {
			t.Fatalf("expected tool[%d] = %q, got %q", i, want, tools[i].Name)
		}
	}

	// Call list_negative_positions, record_bank_payment, import_payments_batch, and record_cash_deposit via MCP JSON-RPC
	mcpListBody := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_negative_positions","arguments":{"legal_entity_id":"` + entityID + `"}}}`
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

	mcpImportBody := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"import_payments_batch","arguments":{"legal_entity_id":"` + entityID + `","csv_content":"lease_id,amount_eur,label,payment_date,external_ref\n` + lse.ID + `,800,Loyer mai 2026,2026-05-05,REF-MAY-MCP\n"}}}`
	reqMCPImport := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(mcpImportBody))
	reqMCPImport.Header.Set("Authorization", "Bearer valid")
	recMCPImport := httptest.NewRecorder()
	reg.ServeHTTP(recMCPImport, reqMCPImport)
	if recMCPImport.Code != http.StatusOK || !strings.Contains(recMCPImport.Body.String(), "REF-MAY-MCP") {
		t.Fatalf("unexpected MCP import_payments_batch response (%d): %s", recMCPImport.Code, recMCPImport.Body.String())
	}

	mcpDepositBody := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"record_cash_deposit","arguments":{"legal_entity_id":"` + entityID + `","amount_cents":50000,"label":"Remise MCP","entry_date":"2026-05-06"}}}`
	reqMCPDeposit := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(mcpDepositBody))
	reqMCPDeposit.Header.Set("Authorization", "Bearer valid")
	recMCPDeposit := httptest.NewRecorder()
	reg.ServeHTTP(recMCPDeposit, reqMCPDeposit)
	if recMCPDeposit.Code != http.StatusOK || !strings.Contains(recMCPDeposit.Body.String(), "Remise MCP") {
		t.Fatalf("unexpected MCP record_cash_deposit response (%d): %s", recMCPDeposit.Code, recMCPDeposit.Body.String())
	}
}

func TestAutomaticAccountingPosting_RentCallAndPayment(t *testing.T) {
	ctx := context.Background()
	paymentSvc, leaseSvc := newTestPaymentSuite()
	entityID := id.MustNewV7()
	lotID := id.MustNewV7()

	jeRepo := accountingmemory.NewJournalEntryRepo()
	abRepo := accountingmemory.NewAccountBalanceRepo()
	aarRepo := accountingmemory.NewAnnualAccountReportRepo()
	accUoW := accountingmemory.NewUnitOfWork(jeRepo, abRepo, aarRepo)
	accSvc := accountingapp.NewService(
		jeRepo,
		abRepo,
		aarRepo,
		accUoW,
		nil,
		nil,
		accountinginfra.NewFECExporter(),
		nil,
	)

	leaseSvc.WithAccountingRecorder(accSvc)
	paymentSvc.WithAccountingRecorder(accSvc)

	tenant, err := leaseSvc.CreateTenant(ctx, leaseapp.CreateTenantCmd{
		FullName: "SARL PHARMACIE CENTRALE",
		Email:    "contact@pharmacie.fr",
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	lse, err := leaseSvc.CreateLease(ctx, leaseapp.CreateLeaseCmd{
		LegalEntityID:      entityID,
		LotID:              lotID,
		TenantID:           tenant.ID,
		LeaseType:          "COMMERCIAL_3_6_9",
		RentNetCents:       money.AmountCents(2000_00),
		ChargesCents:       money.AmountCents(200_00),
		SubjectToVAT:       true,
		VATRateBasisPoints: 2000,
		DueDayOfMonth:      5,
		StartDate:          "2026-01-01",
	})
	if err != nil {
		t.Fatalf("create lease: %v", err)
	}

	calls, err := leaseSvc.GenerateMonthlyRentCalls(ctx, entityID, "2026-01")
	if err != nil || len(calls) != 1 {
		t.Fatalf("generate monthly rent calls: len=%d err=%v", len(calls), err)
	}

	// Total TTC = 2000 + 400 (TVA) + 200 (Charges) = 2600_00
	if _, err := paymentSvc.RecordBankPayment(ctx, application.RecordBankPaymentCmd{
		LeaseID:     lse.ID,
		AmountCents: money.AmountCents(2600_00),
		Method:      "BANK_TRANSFER",
		Label:       "Virement loyer janvier 2026",
		PaymentDate: "2026-01-06",
	}); err != nil {
		t.Fatalf("record bank payment: %v", err)
	}

	if _, err := paymentSvc.RecordCashDepositToBank(ctx, application.RecordCashDepositCmd{
		LegalEntityID: entityID,
		AmountCents:   money.AmountCents(500_00),
		Label:         "Remise espèces janvier 2026",
		EntryDate:     "2026-01-07",
	}); err != nil {
		t.Fatalf("record cash deposit: %v", err)
	}

	entries, err := accSvc.ListJournalEntries(ctx, entityID, 2026)
	if err != nil {
		t.Fatalf("list journal entries: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 automatic journal entries (VE rent call, BQ payment, OD cash deposit), got %d: %+v", len(entries), entries)
	}

	fecBytes, err := accSvc.ExportEntityFEC(ctx, entityID, 2026)
	if err != nil {
		t.Fatalf("export fec: %v", err)
	}
	fecStr := string(fecBytes)
	for _, code := range []string{"411000", "706200", "708000", "445710", "512000", "530000", "580000"} {
		if !strings.Contains(fecStr, code) {
			t.Fatalf("expected FEC export to contain PCG account %s, got:\n%s", code, fecStr)
		}
	}
}

func TestPayment_UUIDv7IdentitiesCashReceiptFormatAndValidation(t *testing.T) {
	paymentSvc, leaseSvc := newTestPaymentSuite()
	ctx := context.Background()
	entityID := id.MustNewV7()

	lse := seedTestLeaseWithRentCalls(t, leaseSvc, entityID, "2026-01")

	receipt, err := paymentSvc.RecordCashPayment(ctx, application.RecordCashPaymentCmd{
		LeaseID:     lse.ID,
		AmountCents: money.AmountCents(800_00),
		Label:       "Loyer espèces janvier 2026",
		PaymentDate: "2026-01-05",
	})
	if err != nil {
		t.Fatalf("RecordCashPayment: %v", err)
	}
	if !id.IsValidUUID(receipt.ID) || receipt.ID[14] != '7' {
		t.Fatalf("expected UUIDv7 PaymentReceiptDTO.ID, got %q", receipt.ID)
	}
	if !id.IsValidUUID(receipt.CashJournalEntryID) || receipt.CashJournalEntryID[14] != '7' {
		t.Fatalf("expected UUIDv7 CashJournalEntryID, got %q", receipt.CashJournalEntryID)
	}
	if len(receipt.Allocations) != 1 || !id.IsValidUUID(receipt.Allocations[0].ID) || receipt.Allocations[0].ID[14] != '7' {
		t.Fatalf("expected UUIDv7 RentAllocation.ID, got %+v", receipt.Allocations)
	}
	receiptPattern := regexp.MustCompile(`^CAISSE-\d{8}-[0-9A-F]{6}$`)
	if !receiptPattern.MatchString(receipt.ReceiptNumber) {
		t.Fatalf("expected auto-generated cash ReceiptNumber matching ^CAISSE-\\d{8}-[0-9A-F]{6}$, got %q", receipt.ReceiptNumber)
	}

	deposit, err := paymentSvc.RecordCashDepositToBank(ctx, application.RecordCashDepositCmd{
		LegalEntityID: entityID,
		AmountCents:   money.AmountCents(800_00),
		Label:         "Remise espèces",
		EntryDate:     "2026-01-06",
	})
	if err != nil {
		t.Fatalf("RecordCashDepositToBank: %v", err)
	}
	if !id.IsValidUUID(deposit.ID) || deposit.ID[14] != '7' {
		t.Fatalf("expected UUIDv7 CashJournalEntryDTO.ID, got %q", deposit.ID)
	}

	if _, err := paymentSvc.RecordCashPayment(ctx, application.RecordCashPaymentCmd{
		LeaseID:     "not-a-uuid",
		AmountCents: money.AmountCents(100_00),
	}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed LeaseID in RecordCashPayment, got %v", err)
	}
	if _, err := paymentSvc.RecordBankPayment(ctx, application.RecordBankPaymentCmd{
		LeaseID:     "not-a-uuid",
		AmountCents: money.AmountCents(100_00),
	}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed LeaseID in RecordBankPayment, got %v", err)
	}
	if _, err := paymentSvc.RecordCashDepositToBank(ctx, application.RecordCashDepositCmd{
		LegalEntityID: "not-a-uuid",
		AmountCents:   money.AmountCents(100_00),
	}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed LegalEntityID in RecordCashDepositToBank, got %v", err)
	}
	if _, err := paymentSvc.ImportPaymentsCSV(ctx, "not-a-uuid", strings.NewReader("lease_id,amount_eur\n")); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed entityID in ImportPaymentsCSV, got %v", err)
	}
	if _, err := paymentSvc.ListNegativePositions(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed entityID in ListNegativePositions, got %v", err)
	}
	if _, err := paymentSvc.ListCashJournalByEntity(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed entityID in ListCashJournalByEntity, got %v", err)
	}
}
