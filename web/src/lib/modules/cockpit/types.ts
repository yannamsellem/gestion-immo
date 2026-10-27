import type { IncidentDTO } from '#lib/modules/inspection/types';
import type { LeaseDTO, RentCallDTO, TenantDTO } from '#lib/modules/lease/types';
import type { NegativePositionDTO } from '#lib/modules/payment/types';
import type { LegalEntityDTO, PropertyDTO } from '#lib/modules/portfolio/types';

export type PendingTaskKind = 'remind' | 'pay' | 'generate-calls' | 'incident' | 'create-lease';

export interface PendingTaskAction {
	kind: PendingTaskKind;
	label: string;
	href: string;
	variant: 'destructive' | 'default' | 'secondary' | 'outline';
}

export interface PropertyCockpitRow {
	entityId: string;
	propertyId: string;
	propertyLabel: string;
	lotId: string;
	lotLabel: string;
	lotType: string;
	leaseId?: string;
	tenantId?: string;
	tenantName?: string;
	monthlyRentIncludingChargesCents: number;
	remainingDueCents: number;
	isVacant: boolean;
	isUpToDate: boolean;
	openIncidentsCount: number;
	actions: PendingTaskAction[];
}

export interface EntityCockpitGroup {
	entity: LegalEntityDTO;
	expectedRentCents: number;
	collectedRentCents: number;
	remainingDueCents: number;
	uncalledLeasesCount: number;
	generateCallsHref?: string;
	rows: PropertyCockpitRow[];
}

export interface CockpitAggregationInput {
	periodMonth: string;
	entities: LegalEntityDTO[];
	propertiesByEntity: Record<string, PropertyDTO[]>;
	leases: LeaseDTO[];
	tenants: TenantDTO[];
	rentCallsByEntity: Record<string, RentCallDTO[]>;
	negativePositionsByEntity: Record<string, NegativePositionDTO[]>;
	incidents: IncidentDTO[];
}
