import { ApiError } from '#lib/core/api/api-error';
import { buildCockpitEntityGroups } from '#lib/modules/cockpit/cockpit-builder';
import type { RentCallDTO } from '#lib/modules/lease/types';
import type { NegativePositionDTO } from '#lib/modules/payment/types';
import type { PropertyDTO } from '#lib/modules/portfolio/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, parent }) => {
	const parentData = await parent();
	const allEntities = parentData.entities;
	const selectedEntityId = parentData.selectedEntityId;
	const periodMonth = parentData.selectedPeriodMonth;

	const targetEntities =
		selectedEntityId === 'ALL'
			? allEntities
			: allEntities.filter((entity) => entity.id === selectedEntityId);

	if (targetEntities.length === 0) {
		return {
			groups: [],
			totalOverdueLeasesCount: 0,
			totalOpenIncidentsCount: 0,
			errorMessage: null as string | null
		};
	}

	try {
		const [leases, tenants, incidents] = await Promise.all([
			locals.api.lease.listLeases(selectedEntityId),
			locals.api.lease.listTenants(false),
			locals.api.inspection.listIncidents().catch(() => [])
		]);

		const propertiesByEntity: Record<string, PropertyDTO[]> = {};
		const rentCallsByEntity: Record<string, RentCallDTO[]> = {};
		const negativePositionsByEntity: Record<string, NegativePositionDTO[]> = {};

		await Promise.all(
			targetEntities.map(async (entity) => {
				const [properties, rentCalls, negativePositions] = await Promise.all([
					locals.api.portfolio.listProperties(entity.id),
					locals.api.lease.listEntityRentCalls(entity.id, periodMonth),
					locals.api.payment.listNegativePositions(entity.id).catch(() => [])
				]);
				propertiesByEntity[entity.id] = properties;
				rentCallsByEntity[entity.id] = rentCalls;
				negativePositionsByEntity[entity.id] = negativePositions;
			})
		);

		const groups = buildCockpitEntityGroups({
			periodMonth,
			entities: targetEntities,
			propertiesByEntity,
			leases,
			tenants,
			rentCallsByEntity,
			negativePositionsByEntity,
			incidents
		});

		const totalOverdueLeasesCount = Object.values(negativePositionsByEntity).reduce(
			(acc, list) => acc + list.filter((item) => item.total_unpaid_cents > 0).length,
			0
		);
		const totalOpenIncidentsCount = groups.reduce(
			(acc, group) => acc + group.rows.reduce((rowAcc, row) => rowAcc + row.openIncidentsCount, 0),
			0
		);

		return {
			groups,
			totalOverdueLeasesCount,
			totalOpenIncidentsCount,
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			groups: [],
			totalOverdueLeasesCount: 0,
			totalOpenIncidentsCount: 0,
			errorMessage:
				err instanceof ApiError
					? err.message
					: 'Impossible de charger les données du cockpit opérationnel.'
		};
	}
};
