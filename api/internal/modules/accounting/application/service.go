package application

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/infrastructure"
	portfolioapp "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/application"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

// PortfolioPort defines the portfolio application operations consumed by the accounting bounded context.
type PortfolioPort interface {
	GetEntityPortfolioSummary(ctx context.Context, entityID string) (portfolioapp.EntityPortfolioSummaryDTO, error)
}

// JournalLineInput carries a single debit or credit line when recording a PCG journal entry.
type JournalLineInput struct {
	LineID                     string            `json:"line_id,omitempty"`
	AccountCode                string            `json:"account_code"`
	AccountLabel               string            `json:"account_label,omitempty"`
	SubAccountCode             string            `json:"sub_account_code,omitempty"`
	SubAccountLabel            string            `json:"sub_account_label,omitempty"`
	Section                    string            `json:"section,omitempty"`
	Label                      string            `json:"label,omitempty"`
	DebitCents                 money.AmountCents `json:"debit_cents,omitempty"`
	CreditCents                money.AmountCents `json:"credit_cents,omitempty"`
	NetNMinus1Cents            money.AmountCents `json:"net_n_minus_1_cents,omitempty"`
	ReconciliationCode         string            `json:"reconciliation_code,omitempty"`
	ReconciliationDate         string            `json:"reconciliation_date,omitempty"`
	ForeignCurrencyAmountCents money.AmountCents `json:"foreign_currency_amount_cents,omitempty"`
	CurrencyCode               string            `json:"currency_code,omitempty"`
}

// RecordExpenseInvoiceCmd carries the input for recording a supplier expense invoice in the AC (Purchases) journal.
type RecordExpenseInvoiceCmd struct {
	LegalEntityID      string            `json:"legal_entity_id" format:"uuid"`
	SupplierName       string            `json:"supplier_name,omitempty"`
	ExpenseAccountCode string            `json:"expense_account_code"`
	Label              string            `json:"label,omitempty"`
	InvoiceRef         string            `json:"invoice_ref,omitempty"`
	InvoiceDate        string            `json:"invoice_date,omitempty"`
	AmountHTCents      money.AmountCents `json:"amount_ht_cents"`
	VATCents           money.AmountCents `json:"vat_cents,omitempty"`
}

// RecordJournalEntryCmd carries the input for recording a balanced double-entry PCG journal entry.
type RecordJournalEntryCmd struct {
	LegalEntityID  string             `json:"legal_entity_id,omitempty"`
	FiscalYear     int                `json:"fiscal_year,omitempty"`
	JournalCode    string             `json:"journal_code,omitempty"`
	JournalLabel   string             `json:"journal_label,omitempty"`
	EntryNumber    string             `json:"entry_number,omitempty"`
	EntryDate      string             `json:"entry_date,omitempty"`
	DocumentRef    string             `json:"document_ref,omitempty"`
	DocumentDate   string             `json:"document_date,omitempty"`
	Label          string             `json:"label"`
	ValidationDate string             `json:"validation_date,omitempty"`
	Lines          []JournalLineInput `json:"lines"`
}

// JournalLineDTO is an immutable projection of a domain.JournalLine.
type JournalLineDTO struct {
	ID              string            `json:"id" format:"uuid"`
	LineID          string            `json:"line_id,omitempty"`
	AccountCode     string            `json:"account_code"`
	AccountLabel    string            `json:"account_label"`
	SubAccountCode  string            `json:"sub_account_code,omitempty"`
	SubAccountLabel string            `json:"sub_account_label,omitempty"`
	Section         string            `json:"section"`
	Label           string            `json:"label"`
	DebitCents      money.AmountCents `json:"debit_cents"`
	CreditCents     money.AmountCents `json:"credit_cents"`
	NetNMinus1Cents money.AmountCents `json:"net_n_minus_1_cents,omitempty"`
}

// JournalEntryDTO is an immutable projection of a domain.JournalEntry.
type JournalEntryDTO struct {
	ID               string            `json:"id" format:"uuid"`
	LegalEntityID    string            `json:"legal_entity_id" format:"uuid"`
	FiscalYear       int               `json:"fiscal_year"`
	JournalCode      string            `json:"journal_code"`
	JournalLabel     string            `json:"journal_label"`
	EntryNumber      string            `json:"entry_number"`
	EntryDate        string            `json:"entry_date"`
	DocumentRef      string            `json:"document_ref"`
	DocumentDate     string            `json:"document_date"`
	Label            string            `json:"label"`
	ValidationDate   string            `json:"validation_date"`
	TotalDebitCents  money.AmountCents `json:"total_debit_cents"`
	TotalCreditCents money.AmountCents `json:"total_credit_cents"`
	Lines            []JournalLineDTO  `json:"lines"`
	CreatedAt        time.Time         `json:"created_at"`
}

// AnnualAccountReportDTO represents a generated Annual Accounts and Tax Return PDF stored on Cloudflare R2.
type AnnualAccountReportDTO struct {
	ID              string            `json:"id" format:"uuid"`
	LegalEntityID   string            `json:"legal_entity_id" format:"uuid"`
	FiscalYear      int               `json:"fiscal_year"`
	Category        string            `json:"category"`
	FileName        string            `json:"file_name"`
	ContentType     string            `json:"content_type"`
	R2Key           string            `json:"r2_key"`
	SizeBytes       int64             `json:"size_bytes"`
	DownloadURL     string            `json:"download_url"`
	DisabledLineIDs []string          `json:"disabled_line_ids"`
	NetIncomeNCents money.AmountCents `json:"net_income_n_cents"`
	CreatedAt       time.Time         `json:"created_at"`
}

