export type StatementSection =
	| 'OPERATING_REVENUE'
	| 'EXTERNAL_EXPENSES'
	| 'TAXES_AND_DUTIES'
	| 'PERSONNEL_EXPENSES'
	| 'DEPRECIATION'
	| 'OTHER_EXPENSES'
	| 'FINANCIAL_INCOME'
	| 'FINANCIAL_EXPENSES'
	| 'EXTRAORDINARY_INCOME'
	| 'EXTRAORDINARY_EXPENSES'
	| 'INCOME_TAX'
	| 'FIXED_ASSETS'
	| 'CURRENT_ASSETS'
	| 'EQUITY'
	| 'LIABILITIES'
	| string;

export interface JournalLineInput {
	line_id?: string;
	account_code: string;
	account_label?: string;
	sub_account_code?: string;
	sub_account_label?: string;
	section?: StatementSection;
	label?: string;
	debit_cents: number;
	credit_cents: number;
	net_n_minus_1_cents?: number;
}

export interface RecordJournalEntryCmd {
	legal_entity_id?: string;
	fiscal_year?: number;
	journal_code?: string;
	journal_label?: string;
	entry_number?: string;
	entry_date?: string;
	document_ref?: string;
	document_date?: string;
	label: string;
	validation_date?: string;
	lines: JournalLineInput[];
}

export interface JournalLineDTO {
	id: string;
	line_id?: string;
	account_code: string;
	account_label: string;
	sub_account_code?: string;
	sub_account_label?: string;
	section: StatementSection;
	label: string;
	debit_cents: number;
	credit_cents: number;
	net_n_minus_1_cents?: number;
}

export interface JournalEntryDTO {
	id: string;
	legal_entity_id: string;
	fiscal_year: number;
	journal_code: string;
	journal_label: string;
	entry_number: string;
	entry_date: string;
	document_ref: string;
	document_date: string;
	label: string;
	validation_date: string;
	total_debit_cents: number;
	total_credit_cents: number;
	lines: JournalLineDTO[];
	created_at: string;
}

export interface AccountBalanceLineDTO {
	id?: string;
	legal_entity_id?: string;
	fiscal_year?: number;
	line_id: string;
	account_code: string;
	label: string;
	section: StatementSection;
	cerfa_code?: string;
	gross_n_cents?: number;
	depreciation_n_cents?: number;
	net_n_cents: number;
	net_n_minus_1_cents: number;
}

export interface ComputedLineDTO {
	line_id: string;
	account_code: string;
	label: string;
	section: StatementSection;
	cerfa_code?: string;
	disabled: boolean;
	gross_n_cents: number;
	depreciation_n_cents: number;
	net_n_cents: number;
	net_n_minus_1_cents: number;
	variation_abs_cents: number;
	variation_pct_basis_points: number;
	variation_not_significant: boolean;
}

export interface ComputedFinancialStatementsDTO {
	lines: ComputedLineDTO[];
	active_lines: ComputedLineDTO[];
	disabled_line_ids: string[];
	total_operating_revenue_n_cents: number;
	total_operating_revenue_n_minus_1_cents: number;
	total_external_expenses_n_cents: number;
	total_external_expenses_n_minus_1_cents: number;
	total_taxes_and_duties_n_cents: number;
	total_taxes_and_duties_n_minus_1_cents: number;
	total_personnel_expenses_n_cents: number;
	total_personnel_expenses_n_minus_1_cents: number;
	total_depreciation_n_cents: number;
	total_depreciation_n_minus_1_cents: number;
	total_other_expenses_n_cents: number;
	total_other_expenses_n_minus_1_cents: number;
	total_operating_expenses_n_cents: number;
	total_operating_expenses_n_minus_1_cents: number;
	value_added_n_cents: number;
	value_added_n_minus_1_cents: number;
	gross_operating_surplus_n_cents: number;
	gross_operating_surplus_n_minus_1_cents: number;
	operating_income_n_cents: number;
	operating_income_n_minus_1_cents: number;
	total_financial_income_n_cents: number;
	total_financial_income_n_minus_1_cents: number;
	total_financial_expenses_n_cents: number;
	total_financial_expenses_n_minus_1_cents: number;
	financial_result_n_cents: number;
	financial_result_n_minus_1_cents: number;
	ordinary_income_before_tax_n_cents: number;
	ordinary_income_before_tax_n_minus_1_cents: number;
	total_extraordinary_income_n_cents: number;
	total_extraordinary_income_n_minus_1_cents: number;
	total_extraordinary_expenses_n_cents: number;
	total_extraordinary_expenses_n_minus_1_cents: number;
	extraordinary_result_n_cents: number;
	extraordinary_result_n_minus_1_cents: number;
	total_income_tax_n_cents: number;
	total_income_tax_n_minus_1_cents: number;
	total_revenue_n_cents: number;
	total_revenue_n_minus_1_cents: number;
	total_expenses_n_cents: number;
	total_expenses_n_minus_1_cents: number;
	net_income_n_cents: number;
	net_income_n_minus_1_cents: number;
	self_financing_capacity_n_cents: number;
	self_financing_capacity_n_minus_1_cents: number;
	fixed_assets_gross_n_cents: number;
	fixed_assets_depreciation_n_cents: number;
	fixed_assets_net_n_cents: number;
	fixed_assets_net_n_minus_1_cents: number;
	current_assets_net_n_cents: number;
	current_assets_net_n_minus_1_cents: number;
	total_assets_n_cents: number;
	total_assets_n_minus_1_cents: number;
	equity_excl_net_income_n_cents: number;
	equity_excl_net_income_n_minus_1_cents: number;
	balance_sheet_net_income_n_cents: number;
	balance_sheet_net_income_n_minus_1_cents: number;
	total_equity_n_cents: number;
	total_equity_n_minus_1_cents: number;
	total_liabilities_n_cents: number;
	total_liabilities_n_minus_1_cents: number;
	total_equity_and_liabilities_n_cents: number;
	total_equity_and_liabilities_n_minus_1_cents: number;
}

export interface AnnualAccountsReportDTO {
	id: string;
	legal_entity_id: string;
	fiscal_year: number;
	category: string;
	file_name: string;
	content_type: string;
	r2_key: string;
	size_bytes: number;
	download_url: string;
	disabled_line_ids: string[];
	net_income_n_cents: number;
	created_at: string;
}
