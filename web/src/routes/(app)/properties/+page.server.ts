import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import { parseEuroToCents } from '#lib/core/formatters/money';
import type { AccountingCategory } from '#lib/modules/portfolio/types';
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
		const [propLists, loadedLeases] = await Promise.all([
			Promise.all(targetEntities.map((entity) => locals.api.portfolio.listProperties(entity.id))),
			locals.api.lease.listLeases(selectedEntityId)
		]);

		return {
			entities,
			properties: propLists.flat(),
			leases: loadedLeases.filter((l) => l.active),
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			entities,
			properties: [],
			leases: [],
			errorMessage:
				err instanceof ApiError
					? err.message
					: 'Impossible de charger les biens et lots immobiliers.'
		};
	}
};

export const actions: Actions = {
	createProperty: async ({ request, locals }) => {
		const formData = await request.formData();
		const entityId = String(formData.get('legal_entity_id') ?? '').trim();
		const label = String(formData.get('label') ?? '').trim();
		const propertyType = String(formData.get('property_type') ?? 'BUILDING').trim();
		const accountingCategory = String(
			formData.get('accounting_category') ?? 'FIXED_ASSET'
		) as AccountingCategory;
		const address = String(formData.get('address') ?? '').trim();
		const acquisitionEuro = String(formData.get('acquisition_euro') ?? '0');
		const workStockEuro = String(formData.get('work_stock_euro') ?? '0');

		if (!entityId || !label || !address) {
			return fail(400, {
				actionError: 'Veuillez renseigner l’entité, la désignation et l’adresse du bien.'
			});
		}

		let acquisitionCents: number;
		let workStockCents: number;
		try {
			acquisitionCents = parseEuroToCents(acquisitionEuro || '0');
			workStockCents = parseEuroToCents(workStockEuro || '0');
		} catch {
			return fail(400, {
				actionError: 'Montant d’acquisition ou de stock travaux invalide.'
			});
		}

		try {
			const created = await locals.api.portfolio.createProperty(entityId, {
				label,
				property_type: propertyType,
				accounting_category: accountingCategory,
				address,
				acquisition_cents: acquisitionCents,
				work_stock_cents: workStockCents
			});
			return {
				createdProperty: created,
				successMessage: `Bien « ${created.label} » enregistré.`
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Échec de la création du bien.'
			});
		}
	},

	createLot: async ({ request, locals }) => {
		const formData = await request.formData();
		const propertyId = String(formData.get('property_id') ?? '').trim();
		const label = String(formData.get('label') ?? '').trim();
		const lotType = String(formData.get('lot_type') ?? 'APARTMENT').trim();
		const surfaceStr = String(formData.get('surface_m2') ?? '0');
		const ownershipShare =
			Number.parseInt(String(formData.get('ownership_share') ?? '100'), 10) || 100;
		const floor = String(formData.get('floor') ?? '').trim();

		if (!propertyId || !label) {
			return fail(400, {
				actionError: 'Veuillez sélectionner un bien et renseigner la référence du lot.'
			});
		}

		try {
			const created = await locals.api.portfolio.createLot(propertyId, {
				label,
				lot_type: lotType,
				surface_m2: Math.round(Number.parseFloat(surfaceStr.replace(',', '.'))) || 0,
				ownership_share: ownershipShare,
				floor
			});
			return {
				createdLot: created,
				successMessage: `Lot « ${created.label} » ajouté.`
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Échec de la création du lot.'
			});
		}
	}
};
