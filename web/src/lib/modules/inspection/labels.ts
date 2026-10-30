import type { ConditionState, IncidentStatus, InspectionType } from './types';

export const INCIDENT_STATUS_OPTIONS: ReadonlyArray<{ value: IncidentStatus; label: string }> = [
	{ value: 'OPEN', label: 'Ouvert' },
	{ value: 'INSURANCE_CLAIM', label: 'Déclaration assurance' },
	{ value: 'WORK_IN_PROGRESS', label: 'Travaux en cours' },
	{ value: 'CLOSED', label: 'Clôturé' }
];

const INCIDENT_STATUS_LABELS = new Map(
	INCIDENT_STATUS_OPTIONS.map((opt) => [opt.value, opt.label])
);

export function formatIncidentStatusLabel(status: IncidentStatus): string {
	return INCIDENT_STATUS_LABELS.get(status) ?? status;
}

export function formatIncidentStatusVariant(
	status: IncidentStatus
): 'destructive' | 'default' | 'secondary' | 'outline' {
	if (status === 'OPEN') return 'destructive';
	if (status === 'INSURANCE_CLAIM') return 'default';
	if (status === 'WORK_IN_PROGRESS') return 'secondary';
	return 'outline';
}

export const INSPECTION_TYPE_OPTIONS: ReadonlyArray<{ value: InspectionType; label: string }> = [
	{ value: 'MOVE_IN', label: 'Entrant' },
	{ value: 'MOVE_OUT', label: 'Sortant' }
];

const INSPECTION_TYPE_LABELS = new Map(
	INSPECTION_TYPE_OPTIONS.map((opt) => [opt.value, opt.label])
);

export function formatInspectionTypeLabel(type: InspectionType): string {
	return INSPECTION_TYPE_LABELS.get(type) ?? type;
}

export const CONDITION_STATE_OPTIONS: ReadonlyArray<{ value: ConditionState; label: string }> = [
	{ value: 'NEW', label: 'Neuf' },
	{ value: 'GOOD', label: 'Bon état' },
	{ value: 'FAIR_WEAR', label: 'État d’usage' },
	{ value: 'POOR', label: 'Mauvais état' },
	{ value: 'OUT_OF_SERVICE', label: 'Hors service' }
];

const CONDITION_STATE_LABELS = new Map(
	CONDITION_STATE_OPTIONS.map((opt) => [opt.value, opt.label])
);

export function formatConditionStateLabel(state: ConditionState): string {
	return CONDITION_STATE_LABELS.get(state) ?? state;
}
