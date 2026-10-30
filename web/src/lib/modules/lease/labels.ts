import type { LeaseType } from './types';

export const LEASE_TYPE_OPTIONS: ReadonlyArray<{ value: LeaseType; label: string }> = [
	{ value: 'RESIDENTIAL_UNFURNISHED', label: 'Habitation vide (Loi 89 — 3 ans)' },
	{ value: 'RESIDENTIAL_FURNISHED', label: 'Habitation meublée (1 an / 9 mois)' },
	{ value: 'COMMERCIAL_3_6_9', label: 'Bail commercial 3 / 6 / 9' },
	{ value: 'PROFESSIONAL', label: 'Bail professionnel' },
	{ value: 'SHORT_TERM', label: 'Bail mobilité / courte durée' },
	{ value: 'PARKING', label: 'Bail parking / garage' }
];

const LEASE_TYPE_LABELS = new Map(LEASE_TYPE_OPTIONS.map((opt) => [opt.value, opt.label]));

export function formatLeaseTypeLabel(value: string): string {
	return LEASE_TYPE_LABELS.get(value) ?? value;
}

export const INDEXATION_TYPE_OPTIONS: ReadonlyArray<{ value: string; label: string }> = [
	{ value: 'IRL', label: 'IRL (Indice de Référence des Loyers)' },
	{ value: 'ILC', label: 'ILC (Indice des Loyers Commerciaux)' },
	{ value: 'ILAT', label: 'ILAT (Indice des Loyers des Activités Tertiaires)' },
	{ value: 'NONE', label: 'Aucune indexation automatique' }
];

const INDEXATION_TYPE_LABELS = new Map(
	INDEXATION_TYPE_OPTIONS.map((opt) => [opt.value, opt.label])
);

export function formatIndexationTypeLabel(value: string): string {
	return INDEXATION_TYPE_LABELS.get(value) ?? value;
}
