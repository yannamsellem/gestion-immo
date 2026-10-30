import { describe, expect, it } from 'vitest';
import type { LegalEntityDTO } from '#lib/modules/portfolio/types';
import {
	evaluateJournalBalance,
	formatStatementSectionLabel,
	resolveAccountingContext,
	toggleDisabledLineId
} from './statement-state';

const SAMPLE_ENTITIES: LegalEntityDTO[] = [
	{
		id: 'ent_1',
		name: 'SCI Rivoli',
		entity_type: 'SCI_IR',
		siret: '12345678900011',
		naf_code: '6820A',
		address: '12 rue de Rivoli, Paris',
		share_capital_cents: 100000,
		total_shares: 100,
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:00Z'
	},
	{
		id: 'ent_2',
		name: 'SARL Vendôme',
		entity_type: 'SARL_FAMILLE',
		siret: '98765432100022',
		naf_code: '6820A',
		address: '8 place Vendôme, Paris',
		share_capital_cents: 500000,
		total_shares: 500,
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:00Z'
	}
];

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

describe('evaluateJournalBalance', () => {
	it('marks equal positive debit and credit amounts as balanced', () => {
		expect(evaluateJournalBalance('1 200,00', '1200,00')).toEqual({
			debitCents: 120000,
			creditCents: 120000,
			balanced: true
		});
	});

	it('rejects zero-amount journal entries as unbalanced', () => {
		expect(evaluateJournalBalance('0,00', '0,00')).toEqual({
			debitCents: 0,
			creditCents: 0,
			balanced: false
		});
	});

	it('never throws on partial or malformed keystrokes and returns balanced: false', () => {
		expect(evaluateJournalBalance('120,', '120,00')).toEqual({
			debitCents: 0,
			creditCents: 12000,
			balanced: false
		});
		expect(evaluateJournalBalance('abc', '100')).toEqual({
			debitCents: 0,
			creditCents: 10000,
			balanced: false
		});
		expect(evaluateJournalBalance('', '')).toEqual({
			debitCents: 0,
			creditCents: 0,
			balanced: false
		});
	});
});

describe('resolveAccountingContext', () => {
	it('falls back to entities[0].id when selectedEntityId is ALL and parses valid ?year=', () => {
		const result = resolveAccountingContext({
			entities: SAMPLE_ENTITIES,
			selectedEntityId: 'ALL',
			url: new URL('http://localhost:5173/accounting?year=2025')
		});
		expect(result).toEqual({
			activeEntityId: 'ent_1',
			fiscalYear: 2025
		});
	});

	it('preserves a matching selectedEntityId and defaults out-of-range years to current year', () => {
		const currentYear = new Date().getFullYear();
		const result = resolveAccountingContext({
			entities: SAMPLE_ENTITIES,
			selectedEntityId: 'ent_2',
			url: new URL('http://localhost:5173/accounting/journal?year=1850')
		});
		expect(result).toEqual({
			activeEntityId: 'ent_2',
			fiscalYear: currentYear
		});
	});
});
