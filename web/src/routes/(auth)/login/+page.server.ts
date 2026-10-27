import { fail, redirect } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import { setSessionCookies } from '#lib/server/cookies';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
	if (locals.user) {
		throw redirect(303, '/');
	}
	return {};
};

export const actions: Actions = {
	default: async ({ request, locals, cookies, url }) => {
		const formData = await request.formData();
		const email = String(formData.get('email') ?? '').trim();
		const password = String(formData.get('password') ?? '');

		if (!email || !password) {
			return fail(400, {
				email,
				isPendingApproval: false,
				errorMessage: 'Veuillez saisir votre adresse e-mail et votre mot de passe.'
			});
		}

		try {
			const response = await locals.api.auth.login({ email, password });
			setSessionCookies(cookies, response, url.protocol === 'https:');
		} catch (err) {
			if (err instanceof ApiError && err.status === 403) {
				return fail(403, {
					email,
					isPendingApproval: true,
					errorMessage:
						'Votre compte est en attente de validation par un administrateur (SUPER_ADMIN).'
				});
			}
			return fail(err instanceof ApiError ? err.status : 500, {
				email,
				isPendingApproval: false,
				errorMessage: err instanceof ApiError ? err.message : 'Impossible de se connecter.'
			});
		}

		throw redirect(303, '/');
	}
};
