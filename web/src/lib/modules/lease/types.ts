export type LeaseType =
	| 'RESIDENTIAL_EMPTY'
	| 'RESIDENTIAL_FURNISHED'
	| 'MOBILITY'
	| 'COMMERCIAL_3_6_9'
	| string;

export type RentCallStatus = 'PENDING' | 'PARTIAL' | 'PAID' | string;

export interface CreateTenantCmd {
	full_name: string;
	email: string;
	phone: string;
	address: string;
}

export interface TenantDTO {
	id: string;
	full_name: string;
	email: string;
	phone: string;
	address: string;
	archived: boolean;
	archived_at?: string;
	created_at: string;
	updated_at: string;
}

export interface CreateLeaseCmd {
	legal_entity_id: string;
	lot_id: string;
	tenant_id: string;
	lease_type: string;
	rent_net_cents: number;
	charges_cents: number;
	subject_to_vat: boolean;
	vat_rate_basis_points: number;
	security_deposit_cents: number;
	due_day_of_month: number;
	gli_policy_number: string;
	guarantor_name: string;
	start_date?: string;
	end_date?: string;
}

export interface LeaseDTO {
	id: string;
	legal_entity_id: string;
	lot_id: string;
	tenant_id: string;
	lease_type: LeaseType;
	rent_net_cents: number;
	charges_cents: number;
	subject_to_vat: boolean;
	vat_rate_basis_points: number;
	security_deposit_cents: number;
	due_day_of_month: number;
	gli_policy_number: string;
	guarantor_name: string;
	start_date: string;
	end_date?: string;
	active: boolean;
	created_at: string;
	updated_at: string;
}

export interface RentCallDTO {
	id: string;
	lease_id: string;
	legal_entity_id: string;
	tenant_id: string;
	period_month: string;
	rent_net_cents: number;
	vat_cents: number;
	charges_cents: number;
	total_due_cents: number;
	remaining_due_cents: number;
	status: RentCallStatus;
	due_date: string;
	created_at: string;
	updated_at: string;
}
