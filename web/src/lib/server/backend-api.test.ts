import type { Cookies, RequestEvent } from '@sveltejs/kit';
import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '#lib/core/api/api-error';
import { createBackendApi } from './backend-api';
import { ACCESS_TOKEN_COOKIE, REFRESH_TOKEN_COOKIE, USER_COOKIE } from './cookies';

function createMockCookies(initial: Record<string, string> = {}): Cookies {
	const store = new Map<string, string>(Object.entries(initial));
	return {
		get: (name: string) => store.get(name),
		getAll: () => Array.from(store.entries()).map(([name, value]) => ({ name, value })),
		set: (name: string, value: string) => {
			store.set(name, value);
		},
		delete: (name: string) => {
			store.delete(name);
		},
		serialize: (name: string, value: string) => `${name}=${value}`
	} as unknown as Cookies;
}

function createMockEvent(options: {
	cookies?: Record<string, string>;
	fetchFn: typeof fetch;
	env?: Record<string, string>;
}): RequestEvent {
	const cookies = createMockCookies(options.cookies);
	return {
		cookies,
		fetch: options.fetchFn,
		url: new URL('https://immo.SaveWithYann.com/entities'),
		locals: {
			user: null,
			accessToken: null,
			api: null as unknown as RequestEvent['locals']['api']
		},
		platform: {
			env: {
				API_BASE_URL: 'https://api-immo.SaveWithYann.com',
				...options.env
			}
		}
	} as unknown as RequestEvent;
}

describe('createBackendApi (openapi-fetch)', () => {
	it('injects Authorization Bearer and Cloudflare Access headers on requests', async () => {
		const fetchMock = vi.fn<typeof fetch>().mockImplementation(async (input) => {
			const req = input instanceof Request ? input : new Request(input);
			expect(req.url).toBe('https://api-immo.savewithyann.com/api/v1/entities');
			expect(req.headers.get('Authorization')).toBe('Bearer access-token-123');
			expect(req.headers.get('CF-Access-Client-Id')).toBe('cf-id-1');
			expect(req.headers.get('CF-Access-Client-Secret')).toBe('cf-secret-1');

			return new Response(JSON.stringify([]), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			});
		});

		const event = createMockEvent({
			cookies: { [ACCESS_TOKEN_COOKIE]: 'access-token-123' },
			fetchFn: fetchMock,
			env: {
				CF_ACCESS_CLIENT_ID: 'cf-id-1',
				CF_ACCESS_CLIENT_SECRET: 'cf-secret-1'
			}
		});

		const api = createBackendApi(event);
		const entities = await api.portfolio.listEntities();

		expect(entities).toEqual([]);
		expect(fetchMock).toHaveBeenCalledTimes(1);
	});

	it('refreshes token once on 401 and retries original request with rotated token', async () => {
		const calls: Array<{ url: string; auth: string | null }> = [];
		const fetchMock = vi.fn<typeof fetch>().mockImplementation(async (input, init) => {
			const req = input instanceof Request ? input : new Request(input, init);
			calls.push({ url: req.url, auth: req.headers.get('Authorization') });

			if (req.url.endsWith('/api/v1/entities') && calls.length === 1) {
				return new Response(JSON.stringify({ error: 'token expired' }), {
					status: 401,
					headers: { 'Content-Type': 'application/json' }
				});
			}

			if (req.url.endsWith('/api/v1/auth/refresh')) {
				return new Response(
					JSON.stringify({
						access_token: 'rotated-access-token',
						refresh_token: 'rotated-refresh-token',
						token_type: 'Bearer',
						user: {
							id: 'usr_1',
							full_name: 'Admin User',
							email: 'admin@example.com',
							role: 'SUPER_ADMIN',
							status: 'APPROVED',
							created_at: '2026-01-01T00:00:00Z'
						}
					}),
					{
						status: 200,
						headers: { 'Content-Type': 'application/json' }
					}
				);
			}

			return new Response(
				JSON.stringify([
					{
						id: 'ent_1',
						name: 'SCI Alpha',
						entity_type: 'SCI_IS',
						siret: '12345678900010',
						naf_code: '6820B',
						address: '10 rue de la Paix',
						share_capital_cents: 100000,
						total_shares: 100,
						created_at: '2026-01-01T00:00:00Z',
						updated_at: '2026-01-01T00:00:00Z'
					}
				]),
				{
					status: 200,
					headers: { 'Content-Type': 'application/json' }
				}
			);
		});

		const event = createMockEvent({
			cookies: {
				[ACCESS_TOKEN_COOKIE]: 'expired-access-token',
				[REFRESH_TOKEN_COOKIE]: 'valid-refresh-token'
			},
			fetchFn: fetchMock
		});

		const api = createBackendApi(event);
		const entities = await api.portfolio.listEntities();

		expect(entities).toHaveLength(1);
		expect(entities[0].name).toBe('SCI Alpha');
		expect(calls).toEqual([
			{
				url: 'https://api-immo.savewithyann.com/api/v1/entities',
				auth: 'Bearer expired-access-token'
			},
			{
				url: 'https://api-immo.savewithyann.com/api/v1/auth/refresh',
				auth: null
			},
			{
				url: 'https://api-immo.savewithyann.com/api/v1/entities',
				auth: 'Bearer rotated-access-token'
			}
		]);
		expect(event.cookies.get(ACCESS_TOKEN_COOKIE)).toBe('rotated-access-token');
		expect(event.cookies.get(REFRESH_TOKEN_COOKIE)).toBe('rotated-refresh-token');
	});

	it('clears cookies and throws ApiError when refresh fails on 401', async () => {
		const fetchMock = vi.fn<typeof fetch>().mockImplementation(async (input, init) => {
			const req = input instanceof Request ? input : new Request(input, init);
			if (req.url.endsWith('/api/v1/auth/refresh')) {
				return new Response(JSON.stringify({ error: 'invalid refresh token' }), {
					status: 401,
					headers: { 'Content-Type': 'application/json' }
				});
			}
			return new Response(JSON.stringify({ error: 'unauthorized' }), {
				status: 401,
				headers: { 'Content-Type': 'application/json' }
			});
		});

		const event = createMockEvent({
			cookies: {
				[ACCESS_TOKEN_COOKIE]: 'expired-access-token',
				[REFRESH_TOKEN_COOKIE]: 'expired-refresh-token',
				[USER_COOKIE]: '{"id":"usr_1"}'
			},
			fetchFn: fetchMock
		});

		const api = createBackendApi(event);
		await expect(api.portfolio.listEntities()).rejects.toEqual(new ApiError(401, 'unauthorized'));
		expect(event.cookies.get(ACCESS_TOKEN_COOKIE)).toBeUndefined();
		expect(event.cookies.get(REFRESH_TOKEN_COOKIE)).toBeUndefined();
		expect(event.cookies.get(USER_COOKIE)).toBeUndefined();
	});

	it('throws ApiError with backend message on 422 validation error', async () => {
		const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
			new Response(
				JSON.stringify({
					status: 422,
					title: 'Unprocessable Entity',
					error: 'entity name is required'
				}),
				{
					status: 422,
					headers: { 'Content-Type': 'application/json' }
				}
			)
		);

		const event = createMockEvent({
			cookies: { [ACCESS_TOKEN_COOKIE]: 'access-token-123' },
			fetchFn: fetchMock
		});

		const api = createBackendApi(event);
		await expect(
			api.portfolio.createEntity({
				name: '',
				entity_type: 'SCI_IS'
			})
		).rejects.toEqual(new ApiError(422, 'entity name is required'));
	});
});
