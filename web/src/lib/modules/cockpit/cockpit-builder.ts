import type {
	CockpitAggregationInput,
	EntityCockpitGroup,
	PendingTaskAction,
	PropertyCockpitRow
} from './types';

function computeLeaseGrossMonthlyCents(params: {
	rentNetCents: number;
	chargesCents: number;
	subjectToVat: boolean;
	vatRateBasisPoints: number;
}): number {
	const vatCents = params.subjectToVat
		? Math.trunc((params.rentNetCents * params.vatRateBasisPoints) / 10000)
		: 0;
	return params.rentNetCents + vatCents + params.chargesCents;
}

export function buildCockpitEntityGroups(input: CockpitAggregationInput): EntityCockpitGroup[] {
	const tenantById = new Map(input.tenants.map((tenant) => [tenant.id, tenant]));
	const activeLeaseByLotId = new Map(
		input.leases.filter((lease) => lease.active).map((lease) => [lease.lot_id, lease])
	);
	const openIncidents = input.incidents.filter(
		(incident) => incident.status === 'OPEN' || incident.status === 'IN_PROGRESS'
	);

	return input.entities.map((entity) => {
		const properties = input.propertiesByEntity[entity.id] ?? [];
		const entityRentCalls = (input.rentCallsByEntity[entity.id] ?? []).filter(
			(rc) => rc.period_month === input.periodMonth
		);
		const rentCallByLeaseId = new Map(entityRentCalls.map((rc) => [rc.lease_id, rc]));

		const negativePositions = input.negativePositionsByEntity[entity.id] ?? [];
		const negativeByLeaseId = new Map(negativePositions.map((pos) => [pos.lease_id, pos]));

		let expectedRentCents = 0;
		let collectedRentCents = 0;
		let remainingDueCents = 0;
		let uncalledLeasesCount = 0;

		const rows: PropertyCockpitRow[] = [];

		for (const property of properties) {
			const lots = property.lots ?? [];
			if (lots.length === 0) {
				const propIncidents = openIncidents.filter((inc) => inc.property_id === property.id);
				const actions: PendingTaskAction[] = [];
				if (propIncidents.length > 0) {
					actions.push({
						kind: 'incident',
						label: `${propIncidents.length} Incident${propIncidents.length > 1 ? 's' : ''} →`,
						href: `/inspections?entityId=${encodeURIComponent(entity.id)}&propertyId=${encodeURIComponent(property.id)}&incidentId=${encodeURIComponent(propIncidents[0].id)}&tab=incidents`,
						variant: 'secondary'
					});
				}
				rows.push({
					entityId: entity.id,
					propertyId: property.id,
					propertyLabel: property.label,
					lotId: property.id,
					lotLabel: property.label,
					lotType: property.property_type,
					monthlyRentIncludingChargesCents: 0,
					remainingDueCents: 0,
					isVacant: true,
					isUpToDate: false,
					openIncidentsCount: propIncidents.length,
					actions: [
						...actions,
						{
							kind: 'create-lease',
							label: 'Ajouter un lot / bail →',
							href: `/properties?entityId=${encodeURIComponent(entity.id)}&propertyId=${encodeURIComponent(property.id)}`,
							variant: 'outline'
						}
					]
				});
				continue;
			}

			for (const lot of lots) {
				const lease = activeLeaseByLotId.get(lot.id);
				const lotIncidents = openIncidents.filter(
					(inc) =>
						inc.lot_id === lot.id ||
						(inc.property_id === property.id && !inc.lot_id) ||
						(lease && inc.lease_id === lease.id)
				);

				if (!lease) {
					const vacantActions: PendingTaskAction[] = [
						{
							kind: 'create-lease',
							label: 'Créer un bail →',
							href: `/leases?entityId=${encodeURIComponent(entity.id)}&lotId=${encodeURIComponent(lot.id)}&action=create`,
							variant: 'outline'
						}
					];
					if (lotIncidents.length > 0) {
						vacantActions.push({
							kind: 'incident',
							label: `${lotIncidents.length} Incident${lotIncidents.length > 1 ? 's' : ''} →`,
							href: `/inspections?entityId=${encodeURIComponent(entity.id)}&propertyId=${encodeURIComponent(property.id)}&incidentId=${encodeURIComponent(lotIncidents[0].id)}&tab=incidents`,
							variant: 'secondary'
						});
					}

					rows.push({
						entityId: entity.id,
						propertyId: property.id,
						propertyLabel: property.label,
						lotId: lot.id,
						lotLabel: lot.label,
						lotType: lot.lot_type,
						monthlyRentIncludingChargesCents: 0,
						remainingDueCents: 0,
						isVacant: true,
						isUpToDate: false,
						openIncidentsCount: lotIncidents.length,
						actions: vacantActions
					});
					continue;
				}

				const tenant = tenantById.get(lease.tenant_id);
				const rentCall = rentCallByLeaseId.get(lease.id);
				const negativePosition = negativeByLeaseId.get(lease.id);
				const leaseMonthlyCents = rentCall
					? rentCall.total_due_cents
					: computeLeaseGrossMonthlyCents({
							rentNetCents: lease.rent_net_cents,
							chargesCents: lease.charges_cents,
							subjectToVat: lease.subject_to_vat,
							vatRateBasisPoints: lease.vat_rate_basis_points
						});

				const rowRemainingDue = rentCall ? rentCall.remaining_due_cents : leaseMonthlyCents;
				const rowCollected = rentCall
					? Math.max(0, rentCall.total_due_cents - rentCall.remaining_due_cents)
					: 0;

				expectedRentCents += leaseMonthlyCents;
				collectedRentCents += rowCollected;
				remainingDueCents += rowRemainingDue;

				const actions: PendingTaskAction[] = [];

				if (negativePosition && negativePosition.total_unpaid_cents > 0) {
					actions.push({
						kind: 'remind',
						label: 'Relancer impayé →',
						href: `/rents?entityId=${encodeURIComponent(entity.id)}&leaseId=${encodeURIComponent(lease.id)}${rentCall ? `&rentCallId=${encodeURIComponent(rentCall.id)}` : ''}&action=remind`,
						variant: 'destructive'
					});
				}

				if (!rentCall) {
					uncalledLeasesCount += 1;
					actions.push({
						kind: 'generate-calls',
						label: 'Générer appel →',
						href: `/rents?entityId=${encodeURIComponent(entity.id)}&leaseId=${encodeURIComponent(lease.id)}&action=generate-calls`,
						variant: 'secondary'
					});
				} else if (rentCall.status !== 'PAID' && rentCall.remaining_due_cents > 0) {
					actions.push({
						kind: 'pay',
						label: 'Saisir loyer →',
						href: `/rents?entityId=${encodeURIComponent(entity.id)}&leaseId=${encodeURIComponent(lease.id)}&rentCallId=${encodeURIComponent(rentCall.id)}&action=pay`,
						variant: 'default'
					});
				}

				if (lotIncidents.length > 0) {
					actions.push({
						kind: 'incident',
						label: `${lotIncidents.length} Incident${lotIncidents.length > 1 ? 's' : ''} →`,
						href: `/inspections?entityId=${encodeURIComponent(entity.id)}&propertyId=${encodeURIComponent(property.id)}&incidentId=${encodeURIComponent(lotIncidents[0].id)}&tab=incidents`,
						variant: 'secondary'
					});
				}

				const isUpToDate =
					Boolean(rentCall && rentCall.status === 'PAID') &&
					(!negativePosition || negativePosition.total_unpaid_cents <= 0) &&
					lotIncidents.length === 0;

				rows.push({
					entityId: entity.id,
					propertyId: property.id,
					propertyLabel: property.label,
					lotId: lot.id,
					lotLabel: lot.label,
					lotType: lot.lot_type,
					leaseId: lease.id,
					tenantId: lease.tenant_id,
					tenantName: tenant?.full_name ?? 'Locataire',
					monthlyRentIncludingChargesCents: leaseMonthlyCents,
					remainingDueCents: rowRemainingDue,
					isVacant: false,
					isUpToDate,
					openIncidentsCount: lotIncidents.length,
					actions
				});
			}
		}

		return {
			entity,
			expectedRentCents,
			collectedRentCents,
			remainingDueCents,
			uncalledLeasesCount,
			generateCallsHref:
				uncalledLeasesCount > 0
					? `/rents?entityId=${encodeURIComponent(entity.id)}&action=generate-calls`
					: undefined,
			rows
		};
	});
}
