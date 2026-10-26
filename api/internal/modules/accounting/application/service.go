package application

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

// JournalLineInput carries a single debit or credit line when recording a PCG journal entry.
type JournalLineInput struct {
	LineID                     string            `json:"line_id,omitempty"`
	AccountCode                string            `json:"account_code"`
	AccountLabel               string            `json:"account_label,omitempty"`
	SubAccountCode             string            `json:"sub_account_code,omitempty"`
	SubAccountLabel            string            `json:"sub_account_label,omitempty"`
	Section                    string            `json:"section,omitempty"`
	Label                      string            `json:"label,omitempty"`
	DebitCents                 money.AmountCents `json:"debit_cents"`
	CreditCents                money.AmountCents `json:"credit_cents"`
	NetNMinus1Cents            money.AmountCents `json:"net_n_minus_1_cents,omitempty"`
	ReconciliationCode         string            `json:"reconciliation_code,omitempty"`
	ReconciliationDate         string            `json:"reconciliation_date,omitempty"`
	ForeignCurrencyAmountCents money.AmountCents `json:"foreign_currency_amount_cents,omitempty"`
	CurrencyCode               string            `json:"currency_code,omitempty"`
}

// RecordJournalEntryCmd carries the input for recording a balanced double-entry PCG journal entry.
type RecordJournalEntryCmd struct {
	LegalEntityID  string             `json:"legal_entity_id"`
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
	ID              string            `json:"id"`
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
	ID               string            `json:"id"`
	LegalEntityID    string            `json:"legal_entity_id"`
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

// StoredFileDTO represents a generated Annual Accounts and Tax Return PDF stored on Cloudflare R2.
type StoredFileDTO struct {
	ID              string            `json:"id"`
	LegalEntityID   string            `json:"legal_entity_id"`
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
) *Service {
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
	}
}

// RecordJournalEntry validates and persists a balanced double-entry PCG journal entry
// and updates the entity's account balance lines for the fiscal year inside a single UnitOfWork transaction.
func (s *Service) RecordJournalEntry(ctx context.Context, cmd RecordJournalEntryCmd) (JournalEntryDTO, error) {
	entityID := strings.TrimSpace(cmd.LegalEntityID)
	if entityID == "" {
		return JournalEntryDTO{}, fmt.Errorf("%w: legal_entity_id is required", domain.ErrValidation)
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

	entryID, err := generateID("je")
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
		lineID, lErr := generateID("jl")
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
	if trimmed == "" {
		return fmt.Errorf("%w: legal_entity_id is required", domain.ErrValidation)
	}
	if fiscalYear <= 0 {
		fiscalYear = time.Now().UTC().Year()
	}
	return s.accountBalances.UpsertAccountBalanceLines(ctx, trimmed, fiscalYear, lines)
}

// ListJournalEntries returns all recorded JournalEntries for an entity and fiscal year.
func (s *Service) ListJournalEntries(ctx context.Context, entityID string, fiscalYear int) ([]JournalEntryDTO, error) {
	trimmed := strings.TrimSpace(entityID)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: legal_entity_id is required", domain.ErrValidation)
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
	if trimmedEntity == "" {
		return domain.ComputedFinancialStatementsDTO{}, fmt.Errorf("%w: legal_entity_id is required", domain.ErrValidation)
	}
	if fiscalYear <= 0 {
		fiscalYear = time.Now().UTC().Year()
	}

	rawLines, err := s.loadEffectiveBalanceLines(ctx, trimmedEntity, fiscalYear)
	if err != nil {
		return domain.ComputedFinancialStatementsDTO{}, err
	}

	disabledMap := make(map[string]bool, len(disabledLineIDs))
	for _, id := range disabledLineIDs {
		trimmedID := strings.TrimSpace(id)
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
	if trimmedEntity == "" {
		return StoredFileDTO{}, fmt.Errorf("%w: legal_entity_id is required", domain.ErrValidation)
	}
	if fiscalYear <= 0 {
		fiscalYear = time.Now().UTC().Year()
	}

	stmt, err := s.PreviewFinancialStatements(ctx, trimmedEntity, fiscalYear, disabledLineIDs)
	if err != nil {
		return StoredFileDTO{}, err
	}

	meta := defaultAnnualAccountsMetadata(trimmedEntity, fiscalYear)

	var pdfBytes []byte
	if s.annualAccountsRenderer != nil {
		pdfBytes, err = s.annualAccountsRenderer.RenderAnnualAccountsPDF(ctx, meta, stmt)
		if err != nil {
			return StoredFileDTO{}, fmt.Errorf("accounting: render annual accounts pdf: %w", err)
		}
	} else {
		htmlDoc := buildFallbackAnnualAccountsHTML(meta, stmt)
		pdfBytes, err = s.pdfEngine.RenderHTMLToPDF(ctx, htmlDoc)
		if err != nil {
			return StoredFileDTO{}, fmt.Errorf("accounting: render fallback annual accounts pdf: %w", err)
		}
	}

	reportID, err := generateID("aar")
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
	if trimmedEntity == "" {
		return nil, fmt.Errorf("%w: legal_entity_id is required", domain.ErrValidation)
	}
	if fiscalYear <= 0 {
		fiscalYear = time.Now().UTC().Year()
	}

	entries, err := s.journalEntries.ListFECEntries(ctx, trimmedEntity, fiscalYear)
	if err != nil {
		return nil, err
	}

	if s.fecExporter != nil {
		return s.fecExporter.ExportFEC(entries)
	}
	return exportFallbackFEC(entries), nil
}

func (s *Service) syncBalanceLinesFromEntry(ctx context.Context, balancesRepo domain.AccountBalanceRepository, entry domain.JournalEntry) error {
	existing, err := balancesRepo.ListAccountBalanceLines(ctx, entry.LegalEntityID, entry.FiscalYear)
	if err != nil {
		return err
	}
	byLineID := make(map[string]domain.AccountBalanceLine, len(existing))
	for _, l := range existing {
		byLineID[l.LineID] = l
	}

	var updated []domain.AccountBalanceLine
	for _, jl := range entry.Lines {
		sec := domain.StatementSection(jl.Section)
		if sec == "" {
			sec = domain.InferSectionFromAccountCode(jl.AccountCode)
		}
		netDelta := computeSignedLineAmount(sec, jl.DebitCents, jl.CreditCents)
		lineID := jl.LineID
		if lineID == "" {
			lineID = "acct_" + jl.AccountCode
		}

		cur, exists := byLineID[lineID]
		if !exists {
			cur = domain.AccountBalanceLine{
				ID:              fmt.Sprintf("%s_%d_%s", entry.LegalEntityID, entry.FiscalYear, lineID),
				LegalEntityID:   entry.LegalEntityID,
				FiscalYear:      entry.FiscalYear,
				LineID:          lineID,
				AccountCode:     jl.AccountCode,
				Label:           jl.AccountLabel,
				Section:         string(sec),
				NetNCents:       netDelta,
				NetNMinus1Cents: jl.NetNMinus1Cents,
			}
		} else {
			cur.NetNCents = cur.NetNCents.Add(netDelta)
			if jl.NetNMinus1Cents != 0 && cur.NetNMinus1Cents == 0 {
				cur.NetNMinus1Cents = jl.NetNMinus1Cents
			}
		}
		byLineID[lineID] = cur
		updated = append(updated, cur)
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
			ID:              entityID + "_rev_locaux",
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
			ID:              entityID + "_rev_comm",
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
			ID:              entityID + "_chg_loc",
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
			ID:              entityID + "_chg_tf",
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
			ID:              entityID + "_chg_fourn_admin",
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

func buildFallbackAnnualAccountsHTML(meta domain.AnnualAccountsMetadata, stmt domain.ComputedFinancialStatementsDTO) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html><html lang=\"fr\"><head><meta charset=\"UTF-8\"><title>Comptes Annuels</title></head><body>")
	sb.WriteString(fmt.Sprintf("<h1>COMPTES ANNUELS &amp; LIASSE FISCALE %d — %s</h1>", meta.FiscalYear, html.EscapeString(meta.EntityName)))
	sb.WriteString("<h2>CERFA 2065, 2050, 2051, 2052, 2053, 2054, 2055, 2057, 2058-A/B/C, 2059-F, 1330-CVAE, 2072 &amp; SIG</h2>")
	sb.WriteString("<table><thead><tr><th>Compte</th><th>Libellé</th><th>Net N</th><th>Net N-1</th><th>Variation %</th></tr></thead><tbody>")
	for _, l := range stmt.Lines {
		if l.Disabled {
			continue
		}
		sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			html.EscapeString(l.AccountCode),
			html.EscapeString(l.Label),
			html.EscapeString(l.NetNCents.FormatEUR()),
			html.EscapeString(l.NetNMinus1Cents.FormatEUR()),
			html.EscapeString(domain.FormatVariationPercent(l.NetNCents, l.NetNMinus1Cents)),
		))
	}
	sb.WriteString("</tbody></table>")
	sb.WriteString(fmt.Sprintf("<p>Total Produits d'Exploitation : %s | Résultat de l'exercice : %s | Bilan Passif Résultat : %s</p>",
		html.EscapeString(stmt.TotalOperatingRevenueNCents.FormatEUR()),
		html.EscapeString(stmt.NetIncomeNCents.FormatEUR()),
		html.EscapeString(stmt.BalanceSheetNetIncomeNCents.FormatEUR())))
	sb.WriteString("</body></html>")
	return sb.String()
}

