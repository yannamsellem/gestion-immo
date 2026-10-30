import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import { resolveAccountingContext } from '#lib/modules/accounting/statement-state';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, parent, url }) => {
	const parentData = await parent();
	const entities = parentData.entities;
	const { activeEntityId, fiscalYear } = resolveAccountingContext({
		entities,
		selectedEntityId: parentData.selectedEntityId,
		url
	});

	const disabledParam = url.searchParams.get('disabled') ?? '';
	const disabledLineIds = disabledParam ? disabledParam.split(',').filter(Boolean) : [];

	if (!activeEntityId) {
		return {
			entities,
			activeEntityId: '',
			fiscalYear,
			disabledLineIds,
			statements: null,
			errorMessage: null as string | null
		};
	}

	try {
		const statements = await locals.api.accounting.previewStatements(
			activeEntityId,
			fiscalYear,
			disabledLineIds
		);

		return {
			entities,
			activeEntityId,
			fiscalYear,
			disabledLineIds,
			statements,
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			entities,
			activeEntityId,
			fiscalYear,
			disabledLineIds,
			statements: null,
			errorMessage:
				err instanceof ApiError
					? err.message
					: 'Impossible de charger la synthèse fiscale de l’exercice.'
		};
	}
};

export const actions: Actions = {
	generatePdf: async ({ request, locals }) => {
		const formData = await request.formData();
		const entityId = String(formData.get('entity_id') ?? '').trim();
		const fiscalYear =
			Number.parseInt(String(formData.get('fiscal_year') ?? ''), 10) || new Date().getFullYear();
		const disabledJson = String(formData.get('disabled_line_ids_json') ?? '[]');

		if (!entityId) {
			return fail(400, { actionError: 'Aucune entité sélectionnée.' });
		}

		try {
			const disabledLineIds = JSON.parse(disabledJson) as string[];
			const generatedReport = await locals.api.accounting.generateAnnualAccountsPdf(
				entityId,
				fiscalYear,
				disabledLineIds
			);
			return {
				generatedReport,
				successMessage: `Plaquette des comptes annuels ${fiscalYear} générée.`
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError:
					err instanceof ApiError
						? err.message
						: 'Échec de la génération du PDF des comptes annuels.'
			});
		}
	}
};
