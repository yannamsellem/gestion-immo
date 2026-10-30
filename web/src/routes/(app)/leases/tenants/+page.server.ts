import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
	try {
		const [tenants, leases] = await Promise.all([
			locals.api.lease.listTenants(true),
			locals.api.lease.listLeases('ALL')
		]);

		const activeLeaseCountByTenantId: Record<string, number> = {};
		for (const lease of leases) {
			if (lease.active) {
				activeLeaseCountByTenantId[lease.tenant_id] =
					(activeLeaseCountByTenantId[lease.tenant_id] ?? 0) + 1;
			}
		}

		return {
			tenants,
			activeLeaseCountByTenantId,
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			tenants: [],
			activeLeaseCountByTenantId: {} as Record<string, number>,
			errorMessage:
				err instanceof ApiError ? err.message : 'Impossible de charger les fiches locataires.'
		};
	}
};

export const actions: Actions = {
	createTenant: async ({ request, locals }) => {
		const formData = await request.formData();
		const fullName = String(formData.get('full_name') ?? '').trim();
		const email = String(formData.get('email') ?? '').trim();
		const phone = String(formData.get('phone') ?? '').trim();
		const address = String(formData.get('address') ?? '').trim();

		if (!fullName || !email) {
			return fail(400, {
				actionError: 'Veuillez renseigner le nom complet et l’adresse e-mail du locataire.'
			});
		}

		try {
			const created = await locals.api.lease.createTenant({
				full_name: fullName,
				email,
				phone,
				address
			});
			return {
				createdTenant: created,
				successMessage: `Locataire « ${created.full_name} » enregistré.`
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Échec de la création du locataire.'
			});
		}
	},

	archiveTenant: async ({ request, locals }) => {
		const formData = await request.formData();
		const tenantId = String(formData.get('tenant_id') ?? '').trim();
		const tenantName = String(formData.get('tenant_name') ?? 'Locataire');

		if (!tenantId) {
			return fail(400, { actionError: 'Locataire introuvable.' });
		}

		try {
			await locals.api.lease.archiveTenant(tenantId);
			return {
				successMessage: `Locataire « ${tenantName} » archivé (conformité RGPD).`
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Impossible d’archiver ce locataire.'
			});
		}
	}
};
