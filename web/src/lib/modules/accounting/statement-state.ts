import { parseEuroToCents } from '#lib/core/formatters/money';
import type { LegalEntityDTO } from '#lib/modules/portfolio/types';
import type { StatementSection } from './types';

export const STATEMENT_SECTION_OPTIONS: Array<{ value: StatementSection; label: string }> = [
	{ value: 'OPERATING_REVENUE', label: "Produits d'exploitation" },
	{ value: 'EXTERNAL_EXPENSES', label: 'Charges externes (61/62)' },
	{ value: 'TAXES_AND_DUTIES', label: 'Impôts et taxes (63)' },
	{ value: 'PERSONNEL_EXPENSES', label: 'Charges de personnel (64)' },
	{ value: 'DEPRECIATION', label: 'Dotations aux amortissements' },
	{ value: 'OTHER_EXPENSES', label: 'Autres charges de gestion' },
	{ value: 'FINANCIAL_INCOME', label: 'Produits financiers (76)' },
	{ value: 'FINANCIAL_EXPENSES', label: 'Charges financières (66)' },
	{ value: 'EXTRAORDINARY_INCOME', label: 'Produits exceptionnels (77)' },
	{ value: 'EXTRAORDINARY_EXPENSES', label: 'Charges exceptionnelles (67)' },
	{ value: 'INCOME_TAX', label: 'Impôt sur les bénéfices (695)' },
	{ value: 'FIXED_ASSETS', label: 'Actif immobilisé (Classe 2)' },
	{ value: 'CURRENT_ASSETS', label: 'Actif circulant & Trésorerie (Classes 3/4/5)' },
	{ value: 'EQUITY', label: 'Capitaux propres (Classe 1)' },
	{ value: 'LIABILITIES', label: 'Dettes financières & diverses (Classes 16/4)' }
];

const SECTION_LABEL_MAP = new Map(
	STATEMENT_SECTION_OPTIONS.map((option) => [option.value, option.label])
);

export function toggleDisabledLineId(currentIds: string[], lineId: string): string[] {
	const trimmed = lineId.trim();
	if (!trimmed) {
		return [...currentIds];
	}
	if (currentIds.includes(trimmed)) {
		return currentIds.filter((id) => id !== trimmed);
	}
	return [...currentIds, trimmed];
}

export function formatStatementSectionLabel(section: StatementSection): string {
	return SECTION_LABEL_MAP.get(section) ?? section;
}

function safeParseEuroCents(rawInput: string): { cents: number; valid: boolean } {
	try {
		return {
			cents: parseEuroToCents(rawInput),
			valid: true
		};
	} catch {
		return {
			cents: 0,
			valid: false
		};
	}
}

export function evaluateJournalBalance(
	debitEuroInput: string,
	creditEuroInput: string
): { debitCents: number; creditCents: number; balanced: boolean } {
	const debit = safeParseEuroCents(debitEuroInput);
	const credit = safeParseEuroCents(creditEuroInput);
	return {
		debitCents: debit.cents,
		creditCents: credit.cents,
		balanced: debit.valid && credit.valid && debit.cents > 0 && debit.cents === credit.cents
	};
}

export function resolveAccountingContext(params: {
	entities: LegalEntityDTO[];
	selectedEntityId: string;
	url: URL;
}): { activeEntityId: string; fiscalYear: number } {
	const hasMatchingEntity =
		params.selectedEntityId !== 'ALL' &&
		params.entities.some((entity) => entity.id === params.selectedEntityId);
	const activeEntityId = hasMatchingEntity
		? params.selectedEntityId
		: (params.entities[0]?.id ?? '');

	const yearParam = Number.parseInt(params.url.searchParams.get('year') ?? '', 10);
	const fiscalYear =
		!Number.isNaN(yearParam) && yearParam >= 2000 && yearParam <= 2100
			? yearParam
			: new Date().getFullYear();

	return {
		activeEntityId,
		fiscalYear
	};
}