// StoredFileDTO is an alias for AnnualAccountReportDTO.
type StoredFileDTO = AnnualAccountReportDTO

// Service orchestrates PCG double-entry accounting, dynamic financial statement previews,
// Annual Accounts and Tax Return PDF generation, and normative DGFiP FEC exports.
type Service struct {
	journalEntries         domain.JournalEntryRepository
	accountBalances        domain.AccountBalanceRepository
	annualReports          domain.AnnualAccountReportRepository
	uow                    domain.UnitOfWork
	calculator             *domain.StatementCalculator
	storage                r2.ObjectStorage
	pdfEngine              templating.PdfEngine
	fecExporter            domain.FECExporterPort
	annualAccountsRenderer domain.AnnualAccountsPDFRendererPort
	portfolioSvc           PortfolioPort
}

// NewService constructs an accounting application Service via constructor injection.
func NewService(
	journalEntries domain.JournalEntryRepository,
	accountBalances domain.AccountBalanceRepository,
	annualReports domain.AnnualAccountReportRepository,
	uow domain.UnitOfWork,
	storage r2.ObjectStorage,
	pdfEngine templating.PdfEngine,
	fecExporter domain.FECExporterPort,
	annualAccountsRenderer domain.AnnualAccountsPDFRendererPort,
	portfolioPorts ...PortfolioPort,
) *Service {
	if fecExporter == nil {
		fecExporter = infrastructure.NewFECExporter()
	}
	if annualAccountsRenderer == nil {
		annualAccountsRenderer = infrastructure.NewAnnualAccountsPDFRenderer(pdfEngine)
	}
	var portfolioSvc PortfolioPort
	if len(portfolioPorts) > 0 {
		portfolioSvc = portfolioPorts[0]
	}
	return &Service{
		journalEntries:         journalEntries,
		accountBalances:        accountBalances,
		annualReports:          annualReports,
		uow:                    uow,
		calculator:             domain.NewStatementCalculator(),
		storage:                storage,
		pdfEngine:              pdfEngine,
		fecExporter:            fecExporter,
		annualAccountsRenderer: annualAccountsRenderer,
		portfolioSvc:           portfolioSvc,
	}
}

// WithPortfolioPort attaches a live PortfolioPort to the accounting Service for resolving legal entity and shareholder metadata.
func (s *Service) WithPortfolioPort(port PortfolioPort) *Service {
	s.portfolioSvc = port
	return s
}

// RecordRentCallEntry posts a balanced PCG VE (Ventes) journal entry for a generated rent call:
// debiting 411000 (totalDueCents, sub-account tenantID) and crediting 706100 (when vatCents == 0)
// or 706200 (when vatCents > 0) for rentNetCents, 708000 for chargesCents (if > 0), and 445710 for vatCents (if > 0).
func (s *Service) RecordRentCallEntry(
	ctx context.Context,
	entityID, rentCallID, leaseID, tenantID, periodMonth string,
	dueDate time.Time,
	rentNetCents, vatCents, chargesCents, totalDueCents money.AmountCents,
) error {
	if totalDueCents <= 0 {
		return nil
	}

	label := fmt.Sprintf("Appel de loyer %s (%s)", strings.TrimSpace(periodMonth), strings.TrimSpace(leaseID))
	var dateStr string
	var fiscalYear int
	if !dueDate.IsZero() {
		dateStr = dueDate.UTC().Format("2006-01-02")
		fiscalYear = dueDate.UTC().Year()
	}

	lines := []JournalLineInput{
		{
			AccountCode:    "411000",
			AccountLabel:   "CLIENTS LOCATAIRES",
			SubAccountCode: strings.TrimSpace(tenantID),
			Label:          label,
			DebitCents:     totalDueCents,
		},
	}

	if rentNetCents > 0 {
		revCode := "706100"
		revLabel := "LOCATIONS LOCAUX"
		if vatCents > 0 {
			revCode = "706200"
			revLabel = "LOYERS COMMERCIAUX"
		}
		lines = append(lines, JournalLineInput{
			AccountCode:  revCode,
			AccountLabel: revLabel,
			Label:        label,
			CreditCents:  rentNetCents,
		})
	}

	if chargesCents > 0 {
		lines = append(lines, JournalLineInput{
			AccountCode:  "708000",
			AccountLabel: "PRODUITS DES ACTIVITES ANNEXES - CHARGES LOCATIVES",
			Label:        label,
			CreditCents:  chargesCents,
		})
	}

	if vatCents > 0 {
		lines = append(lines, JournalLineInput{
			AccountCode:  "445710",
			AccountLabel: "TVA COLLECTEE",
			Label:        label,
			CreditCents:  vatCents,
		})
	}

	_, err := s.RecordJournalEntry(ctx, RecordJournalEntryCmd{
		LegalEntityID:  entityID,
		FiscalYear:     fiscalYear,
		JournalCode:    "VE",
		EntryDate:      dateStr,
		DocumentRef:    strings.TrimSpace(rentCallID),
		DocumentDate:   dateStr,
		ValidationDate: dateStr,
		Label:          label,
		Lines:          lines,
	})
	return err
}

