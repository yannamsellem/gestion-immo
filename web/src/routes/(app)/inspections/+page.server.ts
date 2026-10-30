import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import type {
	InspectionDTO,
	InspectionItemInput,
	InspectionTemplateDTO,
	InspectionType
} from '#lib/modules/inspection/types';
import type { LeaseDTO, TenantDTO } from '#lib/modules/lease/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, parent }) => {
	const parentData = await parent();
	const selectedEntityId = parentData.selectedEntityId;

	try {
		const [templates, loadedLeases, tenants] = await Promise.all([
			locals.api.inspection.listTemplates(),
			locals.api.lease.listLeases(selectedEntityId),
			locals.api.lease.listTenants(true)
		]);

		const activeLeases = loadedLeases.filter((lease) => lease.active);
		const inspectionLists = await Promise.all(
			activeLeases.map((lease) => locals.api.inspection.listInspections(lease.id).catch(() => []))
		);
		const inspections = inspectionLists
			.flat()
			.sort((a, b) => b.inspection_date.localeCompare(a.inspection_date));

		return {
			inspections,
			templates,
			leases: activeLeases,
			tenants,
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			inspections: [] as InspectionDTO[],
			templates: [] as InspectionTemplateDTO[],
			leases: [] as LeaseDTO[],
			tenants: [] as TenantDTO[],
			errorMessage:
				err instanceof ApiError ? err.message : 'Impossible de charger les états des lieux.'
		};
	}
};

export const actions: Actions = {
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
