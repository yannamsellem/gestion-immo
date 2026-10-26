package domain

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// ComputedLineDTO represents a single calculated line in the interactive financial statements preview and tax return package.
type ComputedLineDTO struct {
	LineID                  string            `json:"line_id"`
	AccountCode             string            `json:"account_code"`
	Label                   string            `json:"label"`
	Section                 string            `json:"section"`
	CerfaCode               string            `json:"cerfa_code,omitempty"`
	Disabled                bool              `json:"disabled"`
	GrossNCents             money.AmountCents `json:"gross_n_cents"`
	DepreciationNCents      money.AmountCents `json:"depreciation_n_cents"`
	NetNCents               money.AmountCents `json:"net_n_cents"`
	NetNMinus1Cents         money.AmountCents `json:"net_n_minus_1_cents"`
	VariationAbsCents       money.AmountCents `json:"variation_abs_cents"`
	VariationPctBasisPoints int64             `json:"variation_pct_basis_points"`
	VariationNotSignificant bool              `json:"variation_not_significant"`
}

// ComputedFinancialStatementsDTO holds the complete dynamically recomputed Balance Sheet, Income Statement, and Intermediate Management Balances.
type ComputedFinancialStatementsDTO struct {
	Lines                                []ComputedLineDTO `json:"lines"`
	ActiveLines                          []ComputedLineDTO `json:"active_lines"`
	DisabledLineIDs                      []string          `json:"disabled_line_ids"`
	TotalOperatingRevenueNCents          money.AmountCents `json:"total_operating_revenue_n_cents"`
	TotalOperatingRevenueNMinus1Cents    money.AmountCents `json:"total_operating_revenue_n_minus_1_cents"`
	TotalExternalExpensesNCents          money.AmountCents `json:"total_external_expenses_n_cents"`
	TotalExternalExpensesNMinus1Cents    money.AmountCents `json:"total_external_expenses_n_minus_1_cents"`
	TotalTaxesAndDutiesNCents            money.AmountCents `json:"total_taxes_and_duties_n_cents"`
	TotalTaxesAndDutiesNMinus1Cents      money.AmountCents `json:"total_taxes_and_duties_n_minus_1_cents"`
	TotalPersonnelExpensesNCents         money.AmountCents `json:"total_personnel_expenses_n_cents"`
	TotalPersonnelExpensesNMinus1Cents   money.AmountCents `json:"total_personnel_expenses_n_minus_1_cents"`
	TotalDepreciationNCents              money.AmountCents `json:"total_depreciation_n_cents"`
	TotalDepreciationNMinus1Cents        money.AmountCents `json:"total_depreciation_n_minus_1_cents"`
	TotalOtherExpensesNCents             money.AmountCents `json:"total_other_expenses_n_cents"`
	TotalOtherExpensesNMinus1Cents       money.AmountCents `json:"total_other_expenses_n_minus_1_cents"`
	TotalOperatingExpensesNCents         money.AmountCents `json:"total_operating_expenses_n_cents"`
	TotalOperatingExpensesNMinus1Cents   money.AmountCents `json:"total_operating_expenses_n_minus_1_cents"`
	ValueAddedNCents                     money.AmountCents `json:"value_added_n_cents"`
	ValueAddedNMinus1Cents               money.AmountCents `json:"value_added_n_minus_1_cents"`
	GrossOperatingSurplusNCents          money.AmountCents `json:"gross_operating_surplus_n_cents"`
	GrossOperatingSurplusNMinus1Cents    money.AmountCents `json:"gross_operating_surplus_n_minus_1_cents"`
	OperatingIncomeNCents                money.AmountCents `json:"operating_income_n_cents"`
	OperatingIncomeNMinus1Cents          money.AmountCents `json:"operating_income_n_minus_1_cents"`
	TotalFinancialIncomeNCents           money.AmountCents `json:"total_financial_income_n_cents"`
	TotalFinancialIncomeNMinus1Cents     money.AmountCents `json:"total_financial_income_n_minus_1_cents"`
	TotalFinancialExpensesNCents         money.AmountCents `json:"total_financial_expenses_n_cents"`
	TotalFinancialExpensesNMinus1Cents   money.AmountCents `json:"total_financial_expenses_n_minus_1_cents"`
	FinancialResultNCents                money.AmountCents `json:"financial_result_n_cents"`
	FinancialResultNMinus1Cents          money.AmountCents `json:"financial_result_n_minus_1_cents"`
	OrdinaryIncomeBeforeTaxNCents        money.AmountCents `json:"ordinary_income_before_tax_n_cents"`
	OrdinaryIncomeBeforeTaxNMinus1Cents  money.AmountCents `json:"ordinary_income_before_tax_n_minus_1_cents"`
	TotalExtraordinaryIncomeNCents       money.AmountCents `json:"total_extraordinary_income_n_cents"`
	TotalExtraordinaryIncomeNMinus1Cents money.AmountCents `json:"total_extraordinary_income_n_minus_1_cents"`
	TotalExtraordinaryExpensesNCents     money.AmountCents `json:"total_extraordinary_expenses_n_cents"`
	TotalExtraordinaryExpensesNMinus1Cents money.AmountCents `json:"total_extraordinary_expenses_n_minus_1_cents"`
	ExtraordinaryResultNCents            money.AmountCents `json:"extraordinary_result_n_cents"`
	ExtraordinaryResultNMinus1Cents      money.AmountCents `json:"extraordinary_result_n_minus_1_cents"`
	TotalIncomeTaxNCents                 money.AmountCents `json:"total_income_tax_n_cents"`
	TotalIncomeTaxNMinus1Cents           money.AmountCents `json:"total_income_tax_n_minus_1_cents"`
	TotalRevenueNCents                   money.AmountCents `json:"total_revenue_n_cents"`
	TotalRevenueNMinus1Cents             money.AmountCents `json:"total_revenue_n_minus_1_cents"`
	TotalExpensesNCents                  money.AmountCents `json:"total_expenses_n_cents"`
	TotalExpensesNMinus1Cents            money.AmountCents `json:"total_expenses_n_minus_1_cents"`
	NetIncomeNCents                      money.AmountCents `json:"net_income_n_cents"`
	NetIncomeNMinus1Cents                money.AmountCents `json:"net_income_n_minus_1_cents"`
	SelfFinancingCapacityNCents          money.AmountCents `json:"self_financing_capacity_n_cents"`
	SelfFinancingCapacityNMinus1Cents    money.AmountCents `json:"self_financing_capacity_n_minus_1_cents"`
	FixedAssetsGrossNCents               money.AmountCents `json:"fixed_assets_gross_n_cents"`
	FixedAssetsDepreciationNCents        money.AmountCents `json:"fixed_assets_depreciation_n_cents"`
	FixedAssetsNetNCents                 money.AmountCents `json:"fixed_assets_net_n_cents"`
	FixedAssetsNetNMinus1Cents           money.AmountCents `json:"fixed_assets_net_n_minus_1_cents"`
	CurrentAssetsNetNCents               money.AmountCents `json:"current_assets_net_n_cents"`
	CurrentAssetsNetNMinus1Cents         money.AmountCents `json:"current_assets_net_n_minus_1_cents"`
	TotalAssetsNCents                    money.AmountCents `json:"total_assets_n_cents"`
	TotalAssetsNMinus1Cents              money.AmountCents `json:"total_assets_n_minus_1_cents"`
	EquityExclNetIncomeNCents            money.AmountCents `json:"equity_excl_net_income_n_cents"`
	EquityExclNetIncomeNMinus1Cents      money.AmountCents `json:"equity_excl_net_income_n_minus_1_cents"`
	BalanceSheetNetIncomeNCents          money.AmountCents `json:"balance_sheet_net_income_n_cents"`
	BalanceSheetNetIncomeNMinus1Cents    money.AmountCents `json:"balance_sheet_net_income_n_minus_1_cents"`
	TotalEquityNCents                    money.AmountCents `json:"total_equity_n_cents"`
	TotalEquityNMinus1Cents              money.AmountCents `json:"total_equity_n_minus_1_cents"`
	TotalLiabilitiesNCents               money.AmountCents `json:"total_liabilities_n_cents"`
	TotalLiabilitiesNMinus1Cents         money.AmountCents `json:"total_liabilities_n_minus_1_cents"`
	TotalEquityAndLiabilitiesNCents      money.AmountCents `json:"total_equity_and_liabilities_n_cents"`
	TotalEquityAndLiabilitiesNMinus1Cents money.AmountCents `json:"total_equity_and_liabilities_n_minus_1_cents"`
}