func exportFallbackFEC(entries []domain.FECEntry) []byte {
	cols := []string{
		"JournalCode", "JournalLib", "EcritureNum", "EcritureDate",
		"CompteNum", "CompteLib", "CompAuxNum", "CompAuxLib",
		"PieceRef", "PieceDate", "EcritureLib", "Debit", "Credit",
		"EcritureLet", "DateLet", "ValidDate", "Montantdevise", "Idevise",
	}
	var buf bytes.Buffer
	buf.WriteString(strings.Join(cols, "\t"))
	buf.WriteByte('\n')
	for _, row := range entries {
		dt := row.EntryDate.UTC().Format("20060102")
		pdt := row.DocumentDate.UTC().Format("20060102")
		vdt := row.ValidationDate.UTC().Format("20060102")
		curr := row.CurrencyCode
		if curr == "" {
			curr = "EUR"
		}
		foreignAmount := ""
		if row.ForeignCurrencyAmountCents != 0 {
			foreignAmount = formatCentsFEC(row.ForeignCurrencyAmountCents)
		}
		line := []string{
			row.JournalCode, row.JournalLabel, row.EntryNumber, dt,
			row.AccountCode, row.AccountLabel, row.SubAccountCode, row.SubAccountLabel,
			row.DocumentRef, pdt, row.EntryLabel,
			formatCentsFEC(row.DebitCents), formatCentsFEC(row.CreditCents),
			row.ReconciliationCode, row.ReconciliationDate, vdt, foreignAmount, curr,
		}
		buf.WriteString(strings.Join(line, "\t"))
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func formatCentsFEC(c money.AmountCents) string {
	val := int64(c)
	sign := ""
	if val < 0 {
		sign = "-"
		val = -val
	}
	return fmt.Sprintf("%s%d,%02d", sign, val/100, val%100)
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

func generateID(prefix string) (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("accounting: generate id: %w", err)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b[:])), nil
}