// RecordPaymentCollectionEntry posts a balanced PCG CA (530000 Caisse when method == "CASH")
// or BQ (512000 Banque otherwise) journal entry debiting 530000/512000 and crediting 411000 (amountCents).
func (s *Service) RecordPaymentCollectionEntry(
	ctx context.Context,
	entityID, paymentID, tenantID, method, label string,
	paymentDate time.Time,
	amountCents money.AmountCents,
) error {
	if amountCents <= 0 {
		return nil
	}

	journalCode := "BQ"
	treasuryCode := "512000"
	treasuryLabel := "BANQUE"
	if strings.EqualFold(strings.TrimSpace(method), "CASH") {
		journalCode = "CA"
		treasuryCode = "530000"
		treasuryLabel = "CAISSE"
	}

	entryLabel := strings.TrimSpace(label)
	if entryLabel == "" {
		entryLabel = "Encaissement loyer"
	}

	var dateStr string
	var fiscalYear int
	if !paymentDate.IsZero() {
		dateStr = paymentDate.UTC().Format("2006-01-02")
		fiscalYear = paymentDate.UTC().Year()
	}

	_, err := s.RecordJournalEntry(ctx, RecordJournalEntryCmd{
		LegalEntityID:  entityID,
		FiscalYear:     fiscalYear,
		JournalCode:    journalCode,
		EntryDate:      dateStr,
		DocumentRef:    strings.TrimSpace(paymentID),
		DocumentDate:   dateStr,
		ValidationDate: dateStr,
		Label:          entryLabel,
		Lines: []JournalLineInput{
			{
				AccountCode:  treasuryCode,
				AccountLabel: treasuryLabel,
				Label:        entryLabel,
				DebitCents:   amountCents,
			},
			{
				AccountCode:    "411000",
				AccountLabel:   "CLIENTS LOCATAIRES",
				SubAccountCode: strings.TrimSpace(tenantID),
				Label:          entryLabel,
				CreditCents:    amountCents,
			},
		},
	})
	return err
}

// RecordCashDepositEntry posts a balanced PCG OD entry debiting 512000 and crediting 530000 (via 580000 internal transfer).
func (s *Service) RecordCashDepositEntry(
	ctx context.Context,
	entityID, depositID, label string,
	depositDate time.Time,
	amountCents money.AmountCents,
) error {
	if amountCents <= 0 {
		return nil
	}

	entryLabel := strings.TrimSpace(label)
	if entryLabel == "" {
		entryLabel = "Remise d'espèces en banque"
	}

	var dateStr string
	var fiscalYear int
	if !depositDate.IsZero() {
		dateStr = depositDate.UTC().Format("2006-01-02")
		fiscalYear = depositDate.UTC().Year()
	}

	_, err := s.RecordJournalEntry(ctx, RecordJournalEntryCmd{
		LegalEntityID:  entityID,
		FiscalYear:     fiscalYear,
		JournalCode:    "OD",
		EntryDate:      dateStr,
		DocumentRef:    strings.TrimSpace(depositID),
		DocumentDate:   dateStr,
		ValidationDate: dateStr,
		Label:          entryLabel,
		Lines: []JournalLineInput{
			{
				AccountCode:  "580000",
				AccountLabel: "VIREMENTS INTERNES",
				Label:        entryLabel,
				DebitCents:   amountCents,
			},
			{
				AccountCode:  "530000",
				AccountLabel: "CAISSE",
				Label:        entryLabel,
				CreditCents:  amountCents,
			},
			{
				AccountCode:  "512000",
				AccountLabel: "BANQUE",
				Label:        entryLabel,
				DebitCents:   amountCents,
			},
			{
				AccountCode:  "580000",
				AccountLabel: "VIREMENTS INTERNES",
				Label:        entryLabel,
				CreditCents:  amountCents,
			},
		},
	})
	return err
}

// RecordExpenseInvoice posts a balanced PCG AC (Achats) journal entry debiting expense_account_code
// and 445660 (when vat_cents > 0) and crediting 401000 (Fournisseurs).
func (s *Service) RecordExpenseInvoice(ctx context.Context, cmd RecordExpenseInvoiceCmd) (JournalEntryDTO, error) {
	entityID := strings.TrimSpace(cmd.LegalEntityID)
	if !id.IsValidUUID(entityID) {
		return JournalEntryDTO{}, fmt.Errorf("%w: legal_entity_id must be a valid UUID", domain.ErrValidation)
	}
	if cmd.AmountHTCents <= 0 || cmd.VATCents < 0 {
		return JournalEntryDTO{}, fmt.Errorf("%w: amount_ht_cents must be positive and vat_cents non-negative", domain.ErrValidation)
	}

	expenseAcct := strings.TrimSpace(cmd.ExpenseAccountCode)
	if expenseAcct == "" {
		expenseAcct = "606400"
	}

	supplier := strings.TrimSpace(cmd.SupplierName)
	label := strings.TrimSpace(cmd.Label)
	if label == "" {
		if supplier != "" {
			label = "Facture " + supplier
		} else {
			label = "Facture fournisseur"
		}
	}

	totalTTC := cmd.AmountHTCents.Add(cmd.VATCents)
	lines := []JournalLineInput{
		{
			AccountCode:  expenseAcct,
			AccountLabel: label,
			Label:        label,
			DebitCents:   cmd.AmountHTCents,
		},
	}
	if cmd.VATCents > 0 {
		lines = append(lines, JournalLineInput{
			AccountCode:  "445660",
			AccountLabel: "TVA DEDUCTIBLE SUR AUTRES BIENS ET SERVICES",
			Label:        label,
			DebitCents:   cmd.VATCents,
		})
	}
	lines = append(lines, JournalLineInput{
		AccountCode:     "401000",
		AccountLabel:    "FOURNISSEURS",
		SubAccountCode:  supplier,
		SubAccountLabel: supplier,
		Label:           label,
		CreditCents:     totalTTC,
	})

	return s.RecordJournalEntry(ctx, RecordJournalEntryCmd{
		LegalEntityID:  entityID,
		JournalCode:    "AC",
		EntryDate:      strings.TrimSpace(cmd.InvoiceDate),
		DocumentRef:    strings.TrimSpace(cmd.InvoiceRef),
		DocumentDate:   strings.TrimSpace(cmd.InvoiceDate),
		ValidationDate: strings.TrimSpace(cmd.InvoiceDate),
		Label:          label,
		Lines:          lines,
	})
}

