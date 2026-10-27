import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import type { UserDTO } from '#lib/modules/auth/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, parent }) => {
	const parentData = await parent();
	const user = parentData.user;
	const isSuperAdmin = user?.role === 'SUPER_ADMIN';

	if (!isSuperAdmin) {
		return {
			user,
			isSuperAdmin: false,
			pendingUsers: [] as UserDTO[],
			pendingError: null as string | null
		};
	}

	try {
		const pendingUsers = await locals.api.auth.listPendingUsers();
		return {
			user,
			isSuperAdmin: true,
			pendingUsers,
			pendingError: null as string | null
		};
	} catch (err) {
		return {
			user,
			isSuperAdmin: true,
			pendingUsers: [] as UserDTO[],
			pendingError:
				err instanceof ApiError ? err.message : 'Impossible de charger les comptes en attente.'
		};
	}
};

export const actions: Actions = {
	createMcpToken: async ({ request, locals }) => {
		const formData = await request.formData();
		const label = String(formData.get('label') ?? '').trim();

		if (!label) {
			return fail(400, { actionError: 'Veuillez saisir un libellé pour le jeton MCP.' });
		}

		try {
			const createdMcpToken = await locals.api.auth.createMcpToken({ label });
			return {
				createdMcpToken,
				successMessage: 'Jeton personnel MCP généré.'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Échec de la création du jeton MCP.'
			});
		}
	},

	approveUser: async ({ request, locals }) => {
		const formData = await request.formData();
		const userId = String(formData.get('user_id') ?? '').trim();
		const fullName = String(formData.get('full_name') ?? 'Collaborateur');

		if (!userId) {
			return fail(400, { actionError: 'Utilisateur introuvable.' });
		}

		try {
			await locals.api.auth.approveUser(userId);
			return {
				successMessage: `Compte de « ${fullName} » approuvé.`
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Impossible d’approuver ce compte.'
			});
		}
	},

	rejectUser: async ({ request, locals }) => {
		const formData = await request.formData();
		const userId = String(formData.get('user_id') ?? '').trim();
		const fullName = String(formData.get('full_name') ?? 'Collaborateur');

		if (!userId) {
			return fail(400, { actionError: 'Utilisateur introuvable.' });
		}

		try {
			await locals.api.auth.rejectUser(userId);
			return {
				successMessage: `Demande de « ${fullName} » rejetée.`
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Impossible de rejeter ce compte.'
			});
		}
	}
};
