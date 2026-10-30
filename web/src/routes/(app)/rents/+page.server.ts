import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import { formatPeriodMonthFr } from '#lib/core/formatters/date';
import { formatCentsToEuro, parseEuroToCents } from '#lib/core/formatters/money';
import type { StoredFileDTO } from '#lib/modules/document/types';
import type { RentCallDTO } from '#lib/modules/lease/types';
import type { NegativePositionDTO, PaymentMethod } from '#lib/modules/payment/types';
import type { PropertyDTO } from '#lib/modules/portfolio/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, parent }) => {
	const parentData = await parent();
	const entities = parentData.entities;
	const selectedEntityId = parentData.selectedEntityId;
	const periodMonth = parentData.selectedPeriodMonth;

	const targetEntities =
		selectedEntityId === 'ALL' ? entities : entities.filter((item) => item.id === selectedEntityId);

	try {
		const [loadedLeases, tenants, templates] = await Promise.all([
			locals.api.lease.listLeases(selectedEntityId),
			locals.api.lease.listTenants(true),
			locals.api.document.listTemplates().catch(() => [])
		]);

		const activeLeases = loadedLeases.filter((lease) => lease.active);

		const propertiesByEntity: Record<string, PropertyDTO[]> = {};
		const rentCallsByEntity: Record<string, RentCallDTO[]> = {};
		const negativePositionsByEntity: Record<string, NegativePositionDTO[]> = {};
		const documentsByLeaseId: Record<string, StoredFileDTO[]> = {};

		await Promise.all([
			...targetEntities.map(async (entity) => {
				const [props, calls, negatives] = await Promise.all([
					locals.api.portfolio.listProperties(entity.id),
					locals.api.lease.listEntityRentCalls(entity.id, periodMonth),
					locals.api.payment.listNegativePositions(entity.id).catch(() => [])
				]);
				propertiesByEntity[entity.id] = props;
				rentCallsByEntity[entity.id] = calls;
				negativePositionsByEntity[entity.id] = negatives;
			}),
			...activeLeases.map(async (lease) => {
				const docs = await locals.api.document.listLeaseDocuments(lease.id).catch(() => []);
				documentsByLeaseId[lease.id] = docs;
			})
		]);

		return {
			entities,
			leases: activeLeases,
			tenants,
			templates,
			propertiesByEntity,
			rentCallsByEntity,
			negativePositionsByEntity,
			documentsByLeaseId,
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			entities,
			leases: [],
			tenants: [],
			templates: [],
			propertiesByEntity: {} as Record<string, PropertyDTO[]>,
			rentCallsByEntity: {} as Record<string, RentCallDTO[]>,
			negativePositionsByEntity: {} as Record<string, NegativePositionDTO[]>,
			documentsByLeaseId: {} as Record<string, StoredFileDTO[]>,
			errorMessage:
				err instanceof ApiError
					? err.message
					: 'Impossible de charger les appels de loyers et encaissements.'
		};
	}
};

