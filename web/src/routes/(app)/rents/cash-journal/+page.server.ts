import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import { parseEuroToCents } from '#lib/core/formatters/money';
import type { CashJournalEntryDTO } from '#lib/modules/payment/types';
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
		const entryLists = await Promise.all(
			targetEntities.map((entity) => locals.api.payment.listCashJournal(entity.id))
		);

		const cashEntries: CashJournalEntryDTO[] = entryLists
			.flat()
			.sort((a, b) => b.entry_date.localeCompare(a.entry_date));

		return {
			entities,
			cashEntries,
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			entities,
			cashEntries: [] as CashJournalEntryDTO[],
			errorMessage:
				err instanceof ApiError
					? err.message
					: 'Impossible de charger le livre de caisse espèces (PCG 530).'
		};
	}
};

export const actions: Actions = {
	cashDeposit: async ({ request, locals }) => {
		const formData = await request.formData();
		const entityId = String(formData.get('legal_entity_id') ?? '').trim();
		const amountEuro = String(formData.get('amount_euro') ?? '');
		const label =
			String(formData.get('label') ?? '').trim() || 'Remise espèces en banque (530 → 512)';
		const entryDate = String(formData.get('entry_date') ?? '').trim();

		let amountCents: number;
		try {
			amountCents = parseEuroToCents(amountEuro);
		} catch {
			return fail(400, { actionError: 'Montant de remise en banque invalide.' });
		}

		if (!entityId || amountCents <= 0) {
			return fail(400, { actionError: 'Le montant déposé doit être supérieur à 0 €.' });
		}

		try {
			const recorded = await locals.api.payment.recordCashDeposit({
				legal_entity_id: entityId,
				amount_cents: amountCents,
				label,
				entry_date: entryDate ? new Date(entryDate).toISOString() : undefined
			});
			return {
				depositEntry: recorded,
				depositRecorded: true,
				successMessage: 'Remise d’espèces en banque enregistrée (écriture 530 → 512).'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError:
					err instanceof ApiError ? err.message : 'Impossible d’enregistrer la remise d’espèces.'
			});
		}
	}
};
