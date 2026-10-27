import { fail, redirect } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
	if (locals.user) {
		throw redirect(303, '/');
	}
	return {};
};

export const actions: Actions = {
	default: async ({ request, locals }) => {
		const formData = await request.formData();
		const fullName = String(formData.get('full_name') ?? '').trim();
		const email = String(formData.get('email') ?? '').trim();
		const password = String(formData.get('password') ?? '');

		if (!fullName || !email || password.length < 8) {
			return fail(400, {
				fullName,
				email,
				submittedEmail: null,
				errorMessage: 'Veuillez renseigner tous les champs (mot de passe de 8 caractères minimum).'
			});
		}

		try {
			const created = await locals.api.auth.signup({
				full_name: fullName,
				email,
				password
			});
			return {
				fullName: '',
				email: '',
				submittedEmail: created.email,
				errorMessage: null
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				fullName,
				email,
				submittedEmail: null,
				errorMessage:
					err instanceof ApiError ? err.message : "Impossible d'enregistrer votre demande."
			});
		}
	}
};
