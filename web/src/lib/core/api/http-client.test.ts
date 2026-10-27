import { describe, expect, it, vi } from 'vitest';
import { FetchHttpClient } from './http-client';
import { ApiError } from './api-error';

describe('FetchHttpClient', () => {
	it('attaches Bearer token and coalesces concurrent 401 refresh calls into a single flight', async () => {
		let accessToken = 'expired-token';
		let refreshCalls = 0;

		const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			const auth = (init?.headers as Record<string, string>)?.Authorization;
			if (url.endsWith('/api/v1/entities') && auth === 'Bearer expired-token') {
				return new Response(JSON.stringify({ error: 'token expired' }), { status: 401 });
			}
			return new Response(JSON.stringify([{ id: 'ent_1', name: 'SCI Alpha' }]), { status: 200 });
		});

		const client = new FetchHttpClient({
			baseUrl: 'http://localhost:8080',
			fetchFn: fetchMock as unknown as typeof fetch,
			getAccessToken: () => accessToken,
			refreshSession: async () => {
				refreshCalls += 1;
				await new Promise((r) => setTimeout(r, 10));
				accessToken = 'fresh-token';
				return accessToken;
			},
			onUnauthorized: vi.fn()
		});

		const [res1, res2] = await Promise.all([
			client.requestJson<Array<{ id: string }>>('/api/v1/entities'),
			client.requestJson<Array<{ id: string }>>('/api/v1/entities')
		]);

		expect(refreshCalls).toBe(1);
		expect(res1[0].id).toBe('ent_1');
		expect(res2[0].id).toBe('ent_1');
	});

	it('throws ApiError with backend message on non-2xx responses', async () => {
		const client = new FetchHttpClient({
			baseUrl: 'http://localhost:8080',
			fetchFn: vi.fn(
				async () =>
					new Response(JSON.stringify({ error: 'account is pending approval' }), { status: 403 })
			) as unknown as typeof fetch
		});

		await expect(client.requestJson('/api/v1/auth/login')).rejects.toThrowError(ApiError);
	});
});
