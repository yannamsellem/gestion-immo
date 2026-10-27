export type EntityType = 'SCI_IS' | 'SCI_IR' | 'LMNP' | 'NOM_PROPRE' | string;
export type AccountingCategory = 'FIXED_ASSET' | 'MERCHANDISE' | string;

export interface CreateLegalEntityCmd {
	name: string;
	entity_type: string;
	siret: string;
	naf_code: string;
	address: string;
	share_capital_cents: number;
	total_shares: number;
}

export interface LegalEntityDTO {
	id: string;
	name: string;
	entity_type: EntityType;
	siret: string;
	naf_code: string;
	address: string;
	share_capital_cents: number;
	total_shares: number;
	created_at: string;
	updated_at: string;
}

export interface UpsertShareholderCmd {
	legal_entity_id?: string;
	full_name: string;
	shares_count: number;
	current_account_cents: number;
	remuneration_paid_cents: number;
}

export interface ShareholderDTO {
	id: string;
	legal_entity_id: string;
	full_name: string;
	shares_count: number;
	detention_basis_points: number;
	current_account_cents: number;
	remuneration_paid_cents: number;
	created_at: string;
	updated_at: string;
}

export interface CreatePropertyCmd {
	legal_entity_id?: string;
	label: string;
	property_type: string;
	accounting_category: string;
	address: string;
	acquisition_cents: number;
	work_stock_cents: number;
}

export interface CreateLotCmd {
	property_id?: string;
	label: string;
	lot_type: string;
	surface_m2: number;
	ownership_share: number;
	floor: string;
}

export interface LotDTO {
	id: string;
	property_id: string;
	label: string;
	lot_type: string;
	surface_m2: number;
	ownership_share: number;
	floor: string;
	created_at: string;
	updated_at: string;
}

export interface PropertyDTO {
	id: string;
	legal_entity_id: string;
	label: string;
	property_type: string;
	accounting_category: AccountingCategory;
	address: string;
	acquisition_cents: number;
	work_stock_cents: number;
	total_book_value_cents: number;
	lots?: LotDTO[];
	created_at: string;
	updated_at: string;
}

export interface EntityPortfolioSummaryDTO {
	entity: LegalEntityDTO;
	shareholders: ShareholderDTO[];
	properties: PropertyDTO[];
	total_acquisition_cents: number;
	total_work_stock_cents: number;
	total_book_value_cents: number;
	total_lots_count: number;
}
