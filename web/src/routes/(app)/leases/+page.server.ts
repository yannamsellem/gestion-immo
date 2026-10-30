import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import {
	formatCentsToEuro,
	parseEuroToCents,
	parsePercentToBasisPoints
} from '#lib/core/formatters/money';
import type { StoredFileDTO } from '#lib/modules/document/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, parent }) => {
	const parentData = await parent();
	const entities = parentData.entities;
	const selectedEntityId = parentData.selectedEntityId;

	const targetEntities =
		selectedEntityId === 'ALL'
			? entities
			: entities.filter((entity) => entity.id === selectedEntityId);

	try {
		const [propLists, leases, tenants, templates] = await Promise.all([
			Promise.all(targetEntities.map((entity) => locals.api.portfolio.listProperties(entity.id))),
			locals.api.lease.listLeases(selectedEntityId),
			locals.api.lease.listTenants(true),
			locals.api.document.listTemplates().catch(() => [])
		]);

		const documentsByLeaseId: Record<string, StoredFileDTO[]> = {};
		await Promise.all(
			leases.map(async (lease) => {
				const docs = await locals.api.document.listLeaseDocuments(lease.id).catch(() => []);
				documentsByLeaseId[lease.id] = docs;
			})
		);

		return {
			entities,
			properties: propLists.flat(),
			leases,
			tenants,
			templates,
			documentsByLeaseId,
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			entities,
			properties: [],
			leases: [],
			tenants: [],
			templates: [],
			documentsByLeaseId: {} as Record<string, StoredFileDTO[]>,
			errorMessage:
				err instanceof ApiError ? err.message : 'Impossible de charger les contrats de bail.'
		};
	}
};

export const actions: Actions = {
	createLease: async ({ request, locals }) => {
		const formData = await request.formData();
		const legalEntityId = String(formData.get('legal_entity_id') ?? '').trim();
		const lotId = String(formData.get('lot_id') ?? '').trim();
		const tenantId = String(formData.get('tenant_id') ?? '').trim();
		const leaseType = String(formData.get('lease_type') ?? 'RESIDENTIAL_UNFURNISHED').trim();
		const rentNetEuro = String(formData.get('rent_net_euro') ?? '0');
		const chargesEuro = String(formData.get('charges_euro') ?? '0');
		const depositEuro = String(formData.get('deposit_euro') ?? '0');
		const subjectToVat = formData.get('subject_to_vat') === 'true';
		const vatPercent = String(formData.get('vat_percent') ?? '20,00');
		const dueDay = Number.parseInt(String(formData.get('due_day') ?? '5'), 10) || 5;
		const gliPolicy = String(formData.get('gli_policy_number') ?? '').trim();
		const guarantor = String(formData.get('guarantor_name') ?? '').trim();
		const startDate = String(formData.get('start_date') ?? '').trim();

		if (!legalEntityId || !lotId || !tenantId) {
			return fail(400, {
				actionError: 'Veuillez sélectionner une entité, un lot et un locataire.'
			});
		}

		let rentNetCents: number;
		let chargesCents: number;
		let depositCents: number;
		let vatBasisPoints = 0;

		try {
			rentNetCents = parseEuroToCents(rentNetEuro);
			chargesCents = parseEuroToCents(chargesEuro || '0');
			depositCents = parseEuroToCents(depositEuro || '0');
			if (subjectToVat) {
				vatBasisPoints = parsePercentToBasisPoints(vatPercent);
			}
		} catch {
			return fail(400, {
				actionError: 'Montant de loyer, charges, dépôt ou taux de TVA invalide.'
			});
		}

		try {
			const created = await locals.api.lease.createLease({
				legal_entity_id: legalEntityId,
				lot_id: lotId,
				tenant_id: tenantId,
				lease_type: leaseType,
				rent_net_cents: rentNetCents,
				charges_cents: chargesCents,
				subject_to_vat: subjectToVat,
				vat_rate_basis_points: vatBasisPoints,
				security_deposit_cents: depositCents,
				due_day_of_month: dueDay,
				gli_policy_number: gliPolicy,
				guarantor_name: guarantor,
				start_date: startDate ? new Date(startDate).toISOString() : undefined
			});
			return {
				createdLease: created,
				leaseCreated: true,
				successMessage: 'Contrat de bail créé.'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Impossible de créer le bail.'
			});
		}
	},

	generateDoc: async ({ request, locals }) => {
		const formData = await request.formData();
		const leaseId = String(formData.get('lease_id') ?? '').trim();
		const legalEntityId = String(formData.get('legal_entity_id') ?? '').trim();
		const templateId = String(formData.get('template_id') ?? '').trim();
		const tenantName = String(formData.get('tenant_name') ?? 'Locataire');
		const entityName = String(formData.get('entity_name') ?? 'Bailleur');
		const propertyLabel = String(formData.get('property_label') ?? '');
		const rentNetCents = Number(formData.get('rent_net_cents') ?? 0);
		const chargesCents = Number(formData.get('charges_cents') ?? 0);

		if (!leaseId || !templateId) {
			return fail(400, { actionError: 'Veuillez sélectionner un modèle.' });
		}

		try {
			const generated = await locals.api.document.generateFromTemplate({
				template_id: templateId,
				legal_entity_id: legalEntityId || undefined,
				lease_id: leaseId,
				variables: {
					tenant_name: tenantName,
					entity_name: entityName,
					property_label: propertyLabel,
					rent_net: formatCentsToEuro(rentNetCents),
					charges: formatCentsToEuro(chargesCents)
				}
			});
			return {
				generatedDoc: generated,
				successMessage: `Document généré : ${generated.file_name}`
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Échec de la génération.'
			});
		}
	},

	uploadFile: async ({ request, locals }) => {
		const formData = await request.formData();
		const leaseId = String(formData.get('lease_id') ?? '').trim();
		const legalEntityId = String(formData.get('legal_entity_id') ?? '').trim();
		const category = String(formData.get('category') ?? 'BAIL_SIGNE').trim();
		const file = formData.get('file');

		if (!leaseId || !(file instanceof File) || file.size === 0) {
			return fail(400, { actionError: 'Veuillez sélectionner un fichier à téléverser.' });
		}

		try {
			const uploaded = await locals.api.document.uploadFile({
				legalEntityId: legalEntityId || undefined,
				leaseId,
				category,
				file
			});
			return {
				uploadedDoc: uploaded,
				successMessage: 'Pièce justificative téléversée dans le coffre-fort R2.'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Échec du téléversement.'
			});
		}
	}
};
