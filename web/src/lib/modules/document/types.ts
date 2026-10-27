export type TemplateType =
	'LEASE_CONTRACT' | 'RENT_CALL' | 'RECEIPT' | 'REMINDER' | 'GLI_NOTICE' | string;
export type TemplateFormat = 'DOCX' | 'HTML' | string;

export interface RegisterTemplateCmd {
	name: string;
	template_type: TemplateType;
	format: TemplateFormat;
	docx_base64?: string;
	html_body?: string;
}

export interface DocumentTemplateDTO {
	id: string;
	name: string;
	template_type: TemplateType;
	format: TemplateFormat;
	r2_key?: string;
	html_body?: string;
	created_at: string;
	updated_at: string;
}

export interface GenerateDocCmd {
	template_id: string;
	legal_entity_id?: string;
	lease_id?: string;
	variables?: Record<string, string>;
	tables?: Record<string, Array<Record<string, string>>>;
	output_file_name?: string;
}

export interface StoredFileDTO {
	id: string;
	legal_entity_id?: string;
	lease_id?: string;
	category: string;
	file_name: string;
	content_type: string;
	r2_key: string;
	size_bytes: number;
	download_url: string;
	created_at: string;
}

export interface GLIDossierArchiveDTO {
	id: string;
	lease_id: string;
	legal_entity_id: string;
	tenant_name: string;
	template_id: string;
	gli_policy_number: string;
	total_unpaid_cents: number;
	unpaid_months: string[];
	archive_r2_key: string;
	download_url: string;
	included_files: string[];
	archive_size_bytes: number;
	created_at: string;
}
