import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import { parseEuroToCents } from '#lib/core/formatters/money';
import type { IncidentDTO, IncidentStatus } from '#lib/modules/inspection/types';
import type { LeaseDTO, TenantDTO } from '#lib/modules/lease/types';
import type { PropertyDTO } from '#lib/modules/portfolio/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, parent }) => {
	const parentData = await parent();
	const allEntities = parentData.entities;
	const selectedEntityId = parentData.selectedEntityId;

	const targetEntities =
		selectedEntityId === 'ALL'
			? allEntities
			: allEntities.filter((entity) => entity.id === selectedEntityId);

	try {
		const [loadedIncidents, loadedLeases, tenants, propArrays] = await Promise.all([
			locals.api.inspection.listIncidents(),
			locals.api.lease.listLeases(selectedEntityId),
			locals.api.lease.listTenants(true),
			Promise.all(targetEntities.map((entity) => locals.api.portfolio.listProperties(entity.id)))
		]);

		const properties = propArrays.flat();
		const allowedPropertyIds = new Set(properties.map((property) => property.id));
		const allowedLeaseIds = new Set(loadedLeases.map((lease) => lease.id));

		const incidents =
			selectedEntityId === 'ALL'
				? loadedIncidents
				: loadedIncidents.filter(
						(incident) =>
							(incident.property_id && allowedPropertyIds.has(incident.property_id)) ||
							(incident.lease_id && allowedLeaseIds.has(incident.lease_id)) ||
							(!incident.property_id && !incident.lease_id)
					);

		return {
			incidents,
			properties,
			leases: loadedLeases.filter((lease) => lease.active),
			tenants,
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			incidents: [] as IncidentDTO[],
			properties: [] as PropertyDTO[],
			leases: [] as LeaseDTO[],
			tenants: [] as TenantDTO[],
			errorMessage:
				err instanceof ApiError ? err.message : 'Impossible de charger les incidents et sinistres.'
		};
	}
};

export const actions: Actions = {
	createIncident: async ({ request, locals }) => {
		const formData = await request.formData();
		const title = String(formData.get('title') ?? '').trim();
		const propertyId = String(formData.get('property_id') ?? '').trim();
		const leaseId = String(formData.get('lease_id') ?? '').trim();
		const claimNumber = String(formData.get('insurance_claim_number') ?? '').trim();
		const estimatedEuro = String(formData.get('estimated_cost_euro') ?? '0');
		const observation = String(formData.get('observation') ?? '').trim();

		if (!title) {
			return fail(400, { actionError: 'Veuillez renseigner l’intitulé de l’incident.' });
		}

		let estimatedCostCents: number;
		try {
			estimatedCostCents = parseEuroToCents(estimatedEuro || '0');
		} catch {
			return fail(400, { actionError: 'Montant estimé invalide.' });
		}

		try {
			const created = await locals.api.inspection.createIncident({
				property_id: propertyId || undefined,
				lease_id: leaseId || undefined,
				title,
				status: 'OPEN',
				insurance_claim_number: claimNumber || undefined,
				estimated_cost_cents: estimatedCostCents,
				observations: observation ? [observation] : []
			});
			return {
				createdIncident: created,
				incidentCreated: true,
				successMessage: 'Incident enregistré.'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Erreur lors de la création.'
			});
		}
	},

	updateIncident: async ({ request, locals }) => {
		const formData = await request.formData();
		const incidentId = String(formData.get('incident_id') ?? '').trim();
		const status = String(formData.get('status') ?? 'OPEN') as IncidentStatus;
		const claimNumber = String(formData.get('insurance_claim_number') ?? '').trim();
		const estimatedEuro = String(formData.get('estimated_cost_euro') ?? '0');
		const addObservation = String(formData.get('add_observation') ?? '').trim();

		if (!incidentId) {
			return fail(400, { actionError: 'Incident introuvable.' });
		}

		let estimatedCostCents: number;
		try {
			estimatedCostCents = parseEuroToCents(estimatedEuro || '0');
		} catch {
			return fail(400, { actionError: 'Coût estimé invalide.' });
		}

		try {
			const updated = await locals.api.inspection.updateIncident(incidentId, {
				status,
				insurance_claim_number: claimNumber || undefined,
				estimated_cost_cents: estimatedCostCents,
				add_observation: addObservation || undefined
			});
			return {
				updatedIncident: updated,
				successMessage: 'Incident mis à jour.'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError:
					err instanceof ApiError ? err.message : 'Impossible de mettre à jour l’incident.'
			});
		}
	}
};
