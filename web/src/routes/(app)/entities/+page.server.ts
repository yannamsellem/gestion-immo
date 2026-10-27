import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import { parseEuroToCents } from '#lib/core/formatters/money';
import type { EntityPortfolioSummaryDTO } from '#lib/modules/portfolio/types';
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
		const summariesByEntity: Record<string, EntityPortfolioSummaryDTO> = {};
		await Promise.all(
			targetEntities.map(async (entity) => {
				const summary = await locals.api.portfolio.getEntitySummary(entity.id);
				summariesByEntity[entity.id] = summary;
			})
		);

		return {
			entities,
			summariesByEntity,
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			entities,
			summariesByEntity: {},
			errorMessage:
				err instanceof ApiError
					? err.message
					: 'Impossible de charger les entités juridiques et associés.'
		};
	}
};

export const actions: Actions = {
	createEntity: async ({ request, locals }) => {
		const formData = await request.formData();
		const name = String(formData.get('name') ?? '').trim();
		const entityType = String(formData.get('entity_type') ?? 'SCI_IS').trim();
		const siret = String(formData.get('siret') ?? '').trim();
		const nafCode = String(formData.get('naf_code') ?? '6820B').trim() || '6820B';
		const address = String(formData.get('address') ?? '').trim();
		const shareCapitalEuro = String(formData.get('share_capital_euro') ?? '0');
		const totalShares = Number.parseInt(String(formData.get('total_shares') ?? '100'), 10) || 100;

		if (!name || !address) {
			return fail(400, {
				actionError: 'Veuillez renseigner la dénomination sociale et l’adresse du siège.'
			});
		}

		let shareCapitalCents: number;
		try {
			shareCapitalCents = parseEuroToCents(shareCapitalEuro || '0');
		} catch {
			return fail(400, { actionError: 'Montant du capital social invalide.' });
		}

		try {
			const created = await locals.api.portfolio.createEntity({
				name,
				entity_type: entityType,
				siret,
				naf_code: nafCode,
				address,
				share_capital_cents: shareCapitalCents,
				total_shares: totalShares
			});
			return {
				createdEntity: created,
				successMessage: `Entité « ${created.name} » créée.`
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Échec de la création de l’entité.'
			});
		}
	},

	upsertShareholder: async ({ request, locals }) => {
		const formData = await request.formData();
		const entityId = String(formData.get('legal_entity_id') ?? '').trim();
		const fullName = String(formData.get('full_name') ?? '').trim();
		const sharesCount = Number.parseInt(String(formData.get('shares_count') ?? '0'), 10) || 0;
		const currentAccountEuro = String(formData.get('current_account_euro') ?? '0');
		const remunerationEuro = String(formData.get('remuneration_euro') ?? '0');

		if (!entityId || !fullName) {
			return fail(400, {
				actionError: 'Veuillez sélectionner une entité et saisir le nom de l’associé.'
			});
		}

		let currentAccountCents: number;
		let remunerationPaidCents: number;
		try {
			currentAccountCents = parseEuroToCents(currentAccountEuro || '0');
			remunerationPaidCents = parseEuroToCents(remunerationEuro || '0');
		} catch {
			return fail(400, {
				actionError: 'Montant compte courant ou rémunération invalide.'
			});
		}

		try {
			await locals.api.portfolio.upsertShareholder(entityId, {
				full_name: fullName,
				shares_count: sharesCount,
				current_account_cents: currentAccountCents,
				remuneration_paid_cents: remunerationPaidCents
			});
			return {
				shareholderSaved: true,
				successMessage: 'Associé enregistré (quote-part recalculée en points de base).'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Impossible d’enregistrer l’associé.'
			});
		}
	}
};
