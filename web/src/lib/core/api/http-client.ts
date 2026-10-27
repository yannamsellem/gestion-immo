import { ApiError } from './api-error';

export interface HttpClient {
	requestJson<T>(path: string, init?: RequestInit): Promise<T>;
	requestText(path: string, init?: RequestInit): Promise<string>;
	requestMultipart<T>(path: string, formData: FormData): Promise<T>;
}

export interface FetchHttpClientOptions {
	baseUrl?: string;
	fetchFn?: typeof fetch;
	getAccessToken?: () => string | null;
	refreshSession?: () => Promise<string | null>;
	onUnauthorized?: () => void;
}

export class FetchHttpClient implements HttpClient {
	private readonly baseUrl: string;
	private readonly fetchFn: typeof fetch;
	private readonly getAccessToken?: () => string | null;
	private readonly refreshSession?: () => Promise<string | null>;
	private readonly onUnauthorized?: () => void;
	private refreshPromise: Promise<string | null> | null = null;

	constructor(options: FetchHttpClientOptions = {}) {
		this.baseUrl = (options.baseUrl ?? '').replace(/\/$/, '');
		this.fetchFn = options.fetchFn ?? ((input, init) => fetch(input, init));
		this.getAccessToken = options.getAccessToken;
		this.refreshSession = options.refreshSession;
		this.onUnauthorized = options.onUnauthorized;
	}

	private normalizeHeaders(headersInit?: HeadersInit): Record<string, string> {
		const normalized: Record<string, string> = {};
		if (!headersInit) {
			return normalized;
		}
		if (headersInit instanceof Headers) {
			headersInit.forEach((value, key) => {
				normalized[key] = value;
			});
			return normalized;
		}
		if (Array.isArray(headersInit)) {
			for (const [key, value] of headersInit) {
				normalized[key] = value;
			}
			return normalized;
		}
		return { ...headersInit };
	}

	async requestJson<T>(path: string, init: RequestInit = {}): Promise<T> {
		const headers: Record<string, string> = {
			Accept: 'application/json',
			...this.normalizeHeaders(init.headers)
		};
		if (init.body && !headers['Content-Type']) {
			headers['Content-Type'] = 'application/json';
		}

		const response = await this.executeWithRetry(path, {
			...init,
			headers
		});

		if (response.status === 204) {
			return undefined as T;
		}

		const text = await response.text();
		if (!text) {
			return undefined as T;
		}
		return JSON.parse(text) as T;
	}

	async requestText(path: string, init: RequestInit = {}): Promise<string> {
		const response = await this.executeWithRetry(path, init);
		return response.text();
	}

	async requestMultipart<T>(path: string, formData: FormData): Promise<T> {
		const response = await this.executeWithRetry(path, {
			method: 'POST',
			body: formData
		});
		const text = await response.text();
		return (text ? JSON.parse(text) : undefined) as T;
	}

	private async executeWithRetry(
		path: string,
		init: RequestInit,
		retried = false
	): Promise<Response> {
		const url = path.startsWith('http') ? path : `${this.baseUrl}${path}`;
		const headers: Record<string, string> = this.normalizeHeaders(init.headers);

		const token = this.getAccessToken?.();
		if (token) {
			headers.Authorization = `Bearer ${token}`;
		}

		const response = await this.fetchFn(url, {
			...init,
			headers
		});

		if (
			response.status === 401 &&
			!retried &&
			this.refreshSession &&
			!path.includes('/api/v1/auth/')
		) {
			const freshToken = await this.coalesceRefresh();
			if (freshToken) {
				return this.executeWithRetry(path, init, true);
			}
			this.onUnauthorized?.();
		}

		if (!response.ok) {
			await this.throwApiError(response);
		}

		return response;
	}

	private async coalesceRefresh(): Promise<string | null> {
		if (!this.refreshSession) {
			return null;
		}
		if (!this.refreshPromise) {
			this.refreshPromise = this.refreshSession().finally(() => {
				this.refreshPromise = null;
			});
		}
		return this.refreshPromise;
	}

	private async throwApiError(response: Response): Promise<never> {
		let message = `HTTP ${response.status}`;
		let code: string | undefined;
		try {
			const text = await response.text();
			if (text) {
				const parsed = JSON.parse(text) as { error?: string; message?: string; code?: string };
				message = parsed.error ?? parsed.message ?? text;
				code = parsed.code;
			}
		} catch {
			// Keep fallback status message
		}
		throw new ApiError(response.status, message, code);
	}
}
