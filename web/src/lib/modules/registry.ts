import { FetchHttpClient } from '#lib/core/api/http-client';
import { AccountingApiService } from './accounting/api';
import { AuthApiService } from './auth/api';
import { authStore } from './auth/auth-store.svelte';
import { DocumentApiService } from './document/api';
import { InspectionApiService } from './inspection/api';
import { LeaseApiService } from './lease/api';
import { PaymentApiService } from './payment/api';
import { PortfolioApiService } from './portfolio/api';

const apiBaseUrl =
	(typeof import.meta !== 'undefined' &&
		(import.meta as unknown as { env?: Record<string, string | undefined> }).env
			?.PUBLIC_API_BASE_URL) ||
	'';

const rawAuthHttpClient = new FetchHttpClient({ baseUrl: apiBaseUrl });
const rawAuthService = new AuthApiService(rawAuthHttpClient);

export const httpClient = new FetchHttpClient({
	baseUrl: apiBaseUrl,
	getAccessToken: () => authStore.accessToken,
	refreshSession: async () => {
		const refreshToken = authStore.refreshToken;
		if (!refreshToken) {
			return null;
		}
		try {
			const refreshed = await rawAuthService.refresh(refreshToken);
			authStore.setSession(refreshed);
			return refreshed.access_token;
		} catch {
			authStore.clearSession();
			return null;
		}
	},
	onUnauthorized: () => {
		authStore.clearSession();
	}
});

export const apiRegistry = {
	auth: new AuthApiService(httpClient),
	portfolio: new PortfolioApiService(httpClient),
	lease: new LeaseApiService(httpClient),
	payment: new PaymentApiService(httpClient),
	document: new DocumentApiService(httpClient),
	inspection: new InspectionApiService(httpClient),
	accounting: new AccountingApiService(httpClient)
} as const;
