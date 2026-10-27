export type InspectionType = 'MOVE_IN' | 'MOVE_OUT' | string;
export type ConditionState = 'NEW' | 'GOOD' | 'FAIR_WEAR' | 'POOR' | 'OUT_OF_SERVICE' | string;
export type IncidentStatus = 'OPEN' | 'IN_PROGRESS' | 'RESOLVED' | 'CLOSED' | string;

export interface MeterReadings {
	water_m3?: number;
	electricity_pdl?: string;
	electricity_kwh?: number;
	gas_pce?: string;
	gas_m3?: number;
}

export interface CreateInspectionTemplateCmd {
	name: string;
	rooms: string[];
	default_elements?: string[];
}

export interface InspectionTemplateDTO {
	id: string;
	name: string;
	rooms: string[];
	default_elements: string[];
	created_at: string;
	updated_at: string;
}

export interface CreateInspectionCmd {
	lease_id: string;
	template_id: string;
	type: InspectionType;
	entry_inspection_id?: string;
	inspection_date?: string;
	meter_readings?: MeterReadings;
	keys_handed_over_count?: number;
	keys_description?: string;
	estimated_repair_cost_cents?: number;
}

export interface InspectionItemInput {
	room_name: string;
	element_name: string;
	condition: ConditionState;
	observation?: string;
	photo_r2_keys?: string[];
}

export interface InspectionItemDTO {
	id: string;
	inspection_id: string;
	room_name: string;
	element_name: string;
	condition: ConditionState;
	observation?: string;
	photo_r2_keys: string[];
}

export interface InspectionDTO {
	id: string;
	lease_id: string;
	template_id: string;
	type: InspectionType;
	entry_inspection_id?: string;
	items: InspectionItemDTO[];
	meter_readings?: MeterReadings;
	keys_handed_over_count?: number;
	keys_description?: string;
	estimated_repair_cost_cents?: number;
	report_pdf_r2_key?: string;
	report_download_url?: string;
	inspection_date: string;
	created_at: string;
	updated_at: string;
}

export interface ElementComparisonDTO {
	room_name: string;
	element_name: string;
	entry_condition: ConditionState;
	exit_condition: ConditionState;
	entry_observation?: string;
	exit_observation?: string;
	degraded: boolean;
}

export interface InspectionReportDTO {
	inspection_id: string;
	lease_id: string;
	type: InspectionType;
	entry_inspection_id?: string;
	has_entry_comparison: boolean;
	comparisons?: ElementComparisonDTO[];
	degradations?: ElementComparisonDTO[];
	items: InspectionItemDTO[];
	pdf_r2_key: string;
	download_url: string;
	generated_at: string;
}

export interface CreateIncidentCmd {
	property_id?: string;
	lot_id?: string;
	lease_id?: string;
	title: string;
	status?: IncidentStatus;
	occurred_at?: string;
	insurance_claim_number?: string;
	estimated_cost_cents?: number;
	observations?: string[];
	photo_r2_keys?: string[];
}

export interface UpdateIncidentCmd {
	status?: IncidentStatus;
	insurance_claim_number?: string;
	estimated_cost_cents?: number;
	add_observation?: string;
	observations?: string[];
	add_photo_r2_keys?: string[];
	photo_r2_keys?: string[];
}

export interface IncidentDTO {
	id: string;
	property_id?: string;
	lot_id?: string;
	lease_id?: string;
	title: string;
	status: IncidentStatus;
	occurred_at: string;
	insurance_claim_number?: string;
	estimated_cost_cents: number;
	observations: string[];
	photo_r2_keys: string[];
	created_at: string;
	updated_at: string;
}