export const actions: Actions = {
	generateCalls: async ({ request, locals }) => {
		const formData = await request.formData();
		const periodMonth = String(formData.get('period_month') ?? locals.selectedPeriodMonth);
		const entityId = String(formData.get('entity_id') ?? locals.selectedEntityId);

		try {
			const allEntities = await locals.api.portfolio.listEntities();
			const targetEntities =
				entityId === 'ALL' ? allEntities : allEntities.filter((entity) => entity.id === entityId);

			if (targetEntities.length === 0) {
				return fail(400, { actionError: 'Aucune entité sélectionnée.' });
			}

			let generatedCount = 0;
			for (const entity of targetEntities) {
				const calls = await locals.api.lease.generateRentCalls(entity.id, periodMonth);
				generatedCount += calls.length;
			}

			return {
				successMessage: `${generatedCount} appel(s) de loyer généré(s) pour ${formatPeriodMonthFr(periodMonth)}.`
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError:
					err instanceof ApiError
						? err.message
						: 'Erreur lors de la génération des appels de loyers.'
			});
		}
	},

	importCsv: async ({ request, locals }) => {
		const formData = await request.formData();
		const entityId = String(formData.get('entity_id') ?? '').trim();
		const csvContent = String(formData.get('csv_content') ?? '').trim();

		if (!entityId || !csvContent) {
			return fail(400, {
				actionError: 'Veuillez choisir une entité et fournir un contenu CSV.'
			});
		}

		try {
			const result = await locals.api.payment.importPaymentsCsv(entityId, csvContent);
			return {
				csvImported: true,
				successMessage: `${result.imported_count} virement(s) importé(s) (${formatCentsToEuro(result.total_imported_cents)}).`
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Échec de l’import bancaire CSV.'
			});
		}
	},

	recordPayment: async ({ request, locals }) => {
		const formData = await request.formData();
		const leaseId = String(formData.get('lease_id') ?? '').trim();
		const amountEuro = String(formData.get('amount_euro') ?? '');
		const method = String(formData.get('method') ?? 'BANK_TRANSFER') as PaymentMethod;
		const label = String(formData.get('label') ?? '').trim();
		const reference = String(formData.get('reference') ?? '').trim();
		const paymentDate = String(formData.get('payment_date') ?? '').trim();
		const periodMonth = String(formData.get('period_month') ?? locals.selectedPeriodMonth);

		let amountCents: number;
		try {
			amountCents = parseEuroToCents(amountEuro);
		} catch {
			return fail(400, {
				actionError: 'Veuillez saisir un montant valide en euros (ex: 850,00).'
			});
		}

		if (!leaseId || amountCents <= 0) {
			return fail(400, {
				actionError: 'Le montant du règlement doit être strictement positif.'
			});
		}

		try {
			if (method === 'CASH') {
				await locals.api.payment.recordCashPayment({
					lease_id: leaseId,
					amount_cents: amountCents,
					label: label || `Loyer espèces ${periodMonth}`,
					receipt_number: reference || undefined,
					payment_date: paymentDate ? new Date(paymentDate).toISOString() : undefined
				});
				return {
					paymentRecorded: true,
					successMessage: 'Paiement en espèces imputé et inscrit au livre de caisse (PCG 530).'
				};
			}

			await locals.api.payment.recordBankPayment({
				lease_id: leaseId,
				amount_cents: amountCents,
				method,
				label: label || `Règlement loyer ${periodMonth}`,
				external_ref: reference || undefined,
				payment_date: paymentDate ? new Date(paymentDate).toISOString() : undefined
			});
			return {
				paymentRecorded: true,
				successMessage: 'Règlement enregistré et imputé (FIFO).'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError:
					err instanceof ApiError
						? err.message
						: 'Impossible d’enregistrer le règlement pour ce bail.'
			});
		}
	},

	generateDoc: async ({ request, locals }) => {
		const formData = await request.formData();
		const leaseId = String(formData.get('lease_id') ?? '').trim();
		const legalEntityId = String(formData.get('legal_entity_id') ?? '').trim();
		const templateId = String(formData.get('template_id') ?? '').trim();
		const tenantName = String(formData.get('tenant_name') ?? '');
		const entityName = String(formData.get('entity_name') ?? '');
		const propertyLabel = String(formData.get('property_label') ?? '');
		const periodMonth = String(formData.get('period_month') ?? '');
		const amountDue = String(formData.get('amount_due') ?? '');

		if (!leaseId || !templateId) {
			return fail(400, { actionError: 'Veuillez sélectionner un modèle de document.' });
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
					period_month: formatPeriodMonthFr(periodMonth),
					amount_due: amountDue
				}
			});
			return {
				generatedDoc: generated,
				successMessage: `Document généré : ${generated.file_name}`
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Échec de la génération du document.'
			});
		}
	},

	assembleGli: async ({ request, locals }) => {
		const formData = await request.formData();
		const leaseId = String(formData.get('lease_id') ?? '').trim();
		const templateId = String(formData.get('template_id') ?? '').trim();

		if (!leaseId) {
			return fail(400, { actionError: 'Bail introuvable.' });
		}

		try {
			const archive = await locals.api.document.assembleGliDossier(
				leaseId,
				templateId || undefined
			);
			return {
				gliArchive: archive,
				successMessage: 'Dossier sinistre GLI (.zip) assemblé avec succès.'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError:
					err instanceof ApiError
						? err.message
						: 'Impossible d’assembler le dossier de sinistre GLI.'
			});
		}
	}
};
