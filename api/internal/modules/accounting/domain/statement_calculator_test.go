package domain_test

import (
	"context"
	"errors"
	"testing"

	accountingapp "github.com/yannamsellem/gestion-immo/internal/modules/accounting/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/infrastructure"
	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/infrastructure/memory"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

func TestCalculateStatement_DisabledLineFlipsResultAndZeroPrevYearReturnsNS(t *testing.T) {
	calc := domain.NewStatementCalculator()
	lines := []domain.AccountBalanceLine{
		{LineID: "rev_locaux", AccountCode: "706100", Label: "LOCATIONS LOCAUX", Section: "OPERATING_REVENUE", NetNCents: money.AmountCents(133607_00), NetNMinus1Cents: money.AmountCents(132604_00)},
		{LineID: "rev_comm", AccountCode: "706200", Label: "LOYERS COMMERCIAUX", Section: "OPERATING_REVENUE", NetNCents: money.AmountCents(22952_00), NetNMinus1Cents: money.AmountCents(15301_00)},
		{LineID: "chg_loc", AccountCode: "614000", Label: "CHARGES LOCATIVES", Section: "EXTERNAL_EXPENSES", NetNCents: money.AmountCents(45001_00), NetNMinus1Cents: money.AmountCents(77663_00)},
		{LineID: "chg_tf", AccountCode: "635120", Label: "TAXES FONCIERES", Section: "TAXES_AND_DUTIES", NetNCents: money.AmountCents(56916_00), NetNMinus1Cents: money.AmountCents(55767_00)},
		{LineID: "chg_fourn_admin", AccountCode: "606400", Label: "FOURNITURES ADMINISTRATIVES", Section: "EXTERNAL_EXPENSES", NetNCents: money.AmountCents(50000_00), NetNMinus1Cents: 0},
	}

	// 1. All lines active: Total Revenue = 156559, Total Expenses = 151917 -> Net Income = +4642_00
	full := calc.ComputeStatements(lines, nil)
	if full.TotalOperatingRevenueNCents != 156559_00 || full.NetIncomeNCents != 4642_00 {
		t.Fatalf("unexpected full totals: %+v", full)
	}
	var adminLine domain.ComputedLineDTO
	for _, l := range full.Lines {
		if l.LineID == "chg_fourn_admin" {
			adminLine = l
		}
	}
	if !adminLine.VariationNotSignificant || domain.FormatVariationPercent(adminLine.NetNCents, adminLine.NetNMinus1Cents) != "N/S" {
		t.Fatalf("expected VariationNotSignificant=true and N/S when N-1 is 0, got %+v", adminLine)
	}

	// 2. Disable "rev_comm" (22952_00) -> Total Revenue drops to 133607_00, Net Income flips to deficit -18310_00
	disabled := map[string]bool{"rev_comm": true}
	filtered := calc.ComputeStatements(lines, disabled)
	if filtered.TotalOperatingRevenueNCents != 133607_00 {
		t.Fatalf("expected TotalOperatingRevenueNCents 133607_00, got %d", filtered.TotalOperatingRevenueNCents)
	}
	if filtered.NetIncomeNCents != -18310_00 || filtered.BalanceSheetNetIncomeNCents != -18310_00 {
		t.Fatalf("expected NetIncomeNCents and BalanceSheetNetIncomeNCents -18310_00, got %d / %d", filtered.NetIncomeNCents, filtered.BalanceSheetNetIncomeNCents)
	}
}

func TestCalculateStatement_SIGAndBilanPassifCascade(t *testing.T) {
	calc := domain.NewStatementCalculator()
	lines := []domain.AccountBalanceLine{
		{LineID: "rev_706", AccountCode: "706100", Label: "LOCATIONS", Section: "OPERATING_REVENUE", NetNCents: 200000_00, NetNMinus1Cents: 180000_00},
		{LineID: "chg_614", AccountCode: "614000", Label: "CHARGES COPRO", Section: "EXTERNAL_EXPENSES", NetNCents: 50000_00, NetNMinus1Cents: 45000_00},
		{LineID: "chg_635", AccountCode: "635120", Label: "TAXES FONCIERES", Section: "TAXES_AND_DUTIES", NetNCents: 30000_00, NetNMinus1Cents: 28000_00},
		{LineID: "chg_641", AccountCode: "641100", Label: "REMUNERATION GERANCE", Section: "PERSONNEL_EXPENSES", NetNCents: 20000_00, NetNMinus1Cents: 20000_00},
		{LineID: "chg_681", AccountCode: "681120", Label: "DOTATIONS AMORTISSEMENTS", Section: "DEPRECIATION", NetNCents: 40000_00, NetNMinus1Cents: 40000_00},
		{LineID: "chg_661", AccountCode: "661100", Label: "INTERETS EMPRUNTS", Section: "FINANCIAL_EXPENSES", NetNCents: 10000_00, NetNMinus1Cents: 12000_00},
		{LineID: "cap_101", AccountCode: "101000", Label: "CAPITAL SOCIAL", Section: "EQUITY", NetNCents: 256000_00, NetNMinus1Cents: 256000_00},
		{LineID: "det_164", AccountCode: "164000", Label: "EMPRUNTS BANCAIRES", Section: "LIABILITIES", NetNCents: 400000_00, NetNMinus1Cents: 430000_00},
	}

	res := calc.ComputeStatements(lines, nil)
	if res.ValueAddedNCents != 150000_00 {
		t.Fatalf("expected ValueAddedNCents 150000_00, got %d", res.ValueAddedNCents)
	}
	if res.GrossOperatingSurplusNCents != 100000_00 {
		t.Fatalf("expected GrossOperatingSurplusNCents 100000_00, got %d", res.GrossOperatingSurplusNCents)
	}
	if res.OperatingIncomeNCents != 60000_00 {
		t.Fatalf("expected OperatingIncomeNCents 60000_00, got %d", res.OperatingIncomeNCents)
	}
	if res.NetIncomeNCents != 50000_00 {
		t.Fatalf("expected NetIncomeNCents 50000_00, got %d", res.NetIncomeNCents)
	}
	if res.SelfFinancingCapacityNCents != 90000_00 {
		t.Fatalf("expected SelfFinancingCapacityNCents 90000_00, got %d", res.SelfFinancingCapacityNCents)
	}
	if res.TotalEquityNCents != 306000_00 {
		t.Fatalf("expected TotalEquityNCents 306000_00, got %d", res.TotalEquityNCents)
	}
}

func TestRecordJournalEntry_CreditingDebitNormalAccountDecreasesBalance(t *testing.T) {
	ctx := context.Background()
	journalEntries := memory.NewJournalEntryRepo()
	accountBalances := memory.NewAccountBalanceRepo()
	annualReports := memory.NewAnnualAccountReportRepo()
	uow := memory.NewUnitOfWork(journalEntries, accountBalances, annualReports)
	pdfEngine := templating.NewInMemoryPdfEngine()
	svc := accountingapp.NewService(
		journalEntries,
		accountBalances,
		annualReports,
		uow,
		r2.NewInMemoryStorage(),
		pdfEngine,
		infrastructure.NewFECExporter(),
		infrastructure.NewAnnualAccountsPDFRenderer(pdfEngine),
	)

	entityID := id.MustNewV7()
	// Validate single-line JournalEntry fails domain validation
	singleLineEntry := domain.JournalEntry{
		LegalEntityID: entityID,
		Lines: []domain.JournalLine{
			{AccountCode: "512000", DebitCents: 0, CreditCents: 0},
		},
	}
	if !errors.Is(singleLineEntry.Validate(), domain.ErrValidation) {
		t.Fatalf("expected ErrValidation for < 2 journal lines")
	}

	// 1. Incoming rent receipt: Debit 512000 Banque 10 000,00 € / Credit 706100 Locations 10 000,00 €
	_, err := svc.RecordJournalEntry(ctx, accountingapp.RecordJournalEntryCmd{
		LegalEntityID: entityID,
		FiscalYear:    2024,
		JournalCode:   "BQ",
		Label:         "Encaissement loyer",
		Lines: []accountingapp.JournalLineInput{
			{LineID: "banque_512", AccountCode: "512000", AccountLabel: "Banque", DebitCents: 10000_00},
			{LineID: "loyers_706", AccountCode: "706100", AccountLabel: "Locations", CreditCents: 10000_00},
		},
	})
	if err != nil {
		t.Fatalf("record incoming journal entry: %v", err)
	}

	// 2. Outgoing supplier payment: Debit 614000 Charges 3 000,00 € / Credit 512000 Banque 3 000,00 €
	_, err = svc.RecordJournalEntry(ctx, accountingapp.RecordJournalEntryCmd{
		LegalEntityID: entityID,
		FiscalYear:    2024,
		JournalCode:   "BQ",
		Label:         "Règlement charges locatives",
		Lines: []accountingapp.JournalLineInput{
			{LineID: "charges_614", AccountCode: "614000", AccountLabel: "Charges locatives", DebitCents: 3000_00},
			{LineID: "banque_512", AccountCode: "512000", AccountLabel: "Banque", CreditCents: 3000_00},
		},
	})
	if err != nil {
		t.Fatalf("record outgoing journal entry: %v", err)
	}

	// 3. Rent credit note (avoir): Debit 706100 Locations 1 000,00 € / Credit 512000 Banque 1 000,00 €
	_, err = svc.RecordJournalEntry(ctx, accountingapp.RecordJournalEntryCmd{
		LegalEntityID: entityID,
		FiscalYear:    2024,
		JournalCode:   "BQ",
		Label:         "Avoir loyer remboursé",
		Lines: []accountingapp.JournalLineInput{
			{LineID: "loyers_706", AccountCode: "706100", AccountLabel: "Locations", DebitCents: 1000_00},
			{LineID: "banque_512", AccountCode: "512000", AccountLabel: "Banque", CreditCents: 1000_00},
		},
	})
	if err != nil {
		t.Fatalf("record credit note journal entry: %v", err)
	}

	stmt, err := svc.PreviewFinancialStatements(ctx, entityID, 2024, nil)
	if err != nil {
		t.Fatalf("preview financial statements: %v", err)
	}

	// 512000 Banque (CURRENT_ASSETS) must be 10 000 - 3 000 - 1 000 = 6 000,00 €
	if stmt.CurrentAssetsNetNCents != 6000_00 {
		t.Fatalf("expected CurrentAssetsNetNCents (512000) to decrease to 6000_00, got %d", stmt.CurrentAssetsNetNCents)
	}
	// 706100 Locations (OPERATING_REVENUE) must be 10 000 - 1 000 = 9 000,00 €
	if stmt.TotalOperatingRevenueNCents != 9000_00 {
		t.Fatalf("expected TotalOperatingRevenueNCents (706100) to decrease to 9000_00, got %d", stmt.TotalOperatingRevenueNCents)
	}
	// Net Income = 9 000 - 3 000 = 6 000,00 €
	if stmt.NetIncomeNCents != 6000_00 {
		t.Fatalf("expected NetIncomeNCents 6000_00, got %d", stmt.NetIncomeNCents)
	}
}

