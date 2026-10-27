import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import { parseEuroToCents } from '#lib/core/formatters/money';
import type { AccountBalanceLineDTO, StatementSection } from '#lib/modules/accounting/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, parent, url }) => {
	const parentData = await parent();
	const entities = parentData.entities;

	const activeEntityId =
		parentData.selectedEntityId !== 'ALL' ? parentData.selectedEntityId : (entities[0]?.id ?? '');

	const yearParam = Number.parseInt(url.searchParams.get('year') ?? '', 10);
	const fiscalYear =
		!Number.isNaN(yearParam) && yearParam >= 2000 && yearParam <= 2100
			? yearParam
			: new Date().getFullYear();

	const disabledParam = url.searchParams.get('disabled') ?? '';
	const disabledLineIds = disabledParam ? disabledParam.split(',').filter(Boolean) : [];

	if (!activeEntityId) {
		return {
			entities,
			activeEntityId: '',
			fiscalYear,
			disabledLineIds,
			statements: null,
			journalEntries: [],
			errorMessage: null as string | null
		};
	}

	try {
		const [statements, journalEntries] = await Promise.all([
			locals.api.accounting.previewStatements(activeEntityId, fiscalYear, disabledLineIds),
			locals.api.accounting.listJournalEntries(activeEntityId, fiscalYear)
		]);

		return {
			entities,
			activeEntityId,
			fiscalYear,
			disabledLineIds,
			statements,
			journalEntries,
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			entities,
			activeEntityId,
			fiscalYear,
			disabledLineIds,
			statements: null,
			journalEntries: [],
			errorMessage:
				err instanceof ApiError
					? err.message
					: 'Impossible de charger les états financiers et le journal comptable.'
		};
	}
};

export const actions: Actions = {
	recompute: async ({ request, locals }) => {
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
			const recomputedStatements = await locals.api.accounting.previewStatements(
				entityId,
				fiscalYear,
				disabledLineIds
			);
			return {
				recomputedStatements,
				disabledLineIds
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Erreur lors du recalcul interactif.'
			});
		}
	},

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
	},

	upsertBalanceLine: async ({ request, locals }) => {
		const formData = await request.formData();
		const entityId = String(formData.get('entity_id') ?? '').trim();
		const fiscalYear =
			Number.parseInt(String(formData.get('fiscal_year') ?? ''), 10) || new Date().getFullYear();
		const existingLinesJson = String(formData.get('existing_lines_json') ?? '[]');
		const accountCode = String(formData.get('account_code') ?? '').trim();
		const label = String(formData.get('label') ?? '').trim();
		const section = String(formData.get('section') ?? 'OPERATING_REVENUE') as StatementSection;
		const cerfaCode = String(formData.get('cerfa_code') ?? '').trim();
		const netNEuro = String(formData.get('net_n_euro') ?? '0');
		const netNMinus1Euro = String(formData.get('net_n_minus_1_euro') ?? '0');

		if (!entityId || !accountCode || !label) {
			return fail(400, {
				actionError: 'Veuillez renseigner le numéro et l’intitulé du compte PCG.'
			});
		}

		let netNCents: number;
		let netNMinus1Cents: number;
		let existingLines: AccountBalanceLineDTO[];
		try {
			netNCents = parseEuroToCents(netNEuro || '0');
			netNMinus1Cents = parseEuroToCents(netNMinus1Euro || '0');
			existingLines = JSON.parse(existingLinesJson) as AccountBalanceLineDTO[];
		} catch {
			return fail(400, { actionError: 'Montant N ou N-1 invalide.' });
		}

		const lineId = `line_${accountCode}`;
		const filtered = existingLines.filter((l) => l.line_id !== lineId);
		const nextLines: AccountBalanceLineDTO[] = [
			...filtered,
			{
				line_id: lineId,
				account_code: accountCode,
				label,
				section,
				cerfa_code: cerfaCode || undefined,
				gross_n_cents: netNCents,
				depreciation_n_cents: 0,
				net_n_cents: netNCents,
				net_n_minus_1_cents: netNMinus1Cents
			}
		];

		try {
			await locals.api.accounting.upsertBalanceLines(entityId, fiscalYear, nextLines);
			return {
				balanceLineSaved: true,
				successMessage: 'Ligne de balance enregistrée et états financiers recalculés.'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Erreur lors de la mise à jour.'
			});
		}
	},

	recordJournalEntry: async ({ request, locals }) => {
		const formData = await request.formData();
		const entityId = String(formData.get('entity_id') ?? '').trim();
		const fiscalYear =
			Number.parseInt(String(formData.get('fiscal_year') ?? ''), 10) || new Date().getFullYear();
		const journalCode = String(formData.get('journal_code') ?? 'OD').trim() || 'OD';
		const journalLabel =
			String(formData.get('journal_label') ?? 'Opérations diverses').trim() ||
			'Opérations diverses';
		const documentRef = String(formData.get('document_ref') ?? '').trim() || `PIECE-${Date.now()}`;
		const entryLabel = String(formData.get('entry_label') ?? '').trim();
		const debitAccount = String(formData.get('debit_account') ?? '').trim();
		const debitAccountLabel = String(formData.get('debit_account_label') ?? '').trim();
		const debitSection = String(
			formData.get('debit_section') ?? 'DEPRECIATION'
		) as StatementSection;
		const debitEuro = String(formData.get('debit_euro') ?? '0');
		const creditAccount = String(formData.get('credit_account') ?? '').trim();
		const creditAccountLabel = String(formData.get('credit_account_label') ?? '').trim();
		const creditSection = String(
			formData.get('credit_section') ?? 'FIXED_ASSETS'
		) as StatementSection;
		const creditEuro = String(formData.get('credit_euro') ?? '0');

		let debitCents: number;
		let creditCents: number;
		try {
			debitCents = parseEuroToCents(debitEuro || '0');
			creditCents = parseEuroToCents(creditEuro || '0');
		} catch {
			return fail(400, { actionError: 'Montant débit ou crédit invalide.' });
		}

		if (!entityId || debitCents <= 0 || debitCents !== creditCents) {
			return fail(400, {
				actionError: 'L’écriture doit être équilibrée (Total Débit = Total Crédit > 0).'
			});
		}

		try {
			await locals.api.accounting.recordJournalEntry(entityId, {
				fiscal_year: fiscalYear,
				journal_code: journalCode,
				journal_label: journalLabel,
				document_ref: documentRef,
				label: entryLabel,
				lines: [
					{
						line_id: `jrn_${debitAccount}`,
						account_code: debitAccount,
						account_label: debitAccountLabel,
						section: debitSection,
						label: entryLabel,
						debit_cents: debitCents,
						credit_cents: 0
					},
					{
						line_id: `jrn_${creditAccount}`,
						account_code: creditAccount,
						account_label: creditAccountLabel,
						section: creditSection,
						label: entryLabel,
						debit_cents: 0,
						credit_cents: creditCents
					}
				]
			});
			return {
				journalRecorded: true,
				successMessage: 'Écriture comptable en partie double validée.'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError:
					err instanceof ApiError ? err.message : 'Impossible d’enregistrer l’écriture comptable.'
			});
		}
	}
};