// RecordJournalEntry validates and persists a balanced double-entry PCG journal entry
// and updates the entity's account balance lines for the fiscal year inside a single UnitOfWork transaction.
func (s *Service) RecordJournalEntry(ctx context.Context, cmd RecordJournalEntryCmd) (JournalEntryDTO, error) {
	entityID := strings.TrimSpace(cmd.LegalEntityID)
	if !id.IsValidUUID(entityID) {
		return JournalEntryDTO{}, fmt.Errorf("%w: legal_entity_id must be a valid UUID", domain.ErrValidation)
	}
	label := strings.TrimSpace(cmd.Label)
	if label == "" {
		return JournalEntryDTO{}, fmt.Errorf("%w: journal entry label is required", domain.ErrValidation)
	}
	if len(cmd.Lines) < 2 {
		return JournalEntryDTO{}, fmt.Errorf("%w: at least two journal lines are required for double-entry accounting", domain.ErrValidation)
	}

	entryDate, err := parseDateOrToday(cmd.EntryDate)
	if err != nil {
		return JournalEntryDTO{}, fmt.Errorf("%w: invalid entry_date %q: %v", domain.ErrValidation, cmd.EntryDate, err)
	}
	documentDate := entryDate
	if strings.TrimSpace(cmd.DocumentDate) != "" {
		documentDate, err = parseDateOrToday(cmd.DocumentDate)
		if err != nil {
			return JournalEntryDTO{}, fmt.Errorf("%w: invalid document_date %q: %v", domain.ErrValidation, cmd.DocumentDate, err)
		}
	}
	validationDate := entryDate
	if strings.TrimSpace(cmd.ValidationDate) != "" {
		validationDate, err = parseDateOrToday(cmd.ValidationDate)
		if err != nil {
			return JournalEntryDTO{}, fmt.Errorf("%w: invalid validation_date %q: %v", domain.ErrValidation, cmd.ValidationDate, err)
		}
	}

	fiscalYear := cmd.FiscalYear
	if fiscalYear <= 0 {
		fiscalYear = entryDate.Year()
	}

	journalCode := strings.ToUpper(strings.TrimSpace(cmd.JournalCode))
	if journalCode == "" {
		journalCode = "OD"
	}
	journalLabel := strings.TrimSpace(cmd.JournalLabel)
	if journalLabel == "" {
		journalLabel = defaultJournalLabel(journalCode)
	}

	entryID, err := id.NewV7()
	if err != nil {
		return JournalEntryDTO{}, err
	}
	entryNumber := strings.TrimSpace(cmd.EntryNumber)
	if entryNumber == "" {
		entryNumber = strings.ToUpper(entryID)
	}
	documentRef := strings.TrimSpace(cmd.DocumentRef)
	if documentRef == "" {
		documentRef = entryNumber
	}

	now := time.Now().UTC()
	domainLines := make([]domain.JournalLine, 0, len(cmd.Lines))
	for i, in := range cmd.Lines {
		acctCode := strings.TrimSpace(in.AccountCode)
		if acctCode == "" {
			return JournalEntryDTO{}, fmt.Errorf("%w: account_code is required on line %d", domain.ErrValidation, i+1)
		}
		lineID, lErr := id.NewV7()
		if lErr != nil {
			return JournalEntryDTO{}, lErr
		}
		logicalLineID := strings.TrimSpace(in.LineID)
		if logicalLineID == "" {
			logicalLineID = "acct_" + acctCode
		}
		acctLabel := strings.TrimSpace(in.AccountLabel)
		if acctLabel == "" {
			acctLabel = strings.TrimSpace(in.Label)
		}
		if acctLabel == "" {
			acctLabel = label
		}
		lineLabel := strings.TrimSpace(in.Label)
		if lineLabel == "" {
			lineLabel = label
		}
		sec := strings.ToUpper(strings.TrimSpace(in.Section))
		if sec == "" {
			sec = string(domain.InferSectionFromAccountCode(acctCode))
		}
		currencyCode := strings.TrimSpace(in.CurrencyCode)
		if currencyCode == "" {
			currencyCode = "EUR"
		}

		domainLines = append(domainLines, domain.JournalLine{
			ID:                         lineID,
			JournalEntryID:             entryID,
			LegalEntityID:              entityID,
			FiscalYear:                 fiscalYear,
			LineID:                     logicalLineID,
			AccountCode:                acctCode,
			AccountLabel:               acctLabel,
			SubAccountCode:             strings.TrimSpace(in.SubAccountCode),
			SubAccountLabel:            strings.TrimSpace(in.SubAccountLabel),
			Section:                    sec,
			Label:                      lineLabel,
			DebitCents:                 in.DebitCents,
			CreditCents:                in.CreditCents,
			NetNMinus1Cents:            in.NetNMinus1Cents,
			ReconciliationCode:         strings.TrimSpace(in.ReconciliationCode),
			ReconciliationDate:         strings.TrimSpace(in.ReconciliationDate),
			ForeignCurrencyAmountCents: in.ForeignCurrencyAmountCents,
			CurrencyCode:               currencyCode,
			CreatedAt:                  now,
		})
	}

	entry := domain.JournalEntry{
		ID:             entryID,
		LegalEntityID:  entityID,
		FiscalYear:     fiscalYear,
		JournalCode:    journalCode,
		JournalLabel:   journalLabel,
		EntryNumber:    entryNumber,
		EntryDate:      entryDate,
		DocumentRef:    documentRef,
		DocumentDate:   documentDate,
		Label:          label,
		ValidationDate: validationDate,
		Lines:          domainLines,
		CreatedAt:      now,
	}

	if err := entry.Validate(); err != nil {
		return JournalEntryDTO{}, err
	}

	err = s.uow.WithinTransaction(ctx, func(txCtx context.Context, repos domain.Repositories) error {
		if err := repos.JournalEntries.SaveJournalEntry(txCtx, entry); err != nil {
			return err
		}
		return s.syncBalanceLinesFromEntry(txCtx, repos.AccountBalances, entry)
	})
	if err != nil {
		return JournalEntryDTO{}, err
	}

	return toJournalEntryDTO(entry), nil
}

