import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import { parseEuroToCents } from '#lib/core/formatters/money';
import type {
	IncidentStatus,
	InspectionItemInput,
	InspectionType
} from '#lib/modules/inspection/types';
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
		const [loadedIncidents, templates, loadedLeases, tenants, propArrays] = await Promise.all([
			locals.api.inspection.listIncidents(),
			locals.api.inspection.listTemplates(),
			locals.api.lease.listLeases(selectedEntityId),
			locals.api.lease.listTenants(true),
			Promise.all(targetEntities.map((entity) => locals.api.portfolio.listProperties(entity.id)))
		]);

		const properties = propArrays.flat();
		const allowedPropertyIds = new Set(properties.map((p) => p.id));
		const allowedLeaseIds = new Set(loadedLeases.map((l) => l.id));

		const incidents =
			selectedEntityId === 'ALL'
				? loadedIncidents
				: loadedIncidents.filter(
						(inc) =>
							(inc.property_id && allowedPropertyIds.has(inc.property_id)) ||
							(inc.lease_id && allowedLeaseIds.has(inc.lease_id)) ||
							(!inc.property_id && !inc.lease_id)
					);

		const activeLeases = loadedLeases.filter((l) => l.active);
		const inspectionLists = await Promise.all(
			activeLeases.map((lease) => locals.api.inspection.listInspections(lease.id).catch(() => []))
		);

		return {
			incidents,
			templates,
			inspections: inspectionLists.flat(),
			properties,
			leases: activeLeases,
			tenants,
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			incidents: [],
			templates: [],
			inspections: [],
			properties: [],
			leases: [],
			tenants: [],
			errorMessage:
				err instanceof ApiError
					? err.message
					: 'Impossible de charger les incidents et états des lieux.'
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
			await locals.api.inspection.createIncident({
				property_id: propertyId || undefined,
				lease_id: leaseId || undefined,
				title,
				status: 'OPEN',
				insurance_claim_number: claimNumber || undefined,
				estimated_cost_cents: estimatedCostCents,
				observations: observation ? [observation] : []
			});
			return {
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
	},

	createTemplate: async ({ request, locals }) => {
		const formData = await request.formData();
		const name = String(formData.get('name') ?? '').trim();
		const roomsText = String(formData.get('rooms_text') ?? '');
		const elementsText = String(formData.get('elements_text') ?? '');

		const rooms = roomsText
			.split(',')
			.map((s) => s.trim())
			.filter(Boolean);
		const defaultElements = elementsText
			.split(',')
			.map((s) => s.trim())
			.filter(Boolean);

		if (!name || rooms.length === 0) {
			return fail(400, {
				actionError: 'Veuillez renseigner un nom et au moins une pièce.'
			});
		}

		try {
			const created = await locals.api.inspection.createTemplate({
				name,
				rooms,
				default_elements: defaultElements
			});
			return {
				createdTemplate: created,
				successMessage: 'Modèle d’état des lieux créé.'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Erreur lors de la création du modèle.'
			});
		}
	},

	createInspection: async ({ request, locals }) => {
		const formData = await request.formData();
		const leaseId = String(formData.get('lease_id') ?? '').trim();
		const templateId = String(formData.get('template_id') ?? '').trim();
		const type = String(formData.get('type') ?? 'MOVE_IN') as InspectionType;
		const entryInspectionId = String(formData.get('entry_inspection_id') ?? '').trim();
		const keysCount = Number.parseInt(String(formData.get('keys_count') ?? '0'), 10) || 0;
		const keysDesc = String(formData.get('keys_description') ?? '').trim();

		if (!leaseId || !templateId) {
			return fail(400, {
				actionError: 'Sélectionnez un bail et un modèle d’état des lieux.'
			});
		}

		try {
			const created = await locals.api.inspection.createInspection({
				lease_id: leaseId,
				template_id: templateId,
				type,
				entry_inspection_id:
					type === 'MOVE_OUT' && entryInspectionId ? entryInspectionId : undefined,
				keys_handed_over_count: keysCount,
				keys_description: keysDesc || undefined
			});
			const typeLabel = created.type === 'MOVE_IN' ? 'd’entrée' : 'de sortie';
			return {
				createdInspection: created,
				successMessage: `État des lieux ${typeLabel} initialisé.`
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Impossible de créer l’état des lieux.'
			});
		}
	},

	saveInspectionItems: async ({ request, locals }) => {
		const formData = await request.formData();
		const inspectionId = String(formData.get('inspection_id') ?? '').trim();
		const itemsJson = String(formData.get('items_json') ?? '[]');

		if (!inspectionId) {
			return fail(400, { actionError: 'État des lieux introuvable.' });
		}

		let items: InspectionItemInput[];
		try {
			items = JSON.parse(itemsJson) as InspectionItemInput[];
		} catch {
			return fail(400, { actionError: 'Format de grille invalide.' });
		}

		try {
			const updated = await locals.api.inspection.upsertInspectionItems(inspectionId, items);
			return {
				updatedInspection: updated,
				successMessage: 'Relevé pièce par pièce enregistré.'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError:
					err instanceof ApiError ? err.message : 'Impossible d’enregistrer les éléments.'
			});
		}
	},

	generatePdf: async ({ request, locals }) => {
		const formData = await request.formData();
		const inspectionId = String(formData.get('inspection_id') ?? '').trim();
		const itemsJson = String(formData.get('items_json') ?? '[]');

		if (!inspectionId) {
			return fail(400, { actionError: 'État des lieux introuvable.' });
		}

		try {
			const items = JSON.parse(itemsJson) as InspectionItemInput[];
			const updated = await locals.api.inspection.upsertInspectionItems(inspectionId, items);
			const report = await locals.api.inspection.generateInspectionPdf(inspectionId);
			return {
				updatedInspection: updated,
				generatedReport: report,
				successMessage: 'Rapport PDF contradictoire généré.'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError:
					err instanceof ApiError ? err.message : 'Échec de la génération du rapport PDF.'
			});
		}
	}
};
