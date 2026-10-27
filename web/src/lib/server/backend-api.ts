import type { RequestEvent } from '@sveltejs/kit';
import { FetchHttpClient, type HttpClient } from '#lib/core/api/http-client';
import type {
	AccountBalanceLineDTO,
	AnnualAccountsReportDTO,
	ComputedFinancialStatementsDTO,
	JournalEntryDTO,
	RecordJournalEntryCmd
} from '#lib/modules/accounting/types';
import type {
	AuthTokensResponse,
	CreateMCPTokenRequest,
	LoginRequest,
	MCPTokenResponse,
	SignupRequest,
	UserDTO
} from '#lib/modules/auth/types';
import type {
	DocumentTemplateDTO,
	GenerateDocCmd,
	GLIDossierArchiveDTO,
	RegisterTemplateCmd,
	StoredFileDTO
} from '#lib/modules/document/types';
import type {
	CreateIncidentCmd,
	CreateInspectionCmd,
	CreateInspectionTemplateCmd,
	IncidentDTO,
	InspectionDTO,
	InspectionItemInput,
	InspectionReportDTO,
	InspectionTemplateDTO,
	UpdateIncidentCmd
} from '#lib/modules/inspection/types';
import type {
	CreateLeaseCmd,
	CreateTenantCmd,
	LeaseDTO,
	RentCallDTO,
	TenantDTO
} from '#lib/modules/lease/types';
import type {
	BatchImportResultDTO,
	CashJournalEntryDTO,
	NegativePositionDTO,
	PaymentReceiptDTO,
	RecordBankPaymentCmd,
	RecordCashDepositCmd,
	RecordCashPaymentCmd
} from '#lib/modules/payment/types';
import type {
	CreateLegalEntityCmd,
	CreateLotCmd,
	CreatePropertyCmd,
	EntityPortfolioSummaryDTO,
	LegalEntityDTO,
	LotDTO,
	PropertyDTO,
	ShareholderDTO,
	UpsertShareholderCmd
} from '#lib/modules/portfolio/types';
import { clearSessionCookies, getSessionFromCookies, setSessionCookies } from './cookies';