// StatementCalculator computes Balance Sheet, Income Statement, and Intermediate Management Balances
// with dynamic line toggling using pure integer arithmetic.
type StatementCalculator struct{}

// NewStatementCalculator constructs a new StatementCalculator.
func NewStatementCalculator() *StatementCalculator {
	return &StatementCalculator{}
}

// ComputeStatements calculates line-level variations and cascades active lines into all Income Statement,
// Intermediate Management Balances (Value Added, Gross Operating Surplus, Operating Income, Self-Financing Capacity),
// and Balance Sheet (Assets, Equity, Liabilities) subtotals.
func (c *StatementCalculator) ComputeStatements(
	rawLines []AccountBalanceLine,
	disabledLineIDs map[string]bool,
) ComputedFinancialStatementsDTO {
	var out ComputedFinancialStatementsDTO
	out.Lines = make([]ComputedLineDTO, 0, len(rawLines))
	out.ActiveLines = make([]ComputedLineDTO, 0, len(rawLines))

	disabledSet := make(map[string]bool, len(disabledLineIDs))
	for k, v := range disabledLineIDs {
		trimmed := strings.TrimSpace(k)
		if trimmed != "" && v {
			disabledSet[trimmed] = true
			out.DisabledLineIDs = append(out.DisabledLineIDs, trimmed)
		}
	}
	sort.Strings(out.DisabledLineIDs)
	if out.DisabledLineIDs == nil {
		out.DisabledLineIDs = []string{}
	}

	for _, raw := range rawLines {
		sec := StatementSection(strings.ToUpper(strings.TrimSpace(raw.Section)))
		if sec == "" {
			sec = InferSectionFromAccountCode(raw.AccountCode)
		}

		isDisabled := disabledSet[strings.TrimSpace(raw.LineID)] || disabledSet[strings.TrimSpace(raw.AccountCode)]
		varAbs := raw.NetNCents.Sub(raw.NetNMinus1Cents)
		varBP, notSignificant := ComputeVariationBasisPoints(raw.NetNCents, raw.NetNMinus1Cents)

		grossN := raw.GrossNCents
		if grossN == 0 && raw.DepreciationNCents == 0 {
			grossN = raw.NetNCents
		}

		dto := ComputedLineDTO{
			LineID:                  raw.LineID,
			AccountCode:             raw.AccountCode,
			Label:                   raw.Label,
			Section:                 string(sec),
			CerfaCode:               raw.CerfaCode,
			Disabled:                isDisabled,
			GrossNCents:             grossN,
			DepreciationNCents:      raw.DepreciationNCents,
			NetNCents:               raw.NetNCents,
			NetNMinus1Cents:         raw.NetNMinus1Cents,
			VariationAbsCents:       varAbs,
			VariationPctBasisPoints: varBP,
			VariationNotSignificant: notSignificant,
		}
		out.Lines = append(out.Lines, dto)

		if isDisabled {
			continue
		}
		out.ActiveLines = append(out.ActiveLines, dto)

		switch sec {
		case SectionOperatingRevenue:
			out.TotalOperatingRevenueNCents = out.TotalOperatingRevenueNCents.Add(raw.NetNCents)
			out.TotalOperatingRevenueNMinus1Cents = out.TotalOperatingRevenueNMinus1Cents.Add(raw.NetNMinus1Cents)
		case SectionExternalExpenses:
			out.TotalExternalExpensesNCents = out.TotalExternalExpensesNCents.Add(raw.NetNCents)
			out.TotalExternalExpensesNMinus1Cents = out.TotalExternalExpensesNMinus1Cents.Add(raw.NetNMinus1Cents)
		case SectionTaxesAndDuties:
			out.TotalTaxesAndDutiesNCents = out.TotalTaxesAndDutiesNCents.Add(raw.NetNCents)
			out.TotalTaxesAndDutiesNMinus1Cents = out.TotalTaxesAndDutiesNMinus1Cents.Add(raw.NetNMinus1Cents)
		case SectionPersonnelExpenses:
			out.TotalPersonnelExpensesNCents = out.TotalPersonnelExpensesNCents.Add(raw.NetNCents)
			out.TotalPersonnelExpensesNMinus1Cents = out.TotalPersonnelExpensesNMinus1Cents.Add(raw.NetNMinus1Cents)
		case SectionDepreciation:
			out.TotalDepreciationNCents = out.TotalDepreciationNCents.Add(raw.NetNCents)
			out.TotalDepreciationNMinus1Cents = out.TotalDepreciationNMinus1Cents.Add(raw.NetNMinus1Cents)
		case SectionOtherExpenses:
			out.TotalOtherExpensesNCents = out.TotalOtherExpensesNCents.Add(raw.NetNCents)
			out.TotalOtherExpensesNMinus1Cents = out.TotalOtherExpensesNMinus1Cents.Add(raw.NetNMinus1Cents)
		case SectionFinancialIncome:
			out.TotalFinancialIncomeNCents = out.TotalFinancialIncomeNCents.Add(raw.NetNCents)
			out.TotalFinancialIncomeNMinus1Cents = out.TotalFinancialIncomeNMinus1Cents.Add(raw.NetNMinus1Cents)
		case SectionFinancialExpenses:
			out.TotalFinancialExpensesNCents = out.TotalFinancialExpensesNCents.Add(raw.NetNCents)
			out.TotalFinancialExpensesNMinus1Cents = out.TotalFinancialExpensesNMinus1Cents.Add(raw.NetNMinus1Cents)
		case SectionExtraordinaryIncome:
			out.TotalExtraordinaryIncomeNCents = out.TotalExtraordinaryIncomeNCents.Add(raw.NetNCents)
			out.TotalExtraordinaryIncomeNMinus1Cents = out.TotalExtraordinaryIncomeNMinus1Cents.Add(raw.NetNMinus1Cents)
		case SectionExtraordinaryExpenses:
			out.TotalExtraordinaryExpensesNCents = out.TotalExtraordinaryExpensesNCents.Add(raw.NetNCents)
			out.TotalExtraordinaryExpensesNMinus1Cents = out.TotalExtraordinaryExpensesNMinus1Cents.Add(raw.NetNMinus1Cents)
		case SectionIncomeTax:
			out.TotalIncomeTaxNCents = out.TotalIncomeTaxNCents.Add(raw.NetNCents)
			out.TotalIncomeTaxNMinus1Cents = out.TotalIncomeTaxNMinus1Cents.Add(raw.NetNMinus1Cents)
		case SectionFixedAssets:
			out.FixedAssetsGrossNCents = out.FixedAssetsGrossNCents.Add(grossN)
			out.FixedAssetsDepreciationNCents = out.FixedAssetsDepreciationNCents.Add(raw.DepreciationNCents)
			out.FixedAssetsNetNCents = out.FixedAssetsNetNCents.Add(raw.NetNCents)
			out.FixedAssetsNetNMinus1Cents = out.FixedAssetsNetNMinus1Cents.Add(raw.NetNMinus1Cents)
		case SectionCurrentAssets:
			out.CurrentAssetsNetNCents = out.CurrentAssetsNetNCents.Add(raw.NetNCents)
			out.CurrentAssetsNetNMinus1Cents = out.CurrentAssetsNetNMinus1Cents.Add(raw.NetNMinus1Cents)
		case SectionEquity:
			// Exclude 120/129 (Net Income) if present in raw lines because it is dynamically recomputed
			if !strings.HasPrefix(strings.TrimSpace(raw.AccountCode), "12") {
				out.EquityExclNetIncomeNCents = out.EquityExclNetIncomeNCents.Add(raw.NetNCents)
				out.EquityExclNetIncomeNMinus1Cents = out.EquityExclNetIncomeNMinus1Cents.Add(raw.NetNMinus1Cents)
			}
		case SectionLiabilities:
			out.TotalLiabilitiesNCents = out.TotalLiabilitiesNCents.Add(raw.NetNCents)
			out.TotalLiabilitiesNMinus1Cents = out.TotalLiabilitiesNMinus1Cents.Add(raw.NetNMinus1Cents)
		}
	}

	// Operating expenses subtotal
	out.TotalOperatingExpensesNCents = out.TotalExternalExpensesNCents.
		Add(out.TotalTaxesAndDutiesNCents).
		Add(out.TotalPersonnelExpensesNCents).
		Add(out.TotalDepreciationNCents).
		Add(out.TotalOtherExpensesNCents)
	out.TotalOperatingExpensesNMinus1Cents = out.TotalExternalExpensesNMinus1Cents.
		Add(out.TotalTaxesAndDutiesNMinus1Cents).
		Add(out.TotalPersonnelExpensesNMinus1Cents).
		Add(out.TotalDepreciationNMinus1Cents).
		Add(out.TotalOtherExpensesNMinus1Cents)

	// Value Added = Operating Revenue - External Expenses
	out.ValueAddedNCents = out.TotalOperatingRevenueNCents.Sub(out.TotalExternalExpensesNCents)
	out.ValueAddedNMinus1Cents = out.TotalOperatingRevenueNMinus1Cents.Sub(out.TotalExternalExpensesNMinus1Cents)

	// Gross Operating Surplus = Value Added - Taxes & Duties - Personnel Expenses
	out.GrossOperatingSurplusNCents = out.ValueAddedNCents.Sub(out.TotalTaxesAndDutiesNCents).Sub(out.TotalPersonnelExpensesNCents)
	out.GrossOperatingSurplusNMinus1Cents = out.ValueAddedNMinus1Cents.Sub(out.TotalTaxesAndDutiesNMinus1Cents).Sub(out.TotalPersonnelExpensesNMinus1Cents)

	// Operating Income = Operating Revenue - Operating Expenses
	out.OperatingIncomeNCents = out.TotalOperatingRevenueNCents.Sub(out.TotalOperatingExpensesNCents)
	out.OperatingIncomeNMinus1Cents = out.TotalOperatingRevenueNMinus1Cents.Sub(out.TotalOperatingExpensesNMinus1Cents)

	// Financial Result & Ordinary Income Before Tax
	out.FinancialResultNCents = out.TotalFinancialIncomeNCents.Sub(out.TotalFinancialExpensesNCents)
	out.FinancialResultNMinus1Cents = out.TotalFinancialIncomeNMinus1Cents.Sub(out.TotalFinancialExpensesNMinus1Cents)
	out.OrdinaryIncomeBeforeTaxNCents = out.OperatingIncomeNCents.Add(out.FinancialResultNCents)
	out.OrdinaryIncomeBeforeTaxNMinus1Cents = out.OperatingIncomeNMinus1Cents.Add(out.FinancialResultNMinus1Cents)

	// Extraordinary Result
	out.ExtraordinaryResultNCents = out.TotalExtraordinaryIncomeNCents.Sub(out.TotalExtraordinaryExpensesNCents)
	out.ExtraordinaryResultNMinus1Cents = out.TotalExtraordinaryIncomeNMinus1Cents.Sub(out.TotalExtraordinaryExpensesNMinus1Cents)

	// Total Revenue & Total Expenses
	out.TotalRevenueNCents = out.TotalOperatingRevenueNCents.
		Add(out.TotalFinancialIncomeNCents).
		Add(out.TotalExtraordinaryIncomeNCents)
	out.TotalRevenueNMinus1Cents = out.TotalOperatingRevenueNMinus1Cents.
		Add(out.TotalFinancialIncomeNMinus1Cents).
		Add(out.TotalExtraordinaryIncomeNMinus1Cents)

	out.TotalExpensesNCents = out.TotalOperatingExpensesNCents.
		Add(out.TotalFinancialExpensesNCents).
		Add(out.TotalExtraordinaryExpensesNCents).
		Add(out.TotalIncomeTaxNCents)
	out.TotalExpensesNMinus1Cents = out.TotalOperatingExpensesNMinus1Cents.
		Add(out.TotalFinancialExpensesNMinus1Cents).
		Add(out.TotalExtraordinaryExpensesNMinus1Cents).
		Add(out.TotalIncomeTaxNMinus1Cents)

	// Net Income & Self-Financing Capacity
	out.NetIncomeNCents = out.TotalRevenueNCents.Sub(out.TotalExpensesNCents)
	out.NetIncomeNMinus1Cents = out.TotalRevenueNMinus1Cents.Sub(out.TotalExpensesNMinus1Cents)
	out.SelfFinancingCapacityNCents = out.NetIncomeNCents.Add(out.TotalDepreciationNCents)
	out.SelfFinancingCapacityNMinus1Cents = out.NetIncomeNMinus1Cents.Add(out.TotalDepreciationNMinus1Cents)

	// Cascade net income into Balance Sheet Equity (Account 120/129 Net Income)
	out.BalanceSheetNetIncomeNCents = out.NetIncomeNCents
	out.BalanceSheetNetIncomeNMinus1Cents = out.NetIncomeNMinus1Cents
	out.TotalEquityNCents = out.EquityExclNetIncomeNCents.Add(out.BalanceSheetNetIncomeNCents)
	out.TotalEquityNMinus1Cents = out.EquityExclNetIncomeNMinus1Cents.Add(out.BalanceSheetNetIncomeNMinus1Cents)

	out.TotalAssetsNCents = out.FixedAssetsNetNCents.Add(out.CurrentAssetsNetNCents)
	out.TotalAssetsNMinus1Cents = out.FixedAssetsNetNMinus1Cents.Add(out.CurrentAssetsNetNMinus1Cents)
	out.TotalEquityAndLiabilitiesNCents = out.TotalEquityNCents.Add(out.TotalLiabilitiesNCents)
	out.TotalEquityAndLiabilitiesNMinus1Cents = out.TotalEquityNMinus1Cents.Add(out.TotalLiabilitiesNMinus1Cents)

	return out
}