// UpsertAccountBalanceLines directly seeds or updates AccountBalanceLines for an entity and fiscal year.
func (s *Service) UpsertAccountBalanceLines(ctx context.Context, entityID string, fiscalYear int, lines []domain.AccountBalanceLine) error {
	trimmed := strings.TrimSpace(entityID)
	if !id.IsValidUUID(trimmed) {
		return fmt.Errorf("%w: legal_entity_id must be a valid UUID", domain.ErrValidation)
	}
	for _, l := range lines {
		if trimmedID := strings.TrimSpace(l.ID); trimmedID != "" && !id.IsValidUUID(trimmedID) {
			return fmt.Errorf("%w: account balance line id must be a valid UUID", domain.ErrValidation)
		}
	}
	if fiscalYear <= 0 {
		fiscalYear = time.Now().UTC().Year()
	}
	return s.accountBalances.UpsertAccountBalanceLines(ctx, trimmed, fiscalYear, lines)
}

// ListJournalEntries returns all recorded JournalEntries for an entity and fiscal year.
func (s *Service) ListJournalEntries(ctx context.Context, entityID string, fiscalYear int) ([]JournalEntryDTO, error) {
	trimmed := strings.TrimSpace(entityID)
	if !id.IsValidUUID(trimmed) {
		return nil, fmt.Errorf("%w: legal_entity_id must be a valid UUID", domain.ErrValidation)
	}
	entries, err := s.journalEntries.ListJournalEntries(ctx, trimmed, fiscalYear)
	if err != nil {
		return nil, err
	}
	out := make([]JournalEntryDTO, 0, len(entries))
	for _, e := range entries {
		out = append(out, toJournalEntryDTO(e))
	}
	return out, nil
}

// PreviewFinancialStatements computes the interactive Bilan, Compte de Résultat, and SIG for an entity and fiscal year,
// dynamically excluding any line IDs in disabledLineIDs without mutating the underlying general ledger.
func (s *Service) PreviewFinancialStatements(
	ctx context.Context,
	entityID string,
	fiscalYear int,
	disabledLineIDs []string,
) (domain.ComputedFinancialStatementsDTO, error) {
	trimmedEntity := strings.TrimSpace(entityID)
	if !id.IsValidUUID(trimmedEntity) {
		return domain.ComputedFinancialStatementsDTO{}, fmt.Errorf("%w: legal_entity_id must be a valid UUID", domain.ErrValidation)
	}
	if fiscalYear <= 0 {
		fiscalYear = time.Now().UTC().Year()
	}

	rawLines, err := s.loadEffectiveBalanceLines(ctx, trimmedEntity, fiscalYear)
	if err != nil {
		return domain.ComputedFinancialStatementsDTO{}, err
	}

	disabledMap := make(map[string]bool, len(disabledLineIDs))
	for _, lineSlug := range disabledLineIDs {
		trimmedID := strings.TrimSpace(lineSlug)
		if trimmedID != "" {
			disabledMap[trimmedID] = true
		}
	}

	return s.calculator.ComputeStatements(rawLines, disabledMap), nil
}

