import type { AuthTokensResponse, UserDTO } from './types';

const AUTH_STORAGE_KEY = 'gestion_immo_auth_session';

interface StoredAuthSession {
	accessToken: string;
	refreshToken: string;
	user: UserDTO;
}

class AuthStore {
	user = $state<UserDTO | null>(null);
	accessToken = $state<string | null>(null);
	refreshToken = $state<string | null>(null);
	initialized = $state<boolean>(false);

	readonly isAuthenticated = $derived(Boolean(this.accessToken && this.user));
	readonly isSuperAdmin = $derived(this.user?.role === 'SUPER_ADMIN');

	initialize(): void {
		if (typeof window === 'undefined') {
			this.initialized = true;
			return;
		}
		try {
			const raw = window.localStorage.getItem(AUTH_STORAGE_KEY);
			if (raw) {
				const parsed = JSON.parse(raw) as StoredAuthSession;
				if (parsed.accessToken && parsed.user) {
					this.accessToken = parsed.accessToken;
					this.refreshToken = parsed.refreshToken;
					this.user = parsed.user;
				}
			}
		} catch {
			window.localStorage.removeItem(AUTH_STORAGE_KEY);
		} finally {
			this.initialized = true;
		}
	}

	setSession(tokens: AuthTokensResponse): void {
		this.accessToken = tokens.access_token;
		this.refreshToken = tokens.refresh_token;
		this.user = tokens.user;

		if (typeof window !== 'undefined') {
			const payload: StoredAuthSession = {
				accessToken: tokens.access_token,
				refreshToken: tokens.refresh_token,
				user: tokens.user
			};
			window.localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(payload));
		}
	}

	clearSession(): void {
		this.accessToken = null;
		this.refreshToken = null;
		this.user = null;

		if (typeof window !== 'undefined') {
			window.localStorage.removeItem(AUTH_STORAGE_KEY);
		}
	}
}

export const authStore = new AuthStore();
