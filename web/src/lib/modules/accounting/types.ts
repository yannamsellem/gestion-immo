import type { components } from '#lib/core/api/openapi';

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

export type JournalLineInput = components['schemas']['JournalLineInput'];
export type RecordJournalEntryCmd = components['schemas']['RecordJournalEntryCmd'];
export type JournalLineDTO = components['schemas']['JournalLineDTO'];
export type JournalEntryDTO = components['schemas']['JournalEntryDTO'];
export type AccountBalanceLineDTO = components['schemas']['AccountBalanceLine'];
export type ComputedLineDTO = components['schemas']['ComputedLineDTO'];
export type ComputedFinancialStatementsDTO =
	components['schemas']['ComputedFinancialStatementsDTO'];
export type AnnualAccountsReportDTO = components['schemas']['AnnualAccountReportDTO'];
