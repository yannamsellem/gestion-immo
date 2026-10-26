package domain

import (
	"context"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// ShareholderCapitalRow describes a shareholder's equity and current account position for CERFA 2059-F / 2072.
type ShareholderCapitalRow struct {
	FullName              string            `json:"full_name"`
	Role                  string            `json:"role"`
	SharesCount           int64             `json:"shares_count"`
	DetentionBasisPoints  int64             `json:"detention_basis_points"`
	CurrentAccountCents   money.AmountCents `json:"current_account_cents"`
	RemunerationPaidCents money.AmountCents `json:"remuneration_paid_cents"`
}

// AnnualAccountsMetadata holds the legal entity metadata required to render Annual Accounts and Tax Return (CERFA 2065 / 2050-2059-F / 1330-CVAE / 2072).
type AnnualAccountsMetadata struct {
	LegalEntityID     string                  `json:"legal_entity_id"`
	EntityName        string                  `json:"entity_name"`
	EntityType        string                  `json:"entity_type"`
	SIRET             string                  `json:"siret"`
	NAFCode           string                  `json:"naf_code"`
	Address           string                  `json:"address"`
	FiscalYear        int                     `json:"fiscal_year"`
	PeriodStart       string                  `json:"period_start"`
	PeriodEnd         string                  `json:"period_end"`
	ShareCapitalCents money.AmountCents       `json:"share_capital_cents"`
	TotalShares       int64                   `json:"total_shares"`
	AccountantFirm    string                  `json:"accountant_firm"`
	Shareholders      []ShareholderCapitalRow `json:"shareholders,omitempty"`
}

// AnnualAccountReport represents the Aggregate Root for a generated Annual Accounts and Tax Return PDF stored on Cloudflare R2.
type AnnualAccountReport struct {
	ID              string            `json:"id"`
	LegalEntityID   string            `json:"legal_entity_id"`
	FiscalYear      int               `json:"fiscal_year"`
	DisabledLineIDs []string          `json:"disabled_line_ids"`
	NetIncomeNCents money.AmountCents `json:"net_income_n_cents"`
	PDFR2Key        string            `json:"pdf_r2_key"`
	FileName        string            `json:"file_name"`
	SizeBytes       int64             `json:"size_bytes"`
	CreatedAt       time.Time         `json:"created_at"`
}

// AnnualAccountReportRepository defines the persistence contract for the AnnualAccountReport aggregate root.
type AnnualAccountReportRepository interface {
	SaveAnnualAccountReport(ctx context.Context, report AnnualAccountReport) error
	ListAnnualAccountReports(ctx context.Context, entityID string, fiscalYear int) ([]AnnualAccountReport, error)
}

// AnnualAccountsPDFRendererPort defines the contract for rendering Annual Accounts and Tax Return HTML/PDF documents.
type AnnualAccountsPDFRendererPort interface {
	RenderAnnualAccountsHTML(meta AnnualAccountsMetadata, stmt ComputedFinancialStatementsDTO) (string, error)
	RenderAnnualAccountsPDF(ctx context.Context, meta AnnualAccountsMetadata, stmt ComputedFinancialStatementsDTO) ([]byte, error)
}
