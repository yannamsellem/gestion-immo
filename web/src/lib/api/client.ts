// web/src/lib/api/client.ts

export type UserRole = 'SUPER_ADMIN' | 'MANAGER';
export type UserStatus = 'PENDING_APPROVAL' | 'APPROVED' | 'REJECTED';

export interface UserDTO {
	readonly id: string;
	readonly full_name: string;
	readonly email: string;
	readonly role: UserRole;
	readonly status: UserStatus;
	readonly approved_by?: string;
	readonly approved_at?: string;
	readonly created_at: string;
}

export interface SignupRequest {
	readonly full_name: string;
	readonly email: string;
	readonly password: string;
}

export interface LoginRequest {
	readonly email: string;
	readonly password: string;
}

export interface AuthTokensResponse {
	readonly access_token: string;
	readonly refresh_token: string;
	readonly token_type: string;
	readonly user: UserDTO;
}

export type EntityType =
	| 'SCI_IR'
	| 'SCI_IS'
	| 'SARL_IS'
	| 'FAMILY_SARL'
	| 'SAS'
	| 'SOLE_PROPRIETORSHIP'
	| 'JOINT_OWNERSHIP';
export type AccountingCategory = 'FIXED_ASSET' | 'MERCHANDISE' | 'FINISHED_GOOD';

export interface LegalEntityDTO {
	readonly id: string;
	readonly name: string;
	readonly entity_type: EntityType;
	readonly siret: string;
	readonly naf_code: string;
	readonly address: string;
	readonly share_capital_cents: number;
	readonly total_shares: number;
	readonly created_at: string;
	readonly updated_at: string;
}

export interface CreateLegalEntityRequest {
	readonly name: string;
	readonly entity_type: EntityType;
	readonly siret: string;
	readonly naf_code: string;
	readonly address: string;
	readonly share_capital_cents: number;
	readonly total_shares: number;
}

export interface ShareholderDTO {
	readonly id: string;
	readonly legal_entity_id: string;
	readonly full_name: string;
	readonly shares_count: number;
	readonly detention_basis_points: number;
	readonly current_account_cents: number;
	readonly remuneration_paid_cents: number;
}

export interface LotDTO {
	readonly id: string;
	readonly property_id: string;
	readonly label: string;
	readonly lot_type: string;
	readonly surface_m2: number;
	readonly ownership_share: number;
	readonly floor: string;
}

export interface PropertyDTO {
	readonly id: string;
	readonly legal_entity_id: string;
	readonly label: string;
	readonly property_type: string;
	readonly accounting_category: AccountingCategory;
	readonly address: string;
	readonly acquisition_cents: number;
	readonly work_stock_cents: number;
	readonly total_book_value_cents: number;
	readonly lots?: readonly LotDTO[];
}

export interface CreatePropertyRequest {
	readonly label: string;
	readonly property_type: string;
	readonly accounting_category: AccountingCategory;
	readonly address: string;
	readonly acquisition_cents: number;
	readonly work_stock_cents: number;
}

export interface EntityPortfolioSummaryDTO {
	readonly entity: LegalEntityDTO;
	readonly shareholders: readonly ShareholderDTO[];
	readonly properties: readonly PropertyDTO[];
	readonly total_acquisition_cents: number;
	readonly total_work_stock_cents: number;
	readonly total_book_value_cents: number;
	readonly total_lots_count: number;
}

export type LeaseType =
	| 'RESIDENTIAL_UNFURNISHED'
	| 'RESIDENTIAL_FURNISHED'
	| 'COMMERCIAL_3_6_9'
	| 'PROFESSIONAL'
	| 'PARKING'
	| 'SHORT_TERM';

export type RentCallStatus = 'PENDING' | 'PARTIAL' | 'PAID' | 'UNPAID';

export interface LeaseDTO {
	readonly id: string;
	readonly legal_entity_id: string;
	readonly lot_id: string;
	readonly tenant_id: string;
	readonly lease_type: LeaseType;
	readonly rent_net_cents: number;
	readonly charges_cents: number;
	readonly subject_to_vat: boolean;
	readonly vat_rate_basis_points: number;
	readonly security_deposit_cents: number;
	readonly due_day_of_month: number;
	readonly gli_policy_number: string;
	readonly guarantor_name: string;
	readonly start_date: string;
	readonly end_date?: string;
	readonly active: boolean;
}