// ComputeVariationBasisPoints computes the N vs N-1 percentage variation in integer basis points (100 bp = 1.00%)
// using half-up integer rounding without any floating-point operations.
// When netNMinus1 == 0, it returns (0, true) indicating the variation is not significant (N/S).
func ComputeVariationBasisPoints(netN, netNMinus1 money.AmountCents) (int64, bool) {
	if netNMinus1 == 0 {
		return 0, true
	}

	diff := int64(netN.Sub(netNMinus1))
	base := int64(netNMinus1.Abs())

	if diff >= 0 {
		return (diff*10000 + base/2) / base, false
	}
	return -((-diff*10000 + base/2) / base), false
}

// FormatVariationPercent formats the N vs N-1 percentage variation for server-side PDF/HTML reports.
// When netNMinus1 == 0, it returns "N/S" per SGCAM Comptes Annuels convention.
func FormatVariationPercent(netN, netNMinus1 money.AmountCents) string {
	bp, notSignificant := ComputeVariationBasisPoints(netN, netNMinus1)
	if notSignificant {
		return "N/S"
	}

	sign := ""
	absBP := bp
	if absBP < 0 {
		sign = "-"
		absBP = -absBP
	}
	wholePct := absBP / 100
	fracPct := absBP % 100
	return fmt.Sprintf("%s%d,%02d %%", sign, wholePct, fracPct)
}
