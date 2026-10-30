import type { components } from '#lib/core/api/openapi';

export type LeaseType =
	| 'RESIDENTIAL_UNFURNISHED'
	| 'RESIDENTIAL_FURNISHED'
	| 'COMMERCIAL_3_6_9'
	| 'PROFESSIONAL'
	| 'PARKING'
	| 'SHORT_TERM'
	| string;

export type RentCallStatus = 'PENDING' | 'PARTIAL' | 'PAID' | 'UNPAID' | string;

export type CreateTenantCmd = components['schemas']['CreateTenantCmd'];
export type TenantDTO = components['schemas']['TenantDTO'];
export type CreateLeaseCmd = components['schemas']['CreateLeaseCmd'];
export type LeaseDTO = components['schemas']['LeaseDTO'];
export type RentCallDTO = components['schemas']['RentCallDTO'];