export interface RentCallDTO {
	readonly id: string;
	readonly lease_id: string;
	readonly legal_entity_id: string;
	readonly tenant_id: string;
	readonly period_month: string;
	readonly rent_net_cents: number;
	readonly vat_cents: number;
	readonly charges_cents: number;
	readonly total_due_cents: number;
	readonly remaining_due_cents: number;
	readonly status: RentCallStatus;
	readonly due_date: string;
}

export interface RecordCashPaymentRequest {
	readonly lease_id: string;
	readonly amount_cents: number;
	readonly label: string;
	readonly receipt_number?: string;
	readonly payment_date?: string;
}

export interface RentAllocationDTO {
	readonly rent_call_id: string;
	readonly period_month: string;
	readonly allocated_cents: number;
	readonly remaining_due_cents: number;
	readonly resulting_status: RentCallStatus;
}

export interface PaymentReceiptDTO {
	readonly id: string;
	readonly legal_entity_id: string;
	readonly lease_id: string;
	readonly tenant_id: string;
	readonly amount_cents: number;
	readonly unallocated_cents: number;
	readonly method: 'CASH' | 'BANK_TRANSFER' | 'CHECK' | 'DIRECT_DEBIT' | 'CSV_IMPORT' | 'OPEN_BANKING_DSP2';
	readonly label: string;
	readonly receipt_number?: string;
	readonly payment_date: string;
	readonly allocations: readonly RentAllocationDTO[];
	readonly cash_journal_entry_id?: string;
}

export interface BatchImportResultDTO {
	readonly entity_id: string;
	readonly imported_count: number;
	readonly total_imported_cents: number;
	readonly payments: readonly PaymentReceiptDTO[];
}

export interface NegativePositionDTO {
	readonly lease_id: string;
	readonly legal_entity_id: string;
	readonly lot_id: string;
	readonly tenant_id: string;
	readonly tenant_name: string;
	readonly total_unpaid_cents: number;
	readonly unpaid_months: readonly string[];
	readonly oldest_due_date: string;
	readonly gli_policy_number: string;
	readonly gli_eligible: boolean;
}

export interface GLIDossierArchiveDTO {
	readonly id: string;
	readonly lease_id: string;
	readonly legal_entity_id: string;
	readonly tenant_name: string;
	readonly template_id: string;
	readonly gli_policy_number: string;
	readonly total_unpaid_cents: number;
	readonly unpaid_months: readonly string[];
	readonly archive_r2_key: string;
	readonly download_url: string;
	readonly included_files: readonly string[];
	readonly archive_size_bytes: number;
	readonly created_at: string;
}

export type ConditionState = 'NEW' | 'GOOD' | 'FAIR_WEAR' | 'POOR' | 'OUT_OF_SERVICE';
export type InspectionType = 'MOVE_IN' | 'MOVE_OUT';

export interface InspectionItemInput {
	readonly room_name: string;
	readonly element_name: string;
	readonly condition: ConditionState;
	readonly observation?: string;
	readonly photo_r2_keys?: readonly string[];
}

export interface InspectionItemDTO {
	readonly id: string;
	readonly inspection_id: string;
	readonly room_name: string;
	readonly element_name: string;
	readonly condition: ConditionState;
	readonly observation?: string;
	readonly photo_r2_keys: readonly string[];
}

export interface InspectionDTO {
	readonly id: string;
	readonly lease_id: string;
	readonly template_id: string;
	readonly type: InspectionType;
	readonly entry_inspection_id?: string;
	readonly items: readonly InspectionItemDTO[];
	readonly report_pdf_r2_key?: string;
	readonly report_download_url?: string;
	readonly inspection_date: string;
}