export interface BackendApi {
	readonly auth: {
		signup(payload: SignupRequest): Promise<UserDTO>;
		login(payload: LoginRequest): Promise<AuthTokensResponse>;
		refresh(refreshToken: string): Promise<AuthTokensResponse>;
		logout(refreshToken: string): Promise<void>;
		createMcpToken(payload: CreateMCPTokenRequest): Promise<MCPTokenResponse>;
		listPendingUsers(): Promise<UserDTO[]>;
		approveUser(userId: string): Promise<UserDTO>;
		rejectUser(userId: string): Promise<UserDTO>;
	};
	readonly portfolio: {
		listEntities(): Promise<LegalEntityDTO[]>;
		createEntity(cmd: CreateLegalEntityCmd): Promise<LegalEntityDTO>;
		getEntitySummary(entityId: string): Promise<EntityPortfolioSummaryDTO>;
		upsertShareholder(entityId: string, cmd: UpsertShareholderCmd): Promise<ShareholderDTO>;
		listProperties(entityId: string): Promise<PropertyDTO[]>;
		createProperty(entityId: string, cmd: CreatePropertyCmd): Promise<PropertyDTO>;
		createLot(propertyId: string, cmd: CreateLotCmd): Promise<LotDTO>;
	};
	readonly lease: {
		listTenants(includeArchived?: boolean): Promise<TenantDTO[]>;
		createTenant(cmd: CreateTenantCmd): Promise<TenantDTO>;
		archiveTenant(tenantId: string): Promise<TenantDTO>;
		listLeases(entityId?: string): Promise<LeaseDTO[]>;
		createLease(cmd: CreateLeaseCmd): Promise<LeaseDTO>;
		generateRentCalls(entityId: string, periodMonth: string): Promise<RentCallDTO[]>;
		listEntityRentCalls(entityId: string, periodMonth?: string): Promise<RentCallDTO[]>;
	};
	readonly payment: {
		recordCashPayment(cmd: RecordCashPaymentCmd): Promise<PaymentReceiptDTO>;
		recordBankPayment(cmd: RecordBankPaymentCmd): Promise<PaymentReceiptDTO>;
		recordCashDeposit(cmd: RecordCashDepositCmd): Promise<CashJournalEntryDTO>;
		importPaymentsCsv(entityId: string, csvContent: string): Promise<BatchImportResultDTO>;
		listNegativePositions(entityId: string): Promise<NegativePositionDTO[]>;
		listCashJournal(entityId: string): Promise<CashJournalEntryDTO[]>;
	};
	readonly document: {
		listTemplates(): Promise<DocumentTemplateDTO[]>;
		registerTemplate(cmd: RegisterTemplateCmd): Promise<DocumentTemplateDTO>;
		generateFromTemplate(cmd: GenerateDocCmd): Promise<StoredFileDTO>;
		uploadFile(params: {
			legalEntityId?: string;
			leaseId?: string;
			category: string;
			file: File;
		}): Promise<StoredFileDTO>;
		assembleGliDossier(leaseId: string, templateId?: string): Promise<GLIDossierArchiveDTO>;
		listLeaseDocuments(leaseId: string): Promise<StoredFileDTO[]>;
	};
	readonly inspection: {
		listTemplates(): Promise<InspectionTemplateDTO[]>;
		createTemplate(cmd: CreateInspectionTemplateCmd): Promise<InspectionTemplateDTO>;
		getTemplate(templateId: string): Promise<InspectionTemplateDTO>;
		listInspections(leaseId: string): Promise<InspectionDTO[]>;
		createInspection(cmd: CreateInspectionCmd): Promise<InspectionDTO>;
		getInspection(inspectionId: string): Promise<InspectionDTO>;
		upsertInspectionItems(
			inspectionId: string,
			items: InspectionItemInput[]
		): Promise<InspectionDTO>;
		generateInspectionPdf(inspectionId: string): Promise<InspectionReportDTO>;
		listIncidents(filters?: {
			propertyId?: string;
			lotId?: string;
			leaseId?: string;
			status?: string;
		}): Promise<IncidentDTO[]>;
		createIncident(cmd: CreateIncidentCmd): Promise<IncidentDTO>;
		getIncident(incidentId: string): Promise<IncidentDTO>;
		updateIncident(incidentId: string, cmd: UpdateIncidentCmd): Promise<IncidentDTO>;
	};
	readonly accounting: {
		listJournalEntries(entityId: string, fiscalYear: number): Promise<JournalEntryDTO[]>;
		recordJournalEntry(entityId: string, cmd: RecordJournalEntryCmd): Promise<JournalEntryDTO>;
		upsertBalanceLines(
			entityId: string,
			fiscalYear: number,
			lines: AccountBalanceLineDTO[]
		): Promise<void>;
		previewStatements(
			entityId: string,
			fiscalYear: number,
			disabledLineIds: string[]
		): Promise<ComputedFinancialStatementsDTO>;
		generateAnnualAccountsPdf(
			entityId: string,
			fiscalYear: number,
			disabledLineIds: string[]
		): Promise<AnnualAccountsReportDTO>;
		exportFecTsv(entityId: string, fiscalYear: number): Promise<string>;
	};
}