// GenerateAnnualAccountsPDF computes the financial statements with disabledLineIDs excluded,
// renders the full Cabinet SGCAM-style Comptes Annuels & Liasse Fiscale (CERFA 2065, 2050-2059-F, 1330-CVAE, 2072, Détail, SIG)
// to PDF, uploads it to Cloudflare R2, persists the report record, and returns a StoredFileDTO.
func (s *Service) GenerateAnnualAccountsPDF(
	ctx context.Context,
	entityID string,
	fiscalYear int,
	disabledLineIDs []string,
) (StoredFileDTO, error) {
	trimmedEntity := strings.TrimSpace(entityID)
	if !id.IsValidUUID(trimmedEntity) {
		return StoredFileDTO{}, fmt.Errorf("%w: legal_entity_id must be a valid UUID", domain.ErrValidation)
	}
	if fiscalYear <= 0 {
		fiscalYear = time.Now().UTC().Year()
	}

	meta, err := s.resolveAnnualAccountsMetadata(ctx, trimmedEntity, fiscalYear)
	if err != nil {
		return StoredFileDTO{}, err
	}

	stmt, err := s.PreviewFinancialStatements(ctx, trimmedEntity, fiscalYear, disabledLineIDs)
	if err != nil {
		return StoredFileDTO{}, err
	}

	renderer := s.annualAccountsRenderer
	if renderer == nil {
		renderer = infrastructure.NewAnnualAccountsPDFRenderer(s.pdfEngine)
	}
	pdfBytes, err := renderer.RenderAnnualAccountsPDF(ctx, meta, stmt)
	if err != nil {
		return StoredFileDTO{}, fmt.Errorf("accounting: render annual accounts pdf: %w", err)
	}

	reportID, err := id.NewV7()
	if err != nil {
		return StoredFileDTO{}, err
	}

	fileName := fmt.Sprintf("annual_accounts_%s_%d.pdf", sanitizeSegment(trimmedEntity), fiscalYear)
	r2Key := fmt.Sprintf("accounting/%s/%d/%s_%s", sanitizeSegment(trimmedEntity), fiscalYear, reportID, fileName)

	if err := s.storage.Upload(ctx, r2Key, "application/pdf", bytes.NewReader(pdfBytes)); err != nil {
		return StoredFileDTO{}, err
	}

	now := time.Now().UTC()
	rep := domain.AnnualAccountReport{
		ID:              reportID,
		LegalEntityID:   trimmedEntity,
		FiscalYear:      fiscalYear,
		DisabledLineIDs: stmt.DisabledLineIDs,
		NetIncomeNCents: stmt.NetIncomeNCents,
		PDFR2Key:        r2Key,
		FileName:        fileName,
		SizeBytes:       int64(len(pdfBytes)),
		CreatedAt:       now,
	}

	if err := s.annualReports.SaveAnnualAccountReport(ctx, rep); err != nil {
		return StoredFileDTO{}, err
	}

	downloadURL, _ := s.storage.PresignGetURL(ctx, r2Key, 24*time.Hour)

	return StoredFileDTO{
		ID:              rep.ID,
		LegalEntityID:   rep.LegalEntityID,
		FiscalYear:      rep.FiscalYear,
		Category:        "ANNUAL_ACCOUNTS_TAX_RETURN",
		FileName:        rep.FileName,
		ContentType:     "application/pdf",
		R2Key:           rep.PDFR2Key,
		SizeBytes:       rep.SizeBytes,
		DownloadURL:     downloadURL,
		DisabledLineIDs: rep.DisabledLineIDs,
		NetIncomeNCents: rep.NetIncomeNCents,
		CreatedAt:       rep.CreatedAt,
	}, nil
}

// ExportEntityFEC exports the normative 18-column DGFiP Fichier des Écritures Comptables (FEC) for an entity and fiscal year.
func (s *Service) ExportEntityFEC(ctx context.Context, entityID string, fiscalYear int) ([]byte, error) {
	trimmedEntity := strings.TrimSpace(entityID)
	if !id.IsValidUUID(trimmedEntity) {
		return nil, fmt.Errorf("%w: legal_entity_id must be a valid UUID", domain.ErrValidation)
	}
	if fiscalYear <= 0 {
		fiscalYear = time.Now().UTC().Year()
	}

	entries, err := s.journalEntries.ListFECEntries(ctx, trimmedEntity, fiscalYear)
	if err != nil {
		return nil, err
	}

	exporter := s.fecExporter
	if exporter == nil {
		exporter = infrastructure.NewFECExporter()
	}
	return exporter.ExportFEC(entries)
}

