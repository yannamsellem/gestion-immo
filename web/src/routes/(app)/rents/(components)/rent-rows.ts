import type { LeaseDTO, RentCallDTO, TenantDTO } from '#lib/modules/lease/types';
import type { NegativePositionDTO } from '#lib/modules/payment/types';
import type { LegalEntityDTO, PropertyDTO } from '#lib/modules/portfolio/types';

export type RentStatusFilter = 'ALL' | 'TO_COLLECT' | 'OVERDUE' | 'PAID';

export interface UnifiedRentRow {
	lease: LeaseDTO;
	entity: LegalEntityDTO;
	propertyAndLotLabel: string;
	tenantName: string;
	rentCall: RentCallDTO | null;
	negativePosition: NegativePositionDTO | null;
	expectedGrossCents: number;
	remainingDueCents: number;
	statusCategory: 'PAID' | 'OVERDUE' | 'TO_COLLECT';
	statusBadgeLabel: string;
	statusBadgeVariant: 'default' | 'secondary' | 'destructive' | 'outline';
}

export interface RentSummaryTotals {
	expectedTotalCents: number;
	collectedTotalCents: number;
	remainingTotalCents: number;
	overdueCount: number;
}

export function buildUnifiedRentRows(params: {
	entities: LegalEntityDTO[];
	leases: LeaseDTO[];
	tenants: TenantDTO[];
	propertiesByEntity: Record<string, PropertyDTO[]>;
	rentCallsByEntity: Record<string, RentCallDTO[]>;
	negativePositionsByEntity: Record<string, NegativePositionDTO[]>;
	selectedEntityId: 'ALL' | string;
	selectedPeriodMonth: string;
}): UnifiedRentRow[] {
	const entityMap = new Map(params.entities.map((entity) => [entity.id, entity]));
	const tenantMap = new Map(params.tenants.map((tenant) => [tenant.id, tenant]));
	const lotLabelMap = new Map<string, string>(
		Object.values(params.propertiesByEntity).flatMap((propList) =>
			propList.flatMap((prop) =>
				(prop.lots ?? []).map((lot) => [lot.id, `${prop.label} — ${lot.label}`] as const)
			)
		)
	);
	const rentCallByLeaseId = new Map<string, RentCallDTO>(
		Object.values(params.rentCallsByEntity).flatMap((callList) =>
			callList
				.filter((rentCall) => rentCall.period_month === params.selectedPeriodMonth)
				.map((rentCall) => [rentCall.lease_id, rentCall] as const)
		)
	);
	const negativeByLeaseId = new Map<string, NegativePositionDTO>(
		Object.values(params.negativePositionsByEntity).flatMap((negList) =>
			negList.map((neg) => [neg.lease_id, neg] as const)
		)
	);

	return params.leases
		.filter(
			(lease) =>
				params.selectedEntityId === 'ALL' || lease.legal_entity_id === params.selectedEntityId
		)
		.map((lease) => {
			const entity = entityMap.get(lease.legal_entity_id) ?? params.entities[0];
			const rentCall = rentCallByLeaseId.get(lease.id) ?? null;
			const negativePosition = negativeByLeaseId.get(lease.id) ?? null;
			const vatCents = lease.subject_to_vat
				? Math.trunc((lease.rent_net_cents * lease.vat_rate_basis_points) / 10000)
				: 0;
			const expectedGrossCents = rentCall
				? rentCall.total_due_cents
				: lease.rent_net_cents + vatCents + lease.charges_cents;
			const remainingDueCents = rentCall ? rentCall.remaining_due_cents : expectedGrossCents;

			let statusCategory: 'PAID' | 'OVERDUE' | 'TO_COLLECT' = 'TO_COLLECT';
			let statusBadgeLabel = 'Appel à générer';
			let statusBadgeVariant: 'default' | 'secondary' | 'destructive' | 'outline' = 'outline';

			if (negativePosition && negativePosition.total_unpaid_cents > 0) {
				statusCategory = 'OVERDUE';
				statusBadgeLabel = `Impayé (${negativePosition.unpaid_months.length} m.)`;
				statusBadgeVariant = 'destructive';
			} else if (rentCall?.status === 'PAID' || (rentCall && rentCall.remaining_due_cents === 0)) {
				statusCategory = 'PAID';
				statusBadgeLabel = 'Soldé';
				statusBadgeVariant = 'secondary';
			} else if (rentCall?.status === 'PARTIAL') {
				statusBadgeLabel = 'Partiel';
				statusBadgeVariant = 'default';
			} else if (rentCall) {
				statusBadgeLabel = 'En attente';
			}

			return {
				lease,
				entity,
				propertyAndLotLabel: lotLabelMap.get(lease.lot_id) ?? `Lot ${lease.lot_id.slice(0, 8)}`,
				tenantName: tenantMap.get(lease.tenant_id)?.full_name ?? 'Locataire',
				rentCall,
				negativePosition,
				expectedGrossCents,
				remainingDueCents,
				statusCategory,
				statusBadgeLabel,
				statusBadgeVariant
			};
		});
}

export function summarizeRentRows(rows: UnifiedRentRow[]): RentSummaryTotals {
	const expectedTotalCents = rows.reduce((sum, row) => sum + row.expectedGrossCents, 0);
	const remainingTotalCents = rows.reduce((sum, row) => sum + row.remainingDueCents, 0);
	return {
		expectedTotalCents,
		collectedTotalCents: Math.max(0, expectedTotalCents - remainingTotalCents),
		remainingTotalCents,
		overdueCount: rows.filter((row) => row.statusCategory === 'OVERDUE').length
	};
}
