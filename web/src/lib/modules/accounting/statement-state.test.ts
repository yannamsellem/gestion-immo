import { describe, expect, it } from 'vitest';
import { formatStatementSectionLabel, toggleDisabledLineId } from './statement-state';

describe('toggleDisabledLineId', () => {
	it('adds and removes line IDs deterministically without duplicates', () => {
		const step1 = toggleDisabledLineId([], 'acct_681100');
		expect(step1).toEqual(['acct_681100']);

		const step2 = toggleDisabledLineId(step1, 'acct_615000');
		expect(step2).toEqual(['acct_681100', 'acct_615000']);

		const step3 = toggleDisabledLineId(step2, 'acct_681100');
		expect(step3).toEqual(['acct_615000']);
	});

	it('maps statement sections to French PCG/CERFA labels', () => {
		expect(formatStatementSectionLabel('OPERATING_REVENUE')).toBe("Produits d'exploitation");
		expect(formatStatementSectionLabel('DEPRECIATION')).toBe('Dotations aux amortissements');
	});
});
