import type { AccountingCategory, EntityType } from './types';

export const ENTITY_TYPE_OPTIONS: ReadonlyArray<{ value: EntityType; label: string }> = [
	{ value: 'SCI_IS', label: 'SCI à l’IS' },
	{ value: 'SCI_IR', label: 'SCI à l’IR' },
	{ value: 'SARL_IS', label: 'SARL à l’IS / Marchand de biens' },
	{ value: 'FAMILY_SARL', label: 'SARL de famille' },
	{ value: 'SAS', label: 'SAS / SASU' },
	{ value: 'SOLE_PROPRIETORSHIP', label: 'Détention en nom propre' },
	{ value: 'JOINT_OWNERSHIP', label: 'Indivision' }
];

const ENTITY_TYPE_LABELS = new Map(ENTITY_TYPE_OPTIONS.map((opt) => [opt.value, opt.label]));

export function formatEntityTypeLabel(value: string): string {
	return ENTITY_TYPE_LABELS.get(value) ?? value;
}

export const PROPERTY_TYPE_OPTIONS: ReadonlyArray<{ value: string; label: string }> = [
	{ value: 'BUILDING', label: 'Immeuble de rapport' },
	{ value: 'APARTMENT', label: 'Appartement' },
	{ value: 'HOUSE', label: 'Maison individuelle' },
	{ value: 'COMMERCIAL', label: 'Local commercial / Bureaux' }
];

const EXTRA_PROPERTY_TYPE_LABELS: Readonly<Record<string, string>> = {
	APARTMENT_BUILDING: 'Immeuble de rapport'
};

const PROPERTY_TYPE_LABELS = new Map(PROPERTY_TYPE_OPTIONS.map((opt) => [opt.value, opt.label]));

export function formatPropertyTypeLabel(value: string): string {
	return PROPERTY_TYPE_LABELS.get(value) ?? EXTRA_PROPERTY_TYPE_LABELS[value] ?? value;
}

export const ACCOUNTING_CATEGORY_OPTIONS: ReadonlyArray<{
	value: AccountingCategory;
	label: string;
}> = [
	{ value: 'FIXED_ASSET', label: 'Immobilisation corporelle (PCG 213)' },
	{ value: 'MERCHANDISE', label: 'Stock marchand de biens (PCG 355/370)' },
	{ value: 'FINISHED_GOOD', label: 'Produit fini / Promotion (PCG 355)' }
];

const ACCOUNTING_CATEGORY_LABELS = new Map(
	ACCOUNTING_CATEGORY_OPTIONS.map((opt) => [opt.value, opt.label])
);

export function formatAccountingCategoryLabel(value: string): string {
	return ACCOUNTING_CATEGORY_LABELS.get(value) ?? value;
}

export const LOT_TYPE_OPTIONS: ReadonlyArray<{ value: string; label: string }> = [
	{ value: 'APARTMENT', label: 'Appartement' },
	{ value: 'COMMERCIAL', label: 'Local commercial' },
	{ value: 'PARKING', label: 'Parking / Box' },
	{ value: 'CELLAR', label: 'Cave' },
	{ value: 'HOUSE', label: 'Maison' }
];

const EXTRA_LOT_TYPE_LABELS: Readonly<Record<string, string>> = {
	BUILDING: 'Immeuble',
	APARTMENT_BUILDING: 'Immeuble',
	STUDIO: 'Studio',
	OFFICE: 'Bureaux'
};

const LOT_TYPE_LABELS = new Map(LOT_TYPE_OPTIONS.map((opt) => [opt.value, opt.label]));

export function formatLotTypeLabel(value: string): string {
	return LOT_TYPE_LABELS.get(value) ?? EXTRA_LOT_TYPE_LABELS[value] ?? value;
}
