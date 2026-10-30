import type { components } from '#lib/core/api/openapi';

export type InspectionType = 'MOVE_IN' | 'MOVE_OUT' | string;
export type ConditionState = 'NEW' | 'GOOD' | 'FAIR_WEAR' | 'POOR' | 'OUT_OF_SERVICE' | string;
export type IncidentStatus = 'OPEN' | 'INSURANCE_CLAIM' | 'WORK_IN_PROGRESS' | 'CLOSED' | string;

export type MeterReadings = components['schemas']['MeterReadings'];
export type CreateInspectionTemplateCmd = components['schemas']['CreateInspectionTemplateCmd'];
export type InspectionTemplateDTO = components['schemas']['InspectionTemplateDTO'];
export type CreateInspectionCmd = components['schemas']['CreateInspectionCmd'];
export type InspectionItemInput = components['schemas']['InspectionItemInput'];
export type InspectionItemDTO = components['schemas']['InspectionItemDTO'];
export type InspectionDTO = components['schemas']['InspectionDTO'];
export type ElementComparisonDTO = components['schemas']['ElementComparisonDTO'];
export type InspectionReportDTO = components['schemas']['InspectionReportDTO'];
export type CreateIncidentCmd = components['schemas']['CreateIncidentCmd'];
export type UpdateIncidentCmd = components['schemas']['UpdateIncidentCmd'];
export type IncidentDTO = components['schemas']['IncidentDTO'];
