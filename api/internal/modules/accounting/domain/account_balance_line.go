package domain

import (
	"context"
	"strings"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// StatementSection identifies the Balance Sheet or Income Statement section of an account balance line.
type StatementSection string

const (
	SectionOperatingRevenue      StatementSection = "OPERATING_REVENUE"
	SectionExternalExpenses      StatementSection = "EXTERNAL_EXPENSES"
	SectionTaxesAndDuties        StatementSection = "TAXES_AND_DUTIES"
	SectionPersonnelExpenses     StatementSection = "PERSONNEL_EXPENSES"
	SectionDepreciation          StatementSection = "DEPRECIATION"
	SectionOtherExpenses         StatementSection = "OTHER_EXPENSES"
	SectionFinancialIncome       StatementSection = "FINANCIAL_INCOME"
	SectionFinancialExpenses     StatementSection = "FINANCIAL_EXPENSES"
	SectionExtraordinaryIncome   StatementSection = "EXTRAORDINARY_INCOME"
	SectionExtraordinaryExpenses StatementSection = "EXTRAORDINARY_EXPENSES"
	SectionIncomeTax             StatementSection = "INCOME_TAX"
	SectionFixedAssets           StatementSection = "FIXED_ASSETS"
	SectionCurrentAssets         StatementSection = "CURRENT_ASSETS"
	SectionEquity                StatementSection = "EQUITY"
	SectionLiabilities           StatementSection = "LIABILITIES"
)

// InferSectionFromAccountCode maps a French PCG account code (Classes 1 to 7) to its normative StatementSection.
func InferSectionFromAccountCode(accountCode string) StatementSection {
	code := strings.TrimSpace(accountCode)
	switch {
	case strings.HasPrefix(code, "70"), strings.HasPrefix(code, "71"), strings.HasPrefix(code, "72"),
		strings.HasPrefix(code, "74"), strings.HasPrefix(code, "75"), strings.HasPrefix(code, "781"), strings.HasPrefix(code, "791"):
		return SectionOperatingRevenue
	case strings.HasPrefix(code, "76"), strings.HasPrefix(code, "786"), strings.HasPrefix(code, "796"):
		return SectionFinancialIncome
	case strings.HasPrefix(code, "77"), strings.HasPrefix(code, "787"), strings.HasPrefix(code, "797"):
		return SectionExtraordinaryIncome
	case strings.HasPrefix(code, "60"), strings.HasPrefix(code, "61"), strings.HasPrefix(code, "62"):
		return SectionExternalExpenses
	case strings.HasPrefix(code, "63"):
		return SectionTaxesAndDuties
	case strings.HasPrefix(code, "64"):
		return SectionPersonnelExpenses
	case strings.HasPrefix(code, "65"):
		return SectionOtherExpenses
	case strings.HasPrefix(code, "66"), strings.HasPrefix(code, "686"):
		return SectionFinancialExpenses
	case strings.HasPrefix(code, "67"), strings.HasPrefix(code, "687"):
		return SectionExtraordinaryExpenses
	case strings.HasPrefix(code, "681"), strings.HasPrefix(code, "68"):
		return SectionDepreciation
	case strings.HasPrefix(code, "69"):
		return SectionIncomeTax
	case strings.HasPrefix(code, "2"):
		return SectionFixedAssets
	case strings.HasPrefix(code, "3"), strings.HasPrefix(code, "41"), strings.HasPrefix(code, "46"),
		strings.HasPrefix(code, "486"), strings.HasPrefix(code, "5"):
		return SectionCurrentAssets
	case strings.HasPrefix(code, "10"), strings.HasPrefix(code, "11"), strings.HasPrefix(code, "12"),
		strings.HasPrefix(code, "13"), strings.HasPrefix(code, "14"), strings.HasPrefix(code, "15"):
		return SectionEquity
	case strings.HasPrefix(code, "16"), strings.HasPrefix(code, "17"), strings.HasPrefix(code, "40"),
		strings.HasPrefix(code, "42"), strings.HasPrefix(code, "43"), strings.HasPrefix(code, "44"),
		strings.HasPrefix(code, "45"), strings.HasPrefix(code, "47"), strings.HasPrefix(code, "487"):
		return SectionLiabilities
	default:
		return SectionExternalExpenses
	}
}

// AccountBalanceLine represents the Aggregate Root for a single PCG account or statement balance line for fiscal year N and N-1.
type AccountBalanceLine struct {
	ID                  string            `json:"id,omitempty"`
	LegalEntityID       string            `json:"legal_entity_id,omitempty"`
	FiscalYear          int               `json:"fiscal_year,omitempty"`
	LineID              string            `json:"line_id"`
	AccountCode         string            `json:"account_code"`
	Label               string            `json:"label"`
	Section             string            `json:"section"`
	CerfaCode           string            `json:"cerfa_code,omitempty"`
	GrossNCents         money.AmountCents `json:"gross_n_cents,omitempty"`
	DepreciationNCents  money.AmountCents `json:"depreciation_n_cents,omitempty"`
	NetNCents           money.AmountCents `json:"net_n_cents"`
	NetNMinus1Cents     money.AmountCents `json:"net_n_minus_1_cents"`
}

// AccountBalanceRepository defines the persistence contract for AccountBalanceLine aggregates.
type AccountBalanceRepository interface {
	UpsertAccountBalanceLines(ctx context.Context, entityID string, fiscalYear int, lines []AccountBalanceLine) error
	ListAccountBalanceLines(ctx context.Context, entityID string, fiscalYear int) ([]AccountBalanceLine, error)
}
