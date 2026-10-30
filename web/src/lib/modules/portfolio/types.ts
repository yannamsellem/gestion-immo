import type { components } from '#lib/core/api/openapi';

export type EntityType =
	| 'SCI_IS'
	| 'SCI_IR'
	| 'SARL_IS'
	| 'FAMILY_SARL'
	| 'SAS'
	| 'SOLE_PROPRIETORSHIP'
	| 'JOINT_OWNERSHIP'
	| string;
export type AccountingCategory = 'FIXED_ASSET' | 'MERCHANDISE' | 'FINISHED_GOOD' | string;

export type CreateLegalEntityCmd = components['schemas']['CreateLegalEntityCmd'];
export type LegalEntityDTO = components['schemas']['LegalEntityDTO'];
export type UpsertShareholderCmd = components['schemas']['UpsertShareholderCmd'];
export type ShareholderDTO = components['schemas']['ShareholderDTO'];
export type CreatePropertyCmd = components['schemas']['CreatePropertyCmd'];
export type CreateLotCmd = components['schemas']['CreateLotCmd'];
export type LotDTO = components['schemas']['LotDTO'];
export type PropertyDTO = components['schemas']['PropertyDTO'];
export type EntityPortfolioSummaryDTO = components['schemas']['EntityPortfolioSummaryDTO'];