export interface ElementComparisonDTO {
	readonly room_name: string;
	readonly element_name: string;
	readonly entry_condition: ConditionState;
	readonly exit_condition: ConditionState;
	readonly entry_observation?: string;
	readonly exit_observation?: string;
	readonly degraded: boolean;
}

export interface InspectionReportDTO {
	readonly inspection_id: string;
	readonly lease_id: string;
	readonly type: InspectionType;
	readonly entry_inspection_id?: string;
	readonly has_entry_comparison: boolean;
	readonly comparisons?: readonly ElementComparisonDTO[];
	readonly degradations?: readonly ElementComparisonDTO[];
	readonly items: readonly InspectionItemDTO[];
	readonly pdf_r2_key: string;
	readonly download_url: string;
	readonly generated_at: string;
}

export interface ComputedLineDTO {
	readonly line_id: string;
	readonly account_code: string;
	readonly label: string;
	readonly section: string;
	readonly cerfa_code?: string;
	readonly disabled: boolean;
	readonly gross_n_cents: number;
	readonly depreciation_n_cents: number;
	readonly net_n_cents: number;
	readonly net_n_minus_1_cents: number;
	readonly variation_abs_cents: number;
	readonly variation_pct_basis_points: number;
	readonly variation_not_significant: boolean;
}

export interface ComputedFinancialStatementsDTO {
	readonly lines: readonly ComputedLineDTO[];
	readonly active_lines: readonly ComputedLineDTO[];
	readonly disabled_line_ids: readonly string[];
	readonly total_operating_revenue_n_cents: number;
	readonly total_operating_revenue_n_minus_1_cents: number;
	readonly total_external_expenses_n_cents: number;
	readonly total_taxes_and_duties_n_cents: number;
	readonly total_personnel_expenses_n_cents: number;
	readonly total_depreciation_n_cents: number;
	readonly total_other_expenses_n_cents: number;
	readonly total_operating_expenses_n_cents: number;
	readonly total_operating_expenses_n_minus_1_cents: number;
	readonly value_added_n_cents: number;
	readonly value_added_n_minus_1_cents: number;
	readonly gross_operating_surplus_n_cents: number;
	readonly gross_operating_surplus_n_minus_1_cents: number;
	readonly operating_income_n_cents: number;
	readonly operating_income_n_minus_1_cents: number;
	readonly financial_result_n_cents: number;
	readonly ordinary_income_before_tax_n_cents: number;
	readonly extraordinary_result_n_cents: number;
	readonly total_income_tax_n_cents: number;
	readonly total_revenue_n_cents: number;
	readonly total_expenses_n_cents: number;
	readonly net_income_n_cents: number;
	readonly net_income_n_minus_1_cents: number;
	readonly self_financing_capacity_n_cents: number;
	readonly fixed_assets_gross_n_cents: number;
	readonly fixed_assets_depreciation_n_cents: number;
	readonly fixed_assets_net_n_cents: number;
	readonly current_assets_net_n_cents: number;
	readonly total_assets_n_cents: number;
	readonly equity_excl_net_income_n_cents: number;
	readonly balance_sheet_net_income_n_cents: number;
	readonly total_equity_n_cents: number;
	readonly total_liabilities_n_cents: number;
	readonly total_equity_and_liabilities_n_cents: number;
}

export interface AccountingStoredFileDTO {
	readonly id: string;
	readonly legal_entity_id: string;
	readonly fiscal_year: number;
	readonly category: string;
	readonly file_name: string;
	readonly content_type: string;
	readonly r2_key: string;
	readonly size_bytes: number;
	readonly download_url: string;
	readonly disabled_line_ids: readonly string[];
	readonly net_income_n_cents: number;
	readonly created_at: string;
}

export class ApiError extends Error {
	readonly status: number;
	readonly code?: string;

	constructor(status: number, message: string, code?: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.code = code;
	}
}

export interface ApiClientOptions {
	readonly baseUrl?: string;
	readonly accessToken?: string;
	readonly cookieHeader?: string;
	readonly fetchFn?: typeof fetch;
}

const DEFAULT_BASE_URL =
	(typeof process !== 'undefined' && process.env?.PUBLIC_API_BASE_URL) || 'http://localhost:8080';

