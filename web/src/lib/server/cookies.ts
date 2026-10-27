import type { Cookies } from '@sveltejs/kit';
import { getCurrentPeriodMonth } from '#lib/core/formatters/date';
import type { AuthTokensResponse, UserDTO } from '#lib/modules/auth/types';

export const ACCESS_TOKEN_COOKIE = 'gi_access_token';
export const REFRESH_TOKEN_COOKIE = 'gi_refresh_token';
export const USER_COOKIE = 'gi_user';
export const WORKSPACE_ENTITY_COOKIE = 'gi_entity_id';
export const WORKSPACE_PERIOD_COOKIE = 'gi_period_month';

export interface ServerSession {
	accessToken: string | null;
	refreshToken: string | null;
	user: UserDTO | null;
}

export function getSessionFromCookies(cookies: Cookies): ServerSession {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE) ?? null;
	const refreshToken = cookies.get(REFRESH_TOKEN_COOKIE) ?? null;
	const rawUser = cookies.get(USER_COOKIE);
	let user: UserDTO | null = null;

	if (rawUser) {
		try {
			user = JSON.parse(rawUser) as UserDTO;
		} catch {
			user = null;
		}
	}

	if (!accessToken && !refreshToken) {
		return { accessToken: null, refreshToken: null, user: null };
	}

	return { accessToken, refreshToken, user };
}

export function setSessionCookies(
	cookies: Cookies,
	session: AuthTokensResponse,
	isSecure: boolean
): void {
	const common = {
		path: '/',
		httpOnly: true,
		sameSite: 'lax' as const,
		secure: isSecure
	};

	cookies.set(ACCESS_TOKEN_COOKIE, session.access_token, {
		...common,
		maxAge: 60 * 60 * 24 * 7
	});
	cookies.set(REFRESH_TOKEN_COOKIE, session.refresh_token, {
		...common,
		maxAge: 60 * 60 * 24 * 30
	});
	cookies.set(USER_COOKIE, JSON.stringify(session.user), {
		...common,
		maxAge: 60 * 60 * 24 * 30
	});
}

export function clearSessionCookies(cookies: Cookies): void {
	cookies.delete(ACCESS_TOKEN_COOKIE, { path: '/' });
	cookies.delete(REFRESH_TOKEN_COOKIE, { path: '/' });
	cookies.delete(USER_COOKIE, { path: '/' });
}

export function getWorkspaceFromCookies(cookies: Cookies): {
	selectedEntityId: 'ALL' | string;
	selectedPeriodMonth: string;
} {
	const rawEntity = cookies.get(WORKSPACE_ENTITY_COOKIE);
	const rawPeriod = cookies.get(WORKSPACE_PERIOD_COOKIE);

	const selectedEntityId = rawEntity && rawEntity.trim() !== '' ? rawEntity.trim() : 'ALL';
	const selectedPeriodMonth =
		rawPeriod && /^\d{4}-\d{2}$/.test(rawPeriod) ? rawPeriod : getCurrentPeriodMonth();

	return { selectedEntityId, selectedPeriodMonth };
}

export function setWorkspaceCookies(
	cookies: Cookies,
	params: { selectedEntityId?: string; selectedPeriodMonth?: string },
	isSecure: boolean
): void {
	const common = {
		path: '/',
		httpOnly: false,
		sameSite: 'lax' as const,
		secure: isSecure,
		maxAge: 60 * 60 * 24 * 365
	};

	if (params.selectedEntityId && params.selectedEntityId.trim() !== '') {
		cookies.set(WORKSPACE_ENTITY_COOKIE, params.selectedEntityId.trim(), common);
	}
	if (params.selectedPeriodMonth && /^\d{4}-\d{2}$/.test(params.selectedPeriodMonth)) {
		cookies.set(WORKSPACE_PERIOD_COOKIE, params.selectedPeriodMonth, common);
	}
}
