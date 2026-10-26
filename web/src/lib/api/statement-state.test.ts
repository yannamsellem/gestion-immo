// web/src/lib/api/statement-state.test.ts
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { formatBasisPointsPct, formatEUR, formatVariationPct } from '../utils.ts';
import { ApiError, createApiClient } from './client.ts';
import { toggleStatementLine, toDisabledArray } from './statement-state.ts';

function expect<T>(actual: T) {
	return {
		toBe(expected: T): void {
			assert.strictEqual(actual, expected);
		},
		toEqual(expected: T): void {
			assert.deepStrictEqual(actual, expected);
		}
	};
}

describe('statement-state line toggling', () => {
	it('immutably adds and removes disabled line IDs for dynamic statement recalculation', () => {
		const initial = new Set<string>();
		const afterDisable = toggleStatementLine(initial, 'rev_comm');
		expect(initial.size).toBe(0);
		expect(toDisabledArray(afterDisable)).toEqual(['rev_comm']);

		const afterReenable = toggleStatementLine(afterDisable, 'rev_comm');
		expect(toDisabledArray(afterReenable)).toEqual([]);
	});

	it('normalizes whitespace, ignores empty line IDs, and returns sorted disabled line IDs', () => {
		const initial = new Set<string>(['exp_property_tax']);
		const unchanged = toggleStatementLine(initial, '   ');
		expect(toDisabledArray(unchanged)).toEqual(['exp_property_tax']);

		const withSecond = toggleStatementLine(unchanged, '  rev_residential_rent  ');
		expect(toDisabledArray(withSecond)).toEqual(['exp_property_tax', 'rev_residential_rent']);
	});
});

describe('API client contract & 403 ACCOUNT_NOT_APPROVED handling', () => {
	it('maps 403 account not approved login error to ApiError with ACCOUNT_NOT_APPROVED code', async () => {
		const mockFetch: typeof fetch = async () =>
			new Response(JSON.stringify({ error: 'account is not approved by super admin' }), {
				status: 403,
				headers: { 'Content-Type': 'application/json' }
			});

		const api = createApiClient({ baseUrl: 'http://localhost:8080', fetchFn: mockFetch });
		await assert.rejects(
			() => api.login({ email: 'pending@investissimo.fr', password: 'secret-password' }),
			(err: unknown) => {
				assert.ok(err instanceof ApiError);
				assert.strictEqual(err.status, 403);
				assert.strictEqual(err.code, 'ACCOUNT_NOT_APPROVED');
				return true;
			}
		);
	});

	it('sends disabled_line_ids array to POST /api/v1/entities/{id}/statements/preview', async () => {
		let capturedUrl = '';
		let capturedBody = '';
		const mockFetch: typeof fetch = async (input, init) => {
			capturedUrl = String(input);
			capturedBody = String(init?.body ?? '');
			return new Response(
				JSON.stringify({
					lines: [],
					active_lines: [],
					disabled_line_ids: ['rev_comm'],
					net_income_n_cents: 120000
				}),
				{ status: 200, headers: { 'Content-Type': 'application/json' } }
			);
		};

		const api = createApiClient({
			baseUrl: 'http://localhost:8080',
			accessToken: 'jwt_test_token',
			fetchFn: mockFetch
		});
		const disabled = toggleStatementLine(new Set<string>(), 'rev_comm');
		const res = await api.previewFinancialStatements(
			'ent_sarl_investissimo',
			2026,
			toDisabledArray(disabled)
		);

		expect(capturedUrl).toBe(
			'http://localhost:8080/api/v1/entities/ent_sarl_investissimo/statements/preview'
		);
		expect(JSON.parse(capturedBody)).toEqual({
			fiscal_year: 2026,
			disabled_line_ids: ['rev_comm']
		});
		expect(res.disabled_line_ids).toEqual(['rev_comm']);
		expect(res.net_income_n_cents).toBe(120000);
	});
});

describe('Frontend monetary & basis-points formatting (AmountCents -> UI)', () => {
	it('formats integer euro cents and basis points in French locale', () => {
		assert.ok(formatEUR(120000).includes('200,00'));
		assert.ok(formatEUR(120000).includes('€'));
		assert.strictEqual(formatBasisPointsPct(5000), '50,00 %');
		assert.strictEqual(formatBasisPointsPct(0, true), 'N/S');
		assert.strictEqual(formatVariationPct(500000, 0), 'N/S');
		assert.strictEqual(formatVariationPct(110000, 100000), '10,00 %');
	});
});