function createDomainServices(http: HttpClient): BackendApi {
	return {
		auth: {
			signup(payload) {
				return http.requestJson<UserDTO>('/api/v1/auth/signup', {
					method: 'POST',
					body: JSON.stringify(payload)
				});
			},
			login(payload) {
				return http.requestJson<AuthTokensResponse>('/api/v1/auth/login', {
					method: 'POST',
					body: JSON.stringify(payload)
				});
			},
			refresh(refreshToken) {
				return http.requestJson<AuthTokensResponse>('/api/v1/auth/refresh', {
					method: 'POST',
					body: JSON.stringify({ refresh_token: refreshToken })
				});
			},
			logout(refreshToken) {
				return http.requestJson<void>('/api/v1/auth/logout', {
					method: 'POST',
					body: JSON.stringify({ refresh_token: refreshToken })
				});
			},
			createMcpToken(payload) {
				return http.requestJson<MCPTokenResponse>('/api/v1/auth/mcp-tokens', {
					method: 'POST',
					body: JSON.stringify(payload)
				});
			},
			listPendingUsers() {
				return http.requestJson<UserDTO[]>('/api/v1/admin/users/pending');
			},
			approveUser(userId) {
				return http.requestJson<UserDTO>(
					`/api/v1/admin/users/${encodeURIComponent(userId)}/approve`,
					{ method: 'POST' }
				);
			},
			rejectUser(userId) {
				return http.requestJson<UserDTO>(
					`/api/v1/admin/users/${encodeURIComponent(userId)}/reject`,
					{ method: 'POST' }
				);
			}
		},
		portfolio: {
			listEntities() {
				return http.requestJson<LegalEntityDTO[]>('/api/v1/entities');
			},
			createEntity(cmd) {
				return http.requestJson<LegalEntityDTO>('/api/v1/entities', {
					method: 'POST',
					body: JSON.stringify(cmd)
				});
			},
			getEntitySummary(entityId) {
				return http.requestJson<EntityPortfolioSummaryDTO>(
					`/api/v1/entities/${encodeURIComponent(entityId)}`
				);
			},
			upsertShareholder(entityId, cmd) {
				return http.requestJson<ShareholderDTO>(
					`/api/v1/entities/${encodeURIComponent(entityId)}/shareholders`,
					{
						method: 'POST',
						body: JSON.stringify({ ...cmd, legal_entity_id: entityId })
					}
				);
			},
			listProperties(entityId) {
				return http.requestJson<PropertyDTO[]>(
					`/api/v1/entities/${encodeURIComponent(entityId)}/properties`
				);
			},
			createProperty(entityId, cmd) {
				return http.requestJson<PropertyDTO>(
					`/api/v1/entities/${encodeURIComponent(entityId)}/properties`,
					{
						method: 'POST',
						body: JSON.stringify({ ...cmd, legal_entity_id: entityId })
					}
				);
			},
			createLot(propertyId, cmd) {
				return http.requestJson<LotDTO>(
					`/api/v1/properties/${encodeURIComponent(propertyId)}/lots`,
					{
						method: 'POST',
						body: JSON.stringify({ ...cmd, property_id: propertyId })
					}
				);
			}
		},
		lease: {
			listTenants(includeArchived = false) {
				const query = includeArchived ? '?include_archived=true' : '';
				return http.requestJson<TenantDTO[]>(`/api/v1/tenants${query}`);
			},
			createTenant(cmd) {
				return http.requestJson<TenantDTO>('/api/v1/tenants', {
					method: 'POST',
					body: JSON.stringify(cmd)
				});
			},
			archiveTenant(tenantId) {
				return http.requestJson<TenantDTO>(
					`/api/v1/tenants/${encodeURIComponent(tenantId)}/archive`,
					{ method: 'POST' }
				);
			},
			listLeases(entityId) {
				const query =
					entityId && entityId !== 'ALL' ? `?entity_id=${encodeURIComponent(entityId)}` : '';
				return http.requestJson<LeaseDTO[]>(`/api/v1/leases${query}`);
			},
			createLease(cmd) {
				return http.requestJson<LeaseDTO>('/api/v1/leases', {
					method: 'POST',
					body: JSON.stringify(cmd)
				});
			},
			generateRentCalls(entityId, periodMonth) {
				return http.requestJson<RentCallDTO[]>(
					`/api/v1/entities/${encodeURIComponent(entityId)}/rent-calls/generate`,
					{
						method: 'POST',
						body: JSON.stringify({ period_month: periodMonth })
					}
				);
			},
			listEntityRentCalls(entityId, periodMonth) {
				const query = periodMonth ? `?period_month=${encodeURIComponent(periodMonth)}` : '';
				return http.requestJson<RentCallDTO[]>(
					`/api/v1/entities/${encodeURIComponent(entityId)}/rent-calls${query}`
				);
			}
		},
		payment: {
			recordCashPayment(cmd) {
				return http.requestJson<PaymentReceiptDTO>('/api/v1/payments/cash', {
					method: 'POST',
					body: JSON.stringify(cmd)
				});
			},
			recordBankPayment(cmd) {
				return http.requestJson<PaymentReceiptDTO>('/api/v1/payments/bank', {
					method: 'POST',
					body: JSON.stringify(cmd)
				});
			},
			recordCashDeposit(cmd) {
				return http.requestJson<CashJournalEntryDTO>('/api/v1/payments/cash-deposit', {
					method: 'POST',
					body: JSON.stringify(cmd)
				});
			},
			importPaymentsCsv(entityId, csvContent) {
				return http.requestJson<BatchImportResultDTO>('/api/v1/payments/import-csv', {
					method: 'POST',
					body: JSON.stringify({
						entity_id: entityId,
						csv_content: csvContent
					})
				});
			},
			listNegativePositions(entityId) {
				return http.requestJson<NegativePositionDTO[]>(
					`/api/v1/entities/${encodeURIComponent(entityId)}/negative-positions`
				);
			},
			listCashJournal(entityId) {
				return http.requestJson<CashJournalEntryDTO[]>(
					`/api/v1/entities/${encodeURIComponent(entityId)}/cash-journal`
				);
			}
		},
		document: {
			listTemplates() {
				return http.requestJson<DocumentTemplateDTO[]>('/api/v1/document-templates');
			},
			registerTemplate(cmd) {
				return http.requestJson<DocumentTemplateDTO>('/api/v1/document-templates', {
					method: 'POST',
					body: JSON.stringify(cmd)
				});
			},
			generateFromTemplate(cmd) {
				return http.requestJson<StoredFileDTO>('/api/v1/documents/generate', {
					method: 'POST',
					body: JSON.stringify(cmd)
				});
			},
			uploadFile(params) {
				const formData = new FormData();
				if (params.legalEntityId) {
					formData.set('legal_entity_id', params.legalEntityId);
				}
				if (params.leaseId) {
					formData.set('lease_id', params.leaseId);
				}
				formData.set('category', params.category);
				formData.set('file', params.file);
				return http.requestMultipart<StoredFileDTO>('/api/v1/documents/upload', formData);
			},
			assembleGliDossier(leaseId, templateId) {
				return http.requestJson<GLIDossierArchiveDTO>(
					`/api/v1/leases/${encodeURIComponent(leaseId)}/gli-dossier`,
					{
						method: 'POST',
						body: JSON.stringify({ template_id: templateId ?? '' })
					}
				);
			},
			listLeaseDocuments(leaseId) {
				return http.requestJson<StoredFileDTO[]>(
					`/api/v1/leases/${encodeURIComponent(leaseId)}/documents`
				);
			}
		},
		inspection: {
			listTemplates() {
				return http.requestJson<InspectionTemplateDTO[]>('/api/v1/inspection-templates');
			},
			createTemplate(cmd) {
				return http.requestJson<InspectionTemplateDTO>('/api/v1/inspection-templates', {
					method: 'POST',
					body: JSON.stringify(cmd)
				});
			},
			getTemplate(templateId) {
				return http.requestJson<InspectionTemplateDTO>(
					`/api/v1/inspection-templates/${encodeURIComponent(templateId)}`
				);
			},
			listInspections(leaseId) {
				return http.requestJson<InspectionDTO[]>(
					`/api/v1/inspections?lease_id=${encodeURIComponent(leaseId)}`
				);
			},
			createInspection(cmd) {
				return http.requestJson<InspectionDTO>('/api/v1/inspections', {
					method: 'POST',
					body: JSON.stringify(cmd)
				});
			},
			getInspection(inspectionId) {
				return http.requestJson<InspectionDTO>(
					`/api/v1/inspections/${encodeURIComponent(inspectionId)}`
				);
			},
			upsertInspectionItems(inspectionId, items) {
				return http.requestJson<InspectionDTO>(
					`/api/v1/inspections/${encodeURIComponent(inspectionId)}/items`,
					{
						method: 'PUT',
						body: JSON.stringify({ items })
					}
				);
			},
			generateInspectionPdf(inspectionId) {
				return http.requestJson<InspectionReportDTO>(
					`/api/v1/inspections/${encodeURIComponent(inspectionId)}/pdf`,
					{ method: 'POST' }
				);
			},
			listIncidents(filters) {
				const params = new URLSearchParams();
				if (filters?.propertyId) params.set('property_id', filters.propertyId);
				if (filters?.lotId) params.set('lot_id', filters.lotId);
				if (filters?.leaseId) params.set('lease_id', filters.leaseId);
				if (filters?.status) params.set('status', filters.status);
				const query = params.toString();
				return http.requestJson<IncidentDTO[]>(`/api/v1/incidents${query ? `?${query}` : ''}`);
			},
			createIncident(cmd) {
				return http.requestJson<IncidentDTO>('/api/v1/incidents', {
					method: 'POST',
					body: JSON.stringify(cmd)
				});
			},
			getIncident(incidentId) {
				return http.requestJson<IncidentDTO>(`/api/v1/incidents/${encodeURIComponent(incidentId)}`);
			},
			updateIncident(incidentId, cmd) {
				return http.requestJson<IncidentDTO>(
					`/api/v1/incidents/${encodeURIComponent(incidentId)}`,
					{
						method: 'PATCH',
						body: JSON.stringify(cmd)
					}
				);
			}
		},
		accounting: {
			listJournalEntries(entityId, fiscalYear) {
				return http.requestJson<JournalEntryDTO[]>(
					`/api/v1/entities/${encodeURIComponent(entityId)}/journal-entries?fiscal_year=${fiscalYear}`
				);
			},
			recordJournalEntry(entityId, cmd) {
				return http.requestJson<JournalEntryDTO>(
					`/api/v1/entities/${encodeURIComponent(entityId)}/journal-entries`,
					{
						method: 'POST',
						body: JSON.stringify({ ...cmd, legal_entity_id: entityId })
					}
				);
			},
			upsertBalanceLines(entityId, fiscalYear, lines) {
				return http.requestJson<void>(
					`/api/v1/entities/${encodeURIComponent(entityId)}/statements/lines`,
					{
						method: 'PUT',
						body: JSON.stringify({
							fiscal_year: fiscalYear,
							lines
						})
					}
				);
			},
			previewStatements(entityId, fiscalYear, disabledLineIds) {
				return http.requestJson<ComputedFinancialStatementsDTO>(
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
			generateAnnualAccountsPdf(entityId, fiscalYear, disabledLineIds) {
				return http.requestJson<AnnualAccountsReportDTO>(
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
			exportFecTsv(entityId, fiscalYear) {
				return http.requestText(
					`/api/v1/entities/${encodeURIComponent(entityId)}/fec?fiscal_year=${fiscalYear}`
				);
			}
		}
	};
}

export function createBackendApi(event: RequestEvent): BackendApi {
	const platformEnv = event.platform?.env as Record<string, string | undefined> | undefined;
	const nodeEnv = typeof process !== 'undefined' ? process.env : undefined;
	const baseUrl =
		platformEnv?.API_BASE_URL ||
		nodeEnv?.API_BASE_URL ||
		platformEnv?.PUBLIC_API_BASE_URL ||
		nodeEnv?.PUBLIC_API_BASE_URL ||
		'http://127.0.0.1:8080';

	const cfClientId = platformEnv?.CF_ACCESS_CLIENT_ID || nodeEnv?.CF_ACCESS_CLIENT_ID;
	const cfClientSecret = platformEnv?.CF_ACCESS_CLIENT_SECRET || nodeEnv?.CF_ACCESS_CLIENT_SECRET;

	const defaultHeaders: Record<string, string> = {};
	if (cfClientId && cfClientSecret) {
		defaultHeaders['CF-Access-Client-Id'] = cfClientId;
		defaultHeaders['CF-Access-Client-Secret'] = cfClientSecret;
	}

	const initialSession = getSessionFromCookies(event.cookies);
	let currentAccessToken = initialSession.accessToken;
	let currentRefreshToken = initialSession.refreshToken;
	const isSecure = event.url.protocol === 'https:';

	const http = new FetchHttpClient({
		baseUrl,
		fetchFn: event.fetch,
		defaultHeaders,
		getAccessToken: () => currentAccessToken,
		refreshSession: async () => {
			if (!currentRefreshToken) {
				return null;
			}
			try {
				const refreshUrl = `${baseUrl.replace(/\/$/, '')}/api/v1/auth/refresh`;
				const response = await event.fetch(refreshUrl, {
					method: 'POST',
					headers: {
						...defaultHeaders,
						Accept: 'application/json',
						'Content-Type': 'application/json'
					},
					body: JSON.stringify({ refresh_token: currentRefreshToken })
				});
				if (!response.ok) {
					return null;
				}
				const refreshed = (await response.json()) as AuthTokensResponse;
				currentAccessToken = refreshed.access_token;
				currentRefreshToken = refreshed.refresh_token;
				event.locals.user = refreshed.user;
				setSessionCookies(event.cookies, refreshed, isSecure);
				return currentAccessToken;
			} catch {
				return null;
			}
		},
		onUnauthorized: () => {
			currentAccessToken = null;
			currentRefreshToken = null;
			event.locals.user = null;
			clearSessionCookies(event.cookies);
		}
	});

	return createDomainServices(http);
}