export function createApiClient(options: ApiClientOptions = {}) {
	const baseUrl = (options.baseUrl ?? DEFAULT_BASE_URL).replace(/\/+$/, '');
	const fetchImpl = options.fetchFn ?? fetch;

	async function requestJson<T>(path: string, init: RequestInit = {}): Promise<T> {
		const headers = new Headers(init.headers);
		if (!headers.has('Content-Type') && init.body) {
			headers.set('Content-Type', 'application/json');
		}
		if (options.accessToken) {
			headers.set('Authorization', `Bearer ${options.accessToken}`);
		}
		if (options.cookieHeader) {
			headers.set('Cookie', options.cookieHeader);
		}

		const response = await fetchImpl(`${baseUrl}${path}`, {
			...init,
			headers,
			credentials: 'include'
		});

		if (!response.ok) {
			let errorMessage = `HTTP ${response.status}`;
			let errorCode: string | undefined;
			try {
				const errBody = (await response.json()) as { error?: string; code?: string };
				if (errBody?.error) {
					errorMessage = errBody.error;
				}
				if (errBody?.code) {
					errorCode = errBody.code;
				}
			} catch {
				// Fallback to status text if non-JSON response
			}
			if (
				response.status === 403 &&
				!errorCode &&
				errorMessage.toLowerCase().includes('not approved')
			) {
				errorCode = 'ACCOUNT_NOT_APPROVED';
			}
			throw new ApiError(response.status, errorMessage, errorCode);
		}

		if (response.status === 204) {
			return undefined as T;
		}

		return (await response.json()) as T;
	}

	return {
		// Auth & Super Admin User Approval
		signup(cmd: SignupRequest): Promise<UserDTO> {
			return requestJson<UserDTO>('/api/v1/auth/signup', {
				method: 'POST',
				body: JSON.stringify(cmd)
			});
		},
		login(cmd: LoginRequest): Promise<AuthTokensResponse> {
			return requestJson<AuthTokensResponse>('/api/v1/auth/login', {
				method: 'POST',
				body: JSON.stringify(cmd)
			});
		},
		listPendingUsers(): Promise<readonly UserDTO[]> {
			return requestJson<readonly UserDTO[]>('/api/v1/admin/users/pending');
		},
		approveUser(userId: string): Promise<UserDTO> {
			return requestJson<UserDTO>(`/api/v1/admin/users/${encodeURIComponent(userId)}/approve`, {
				method: 'POST'
			});
		},
		rejectUser(userId: string): Promise<UserDTO> {
			return requestJson<UserDTO>(`/api/v1/admin/users/${encodeURIComponent(userId)}/reject`, {
				method: 'POST'
			});
		},

		// Portfolio
		listEntities(): Promise<readonly LegalEntityDTO[]> {
			return requestJson<readonly LegalEntityDTO[]>('/api/v1/entities');
		},
		getEntitySummary(entityId: string): Promise<EntityPortfolioSummaryDTO> {
			return requestJson<EntityPortfolioSummaryDTO>(
				`/api/v1/entities/${encodeURIComponent(entityId)}`
			);
		},
		createEntity(cmd: CreateLegalEntityRequest): Promise<LegalEntityDTO> {
			return requestJson<LegalEntityDTO>('/api/v1/entities', {
				method: 'POST',
				body: JSON.stringify(cmd)
			});
		},
		createProperty(entityId: string, cmd: CreatePropertyRequest): Promise<PropertyDTO> {
			return requestJson<PropertyDTO>(
				`/api/v1/entities/${encodeURIComponent(entityId)}/properties`,
				{
					method: 'POST',
					body: JSON.stringify(cmd)
				}
			);
		},

		// Leases
		listLeases(entityId?: string): Promise<readonly LeaseDTO[]> {
			const query = entityId ? `?legal_entity_id=${encodeURIComponent(entityId)}` : '';
			return requestJson<readonly LeaseDTO[]>(`/api/v1/leases${query}`);
		},
		generateRentCalls(entityId: string, periodMonth: string): Promise<readonly RentCallDTO[]> {
			return requestJson<readonly RentCallDTO[]>(
				`/api/v1/entities/${encodeURIComponent(entityId)}/rent-calls/generate`,
				{
					method: 'POST',
					body: JSON.stringify({ period_month: periodMonth })
				}
			);
		},

		// Payments & Negative Positions
		recordCashPayment(cmd: RecordCashPaymentRequest): Promise<PaymentReceiptDTO> {
			return requestJson<PaymentReceiptDTO>('/api/v1/payments/cash', {
				method: 'POST',
				body: JSON.stringify(cmd)
			});
		},
		importPaymentsCSV(entityId: string, csvContent: string): Promise<BatchImportResultDTO> {
			return requestJson<BatchImportResultDTO>('/api/v1/payments/import-csv', {
				method: 'POST',
				body: JSON.stringify({
					entity_id: entityId,
					csv_content: csvContent
				})
			});
		},
		listNegativePositions(entityId: string): Promise<readonly NegativePositionDTO[]> {
			return requestJson<readonly NegativePositionDTO[]>(
				`/api/v1/entities/${encodeURIComponent(entityId)}/negative-positions`
			);
		},

		// Documents & 1-click GLI Dossier
		assembleGLIDossier(leaseId: string, templateId?: string): Promise<GLIDossierArchiveDTO> {
			return requestJson<GLIDossierArchiveDTO>(
				`/api/v1/leases/${encodeURIComponent(leaseId)}/gli-dossier`,
				{
					method: 'POST',
					body: JSON.stringify({ template_id: templateId ?? '' })
				}
			);
		},

		// Inspections
		getInspection(inspectionId: string): Promise<InspectionDTO> {
			return requestJson<InspectionDTO>(`/api/v1/inspections/${encodeURIComponent(inspectionId)}`);
		},
		upsertInspectionItems(
			inspectionId: string,
			items: readonly InspectionItemInput[]
		): Promise<InspectionDTO> {
			return requestJson<InspectionDTO>(
				`/api/v1/inspections/${encodeURIComponent(inspectionId)}/items`,
				{
					method: 'PUT',
					body: JSON.stringify({ items })
				}
			);
		},
		generateInspectionPDF(inspectionId: string): Promise<InspectionReportDTO> {
			return requestJson<InspectionReportDTO>(
				`/api/v1/inspections/${encodeURIComponent(inspectionId)}/pdf`,
				{
					method: 'POST'
				}
			);
		},

		// Accounting, Liasse Fiscale & FEC
		previewFinancialStatements(
			entityId: string,
			fiscalYear: number,
			disabledLineIds: readonly string[]
		): Promise<ComputedFinancialStatementsDTO> {
			return requestJson<ComputedFinancialStatementsDTO>(
				`/api/v1/entities/${encodeURIComponent(entityId)}/statements/preview`,
				{
					method: 'POST',
					body: JSON.stringify({
						fiscal_year: fiscalYear,
						disabled_line_ids: disabledLineIds
					})
				}
			);
		},
		generateAnnualAccountsPDF(
			entityId: string,
			fiscalYear: number,
			disabledLineIds: readonly string[]
		): Promise<AccountingStoredFileDTO> {
			return requestJson<AccountingStoredFileDTO>(
				`/api/v1/entities/${encodeURIComponent(entityId)}/statements/pdf`,
				{
					method: 'POST',
					body: JSON.stringify({
						fiscal_year: fiscalYear,
						disabled_line_ids: disabledLineIds
					})
				}
			);
		},
		async exportFEC(entityId: string, fiscalYear: number): Promise<string> {
			const response = await fetchImpl(
				`${baseUrl}/api/v1/entities/${encodeURIComponent(entityId)}/fec?fiscal_year=${fiscalYear}`,
				{
					headers: options.accessToken
						? { Authorization: `Bearer ${options.accessToken}` }
						: undefined,
					credentials: 'include'
				}
			);
			if (!response.ok) {
				throw new ApiError(response.status, `Failed to export FEC (HTTP ${response.status})`);
			}
			return response.text();
		}
	};
}