func (s *Service) syncBalanceLinesFromEntry(ctx context.Context, balancesRepo domain.AccountBalanceRepository, entry domain.JournalEntry) error {
	existing, err := balancesRepo.ListAccountBalanceLines(ctx, entry.LegalEntityID, entry.FiscalYear)
	if err != nil {
		return err
	}
	byLineID := make(map[string]domain.AccountBalanceLine, len(existing))
	lineIDByAccountCode := make(map[string]string, len(existing))
	for _, l := range existing {
		byLineID[l.LineID] = l
		if code := strings.TrimSpace(l.AccountCode); code != "" {
			if _, exists := lineIDByAccountCode[code]; !exists {
				lineIDByAccountCode[code] = l.LineID
			}
		}
	}

	var touchedOrder []string
	touched := make(map[string]bool, len(entry.Lines))
	for _, jl := range entry.Lines {
		sec := domain.StatementSection(jl.Section)
		if sec == "" {
			sec = domain.InferSectionFromAccountCode(jl.AccountCode)
		}
		netDelta := computeSignedLineAmount(sec, jl.DebitCents, jl.CreditCents)
		lineID := strings.TrimSpace(jl.LineID)
		if lineID == "" || lineID == "acct_"+jl.AccountCode {
			if existingLineID, ok := lineIDByAccountCode[strings.TrimSpace(jl.AccountCode)]; ok {
				lineID = existingLineID
			} else if lineID == "" {
				lineID = "acct_" + jl.AccountCode
			}
		}

		cur, exists := byLineID[lineID]
		if !exists {
			balID, bErr := id.NewV7()
			if bErr != nil {
				return bErr
			}
			cur = domain.AccountBalanceLine{
				ID:              balID,
				LegalEntityID:   entry.LegalEntityID,
				FiscalYear:      entry.FiscalYear,
				LineID:          lineID,
				AccountCode:     jl.AccountCode,
				Label:           jl.AccountLabel,
				Section:         string(sec),
				NetNCents:       netDelta,
				NetNMinus1Cents: jl.NetNMinus1Cents,
			}
			if code := strings.TrimSpace(jl.AccountCode); code != "" {
				if _, ok := lineIDByAccountCode[code]; !ok {
					lineIDByAccountCode[code] = lineID
				}
			}
		} else {
			cur.NetNCents = cur.NetNCents.Add(netDelta)
			if jl.NetNMinus1Cents != 0 && cur.NetNMinus1Cents == 0 {
				cur.NetNMinus1Cents = jl.NetNMinus1Cents
			}
		}
		byLineID[lineID] = cur
		if !touched[lineID] {
			touched[lineID] = true
			touchedOrder = append(touchedOrder, lineID)
		}
	}

	updated := make([]domain.AccountBalanceLine, 0, len(touchedOrder))
	for _, lineSlug := range touchedOrder {
		updated = append(updated, byLineID[lineSlug])
	}

	return balancesRepo.UpsertAccountBalanceLines(ctx, entry.LegalEntityID, entry.FiscalYear, updated)
}

func (s *Service) loadEffectiveBalanceLines(ctx context.Context, entityID string, fiscalYear int) ([]domain.AccountBalanceLine, error) {
	lines, err := s.accountBalances.ListAccountBalanceLines(ctx, entityID, fiscalYear)
	if err != nil {
		return nil, err
	}
	if len(lines) > 0 {
		return lines, nil
	}
	return defaultSGCAMBalanceLines(entityID, fiscalYear), nil
}

func computeSignedLineAmount(sec domain.StatementSection, debit, credit money.AmountCents) money.AmountCents {
	switch sec {
	case domain.SectionOperatingRevenue,
		domain.SectionFinancialIncome,
		domain.SectionExtraordinaryIncome,
		domain.SectionEquity,
		domain.SectionLiabilities:
		return credit.Sub(debit)
	default:
		return debit.Sub(credit)
	}
}

func defaultSGCAMBalanceLines(entityID string, fiscalYear int) []domain.AccountBalanceLine {
	return []domain.AccountBalanceLine{
		{
			ID:              id.MustNewV7(),
			LegalEntityID:   entityID,
			FiscalYear:      fiscalYear,
			LineID:          "rev_locaux",
			AccountCode:     "706100",
			Label:           "LOCATIONS LOCAUX",
			Section:         string(domain.SectionOperatingRevenue),
			CerfaCode:       "FR",
			NetNCents:       money.AmountCents(133607_00),
			NetNMinus1Cents: money.AmountCents(132604_00),
		},
		{
			ID:              id.MustNewV7(),
			LegalEntityID:   entityID,
			FiscalYear:      fiscalYear,
			LineID:          "rev_comm",
			AccountCode:     "706200",
			Label:           "LOYERS COMMERCIAUX",
			Section:         string(domain.SectionOperatingRevenue),
			CerfaCode:       "FR",
			NetNCents:       money.AmountCents(22952_00),
			NetNMinus1Cents: money.AmountCents(15301_00),
		},
		{
			ID:              id.MustNewV7(),
			LegalEntityID:   entityID,
			FiscalYear:      fiscalYear,
			LineID:          "chg_loc",
			AccountCode:     "614000",
			Label:           "CHARGES LOCATIVES",
			Section:         string(domain.SectionExternalExpenses),
			CerfaCode:       "FS",
			NetNCents:       money.AmountCents(45001_00),
			NetNMinus1Cents: money.AmountCents(77663_00),
		},
		{
			ID:              id.MustNewV7(),
			LegalEntityID:   entityID,
			FiscalYear:      fiscalYear,
			LineID:          "chg_tf",
			AccountCode:     "635120",
			Label:           "TAXES FONCIERES",
			Section:         string(domain.SectionTaxesAndDuties),
			CerfaCode:       "FX",
			NetNCents:       money.AmountCents(56916_00),
			NetNMinus1Cents: money.AmountCents(55767_00),
		},
		{
			ID:              id.MustNewV7(),
			LegalEntityID:   entityID,
			FiscalYear:      fiscalYear,
			LineID:          "chg_fourn_admin",
			AccountCode:     "606400",
			Label:           "FOURNITURES ADMINISTRATIVES",
			Section:         string(domain.SectionExternalExpenses),
			CerfaCode:       "FS",
			NetNCents:       money.AmountCents(50000_00),
			NetNMinus1Cents: 0,
		},
	}
}

