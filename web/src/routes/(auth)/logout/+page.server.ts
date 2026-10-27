import { redirect } from '@sveltejs/kit';
import { clearSessionCookies, getSessionFromCookies } from '#lib/server/cookies';
import type { Actions } from './$types';

export const actions: Actions = {
	default: async ({ locals, cookies }) => {
		const session = getSessionFromCookies(cookies);
		if (session.refreshToken) {
			try {
				await locals.api.auth.logout(session.refreshToken);
			} catch {
				// Proceed with cookie cleanup even if backend token expired
			}
		}
		clearSessionCookies(cookies);
		locals.user = null;
		throw redirect(303, '/login');
	}
};