func (s *Service) resolveAnnualAccountsMetadata(ctx context.Context, entityID string, fiscalYear int) (domain.AnnualAccountsMetadata, error) {
	if s.portfolioSvc != nil {
		summary, err := s.portfolioSvc.GetEntityPortfolioSummary(ctx, entityID)
		if err != nil {
			return domain.AnnualAccountsMetadata{}, err
		}
		if strings.TrimSpace(summary.Entity.ID) != "" {
			shareholders := make([]domain.ShareholderCapitalRow, 0, len(summary.Shareholders))
			for _, sh := range summary.Shareholders {
				role := "Associé"
				if sh.RemunerationPaidCents > 0 {
					role = "Gérant Associé"
				}
				shareholders = append(shareholders, domain.ShareholderCapitalRow{
					FullName:              sh.FullName,
					Role:                  role,
					SharesCount:           sh.SharesCount,
					DetentionBasisPoints:  sh.DetentionBasisPoints,
					CurrentAccountCents:   sh.CurrentAccountCents,
					RemunerationPaidCents: sh.RemunerationPaidCents,
				})
			}
			return domain.AnnualAccountsMetadata{
				LegalEntityID:     summary.Entity.ID,
				EntityName:        summary.Entity.Name,
				EntityType:        string(summary.Entity.EntityType),
				SIRET:             summary.Entity.SIRET,
				NAFCode:           summary.Entity.NAFCode,
				Address:           summary.Entity.Address,
				FiscalYear:        fiscalYear,
				PeriodStart:       fmt.Sprintf("01/01/%d", fiscalYear),
				PeriodEnd:         fmt.Sprintf("31/12/%d", fiscalYear),
				ShareCapitalCents: summary.Entity.ShareCapitalCents,
				TotalShares:       summary.Entity.TotalShares,
				AccountantFirm:    "Cabinet SGCAM — Société de Gestion et de Comptabilité",
				Shareholders:      shareholders,
			}, nil
		}
	}
	return defaultAnnualAccountsMetadata(entityID, fiscalYear), nil
}

func defaultAnnualAccountsMetadata(entityID string, fiscalYear int) domain.AnnualAccountsMetadata {
	return domain.AnnualAccountsMetadata{
		LegalEntityID:     entityID,
		EntityName:        "SARL INVESTISSIMO",
		EntityType:        "SARL_IS",
		SIRET:             "38012345600018",
		NAFCode:           "6820B",
		Address:           "93300 AUBERVILLIERS",
		FiscalYear:        fiscalYear,
		PeriodStart:       fmt.Sprintf("01/01/%d", fiscalYear),
		PeriodEnd:         fmt.Sprintf("31/12/%d", fiscalYear),
		ShareCapitalCents: money.AmountCents(256000_00),
		TotalShares:       500,
		AccountantFirm:    "Cabinet SGCAM — Société de Gestion et de Comptabilité",
		Shareholders: []domain.ShareholderCapitalRow{
			{
				FullName:              "Associé Gérant",
				Role:                  "Gérant Associé",
				SharesCount:           250,
				DetentionBasisPoints:  5000,
				CurrentAccountCents:   money.AmountCents(45000_00),
				RemunerationPaidCents: money.AmountCents(24000_00),
			},
			{
				FullName:              "Associé Co-Gérant",
				Role:                  "Associé",
				SharesCount:           250,
				DetentionBasisPoints:  5000,
				CurrentAccountCents:   money.AmountCents(35000_00),
				RemunerationPaidCents: 0,
			},
		},
	}
}

func toJournalEntryDTO(entry domain.JournalEntry) JournalEntryDTO {
	var totalDebit, totalCredit money.AmountCents
	lines := make([]JournalLineDTO, 0, len(entry.Lines))
	for _, l := range entry.Lines {
		totalDebit = totalDebit.Add(l.DebitCents)
		totalCredit = totalCredit.Add(l.CreditCents)
		lines = append(lines, JournalLineDTO{
			ID:              l.ID,
			LineID:          l.LineID,
			AccountCode:     l.AccountCode,
			AccountLabel:    l.AccountLabel,
			SubAccountCode:  l.SubAccountCode,
			SubAccountLabel: l.SubAccountLabel,
			Section:         l.Section,
			Label:           l.Label,
			DebitCents:      l.DebitCents,
			CreditCents:     l.CreditCents,
			NetNMinus1Cents: l.NetNMinus1Cents,
		})
	}
	return JournalEntryDTO{
		ID:               entry.ID,
		LegalEntityID:    entry.LegalEntityID,
		FiscalYear:       entry.FiscalYear,
		JournalCode:      entry.JournalCode,
		JournalLabel:     entry.JournalLabel,
		EntryNumber:      entry.EntryNumber,
		EntryDate:        entry.EntryDate.Format("2006-01-02"),
		DocumentRef:      entry.DocumentRef,
		DocumentDate:     entry.DocumentDate.Format("2006-01-02"),
		Label:            entry.Label,
		ValidationDate:   entry.ValidationDate.Format("2006-01-02"),
		TotalDebitCents:  totalDebit,
		TotalCreditCents: totalCredit,
		Lines:            lines,
		CreatedAt:        entry.CreatedAt,
	}
}

func defaultJournalLabel(code string) string {
	switch code {
	case "VT", "VE":
		return "Sales & Rent Journal"
	case "AC", "HA":
		return "Purchases Journal"
	case "BQ", "BA":
		return "Bank Journal"
	case "CA":
		return "Cash Journal"
	case "AN":
		return "Opening Balances"
	default:
		return "General Journal"
	}
}

func parseDateOrToday(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	t, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

func sanitizeSegment(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			sb.WriteRune(r)
		} else {
			sb.WriteByte('_')
		}
	}
	res := strings.Trim(sb.String(), "_")
	if res == "" {
		return "entity"
	}
	return res
}

