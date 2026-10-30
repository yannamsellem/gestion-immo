import type { RequestEvent } from '@sveltejs/kit';
import createClient from 'openapi-fetch';
import { ApiError } from '#lib/core/api/api-error';
import type { paths } from '#lib/core/api/openapi';
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
		archiveTenant(tenantId: string): Promise<void>;
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

type OpenApiClient = ReturnType<typeof createClient<paths>>;

function extractErrorMessage(error: unknown, response: Response): string {
	if (error && typeof error === 'object') {
		const record = error as Record<string, unknown>;
		if (typeof record.error === 'string' && record.error.trim().length > 0) {
			return record.error;
		}
		if (typeof record.message === 'string' && record.message.trim().length > 0) {
			return record.message;
		}
		if (typeof record.detail === 'string' && record.detail.trim().length > 0) {
			return record.detail;
		}
	}
	return response.statusText || `HTTP ${response.status}`;
}

async function unwrap<T>(
	promise: Promise<{ data?: T; error?: unknown; response: Response }>
): Promise<T> {
	const { data, error, response } = await promise;
	if (!response.ok || error !== undefined) {
		throw new ApiError(response.status, extractErrorMessage(error, response));
	}
	return data as T;
}

async function unwrapVoid(
	promise: Promise<{ data?: unknown; error?: unknown; response: Response }>
): Promise<void> {
	const { error, response } = await promise;
	if (!response.ok || error !== undefined) {
		throw new ApiError(response.status, extractErrorMessage(error, response));
	}
}

function arrayBufferToBase64(buffer: ArrayBuffer): string {
	const bytes = new Uint8Array(buffer);
	let binary = '';
	for (let i = 0; i < bytes.byteLength; i++) {
		binary += String.fromCharCode(bytes[i]);
	}
	return btoa(binary);
}

function createDomainServices(client: OpenApiClient): BackendApi {
	return {
		auth: {
			signup(payload) {
				return unwrap(client.POST('/api/v1/auth/signup', { body: payload }));
			},
			login(payload) {
				return unwrap(client.POST('/api/v1/auth/login', { body: payload }));
			},
			refresh(refreshToken) {
				return unwrap(
					client.POST('/api/v1/auth/refresh', {
						body: { refresh_token: refreshToken }
					})
				);
			},
			async logout(refreshToken) {
				await unwrap(
					client.POST('/api/v1/auth/logout', {
						body: { refresh_token: refreshToken }
					})
				);
			},
			createMcpToken(payload) {
				return unwrap(client.POST('/api/v1/auth/mcp-tokens', { body: payload }));
			},
			listPendingUsers() {
				return unwrap(client.GET('/api/v1/admin/users/pending'));
			},
			approveUser(userId) {
				return unwrap(
					client.POST('/api/v1/admin/users/{id}/approve', {
						params: { path: { id: userId } }
					})
				);
			},
			rejectUser(userId) {
				return unwrap(
					client.POST('/api/v1/admin/users/{id}/reject', {
						params: { path: { id: userId } }
					})
				);
			}
		},
		portfolio: {
			listEntities() {
				return unwrap(client.GET('/api/v1/entities'));
			},
			createEntity(cmd) {
				return unwrap(client.POST('/api/v1/entities', { body: cmd }));
			},
			getEntitySummary(entityId) {
				return unwrap(
					client.GET('/api/v1/entities/{id}', {
						params: { path: { id: entityId } }
					})
				);
			},
			upsertShareholder(entityId, cmd) {
				return unwrap(
					client.POST('/api/v1/entities/{id}/shareholders', {
						params: { path: { id: entityId } },
						body: { ...cmd, legal_entity_id: entityId }
					})
				);
			},
			listProperties(entityId) {
				return unwrap(
					client.GET('/api/v1/entities/{id}/properties', {
						params: { path: { id: entityId } }
					})
				);
			},
			createProperty(entityId, cmd) {
				return unwrap(
					client.POST('/api/v1/entities/{id}/properties', {
						params: { path: { id: entityId } },
						body: { ...cmd, legal_entity_id: entityId }
					})
				);
			},
			createLot(propertyId, cmd) {
				return unwrap(
					client.POST('/api/v1/properties/{id}/lots', {
						params: { path: { id: propertyId } },
						body: { ...cmd, property_id: propertyId }
					})
				);
			}
		},
		lease: {
			listTenants(includeArchived = false) {
				return unwrap(
					client.GET('/api/v1/tenants', {
						params: {
							query: includeArchived ? { include_archived: 'true' } : undefined
						}
					})
				);
			},
			createTenant(cmd) {
				return unwrap(client.POST('/api/v1/tenants', { body: cmd }));
			},
			archiveTenant(tenantId) {
				return unwrapVoid(
					client.POST('/api/v1/tenants/{id}/archive', {
						params: { path: { id: tenantId } }
					})
				);
			},
			listLeases(entityId) {
				return unwrap(
					client.GET('/api/v1/leases', {
						params: {
							query: entityId && entityId !== 'ALL' ? { entity_id: entityId } : undefined
						}
					})
				);
			},
			createLease(cmd) {
				return unwrap(client.POST('/api/v1/leases', { body: cmd }));
			},
			generateRentCalls(entityId, periodMonth) {
				return unwrap(
					client.POST('/api/v1/entities/{id}/rent-calls/generate', {
						params: { path: { id: entityId } },
						body: { period_month: periodMonth }
					})
				);
			},
			listEntityRentCalls(entityId, periodMonth) {
				return unwrap(
					client.GET('/api/v1/entities/{id}/rent-calls', {
						params: {
							path: { id: entityId },
							query: periodMonth ? { period_month: periodMonth } : undefined
						}
					})
				);
			}
		},
		payment: {
			recordCashPayment(cmd) {
				return unwrap(client.POST('/api/v1/payments/cash', { body: cmd }));
			},
			recordBankPayment(cmd) {
				return unwrap(client.POST('/api/v1/payments/bank', { body: cmd }));
			},
			recordCashDeposit(cmd) {
				return unwrap(client.POST('/api/v1/payments/cash-deposit', { body: cmd }));
			},
			importPaymentsCsv(entityId, csvContent) {
				return unwrap(
					client.POST('/api/v1/payments/import-csv', {
						body: {
							entity_id: entityId,
							csv_content: csvContent
						}
					})
				);
			},
			listNegativePositions(entityId) {
				return unwrap(
					client.GET('/api/v1/entities/{id}/negative-positions', {
						params: { path: { id: entityId } }
					})
				);
			},
			listCashJournal(entityId) {
				return unwrap(
					client.GET('/api/v1/entities/{id}/cash-journal', {
						params: { path: { id: entityId } }
					})
				);
			}
		},
		document: {
			listTemplates() {
				return unwrap(client.GET('/api/v1/document-templates'));
			},
			registerTemplate(cmd) {
				return unwrap(client.POST('/api/v1/document-templates', { body: cmd }));
			},
			generateFromTemplate(cmd) {
				return unwrap(client.POST('/api/v1/documents/generate', { body: cmd }));
			},
			async uploadFile(params) {
				const buffer = await params.file.arrayBuffer();
				return unwrap(
					client.POST('/api/v1/documents/upload', {
						body: {
							legal_entity_id: params.legalEntityId,
							lease_id: params.leaseId,
							category: params.category,
							file_name: params.file.name,
							content_type: params.file.type || 'application/octet-stream',
							content_base64: arrayBufferToBase64(buffer)
						}
					})
				);
			},
			assembleGliDossier(leaseId, templateId) {
				return unwrap(
					client.POST('/api/v1/leases/{id}/gli-dossier', {
						params: { path: { id: leaseId } },
						body: { template_id: templateId ?? '' }
					})
				);
			},
			listLeaseDocuments(leaseId) {
				return unwrap(
					client.GET('/api/v1/leases/{id}/documents', {
						params: { path: { id: leaseId } }
					})
				);
			}
		},
		inspection: {
			listTemplates() {
				return unwrap(client.GET('/api/v1/inspection-templates'));
			},
			createTemplate(cmd) {
				return unwrap(client.POST('/api/v1/inspection-templates', { body: cmd }));
			},
			getTemplate(templateId) {
				return unwrap(
					client.GET('/api/v1/inspection-templates/{id}', {
						params: { path: { id: templateId } }
					})
				);
			},
			listInspections(leaseId) {
				return unwrap(
					client.GET('/api/v1/inspections', {
						params: { query: { lease_id: leaseId } }
					})
				);
			},
			createInspection(cmd) {
				return unwrap(client.POST('/api/v1/inspections', { body: cmd }));
			},
			getInspection(inspectionId) {
				return unwrap(
					client.GET('/api/v1/inspections/{id}', {
						params: { path: { id: inspectionId } }
					})
				);
			},
			upsertInspectionItems(inspectionId, items) {
				return unwrap(
					client.PUT('/api/v1/inspections/{id}/items', {
						params: { path: { id: inspectionId } },
						body: { items }
					})
				);
			},
			generateInspectionPdf(inspectionId) {
				return unwrap(
					client.POST('/api/v1/inspections/{id}/pdf', {
						params: { path: { id: inspectionId } }
					})
				);
			},
			listIncidents(filters) {
				return unwrap(
					client.GET('/api/v1/incidents', {
						params: {
							query: {
								property_id: filters?.propertyId,
								lease_id: filters?.leaseId
							}
						}
					})
				);
			},
			createIncident(cmd) {
				return unwrap(client.POST('/api/v1/incidents', { body: cmd }));
			},
			getIncident(incidentId) {
				return unwrap(
					client.GET('/api/v1/incidents/{id}', {
						params: { path: { id: incidentId } }
					})
				);
			},
			updateIncident(incidentId, cmd) {
				return unwrap(
					client.PATCH('/api/v1/incidents/{id}', {
						params: { path: { id: incidentId } },
						body: cmd
					})
				);
			}
		},
		accounting: {
			listJournalEntries(entityId, fiscalYear) {
				return unwrap(
					client.GET('/api/v1/entities/{id}/journal-entries', {
						params: {
							path: { id: entityId },
							query: { fiscal_year: String(fiscalYear) }
						}
					})
				);
			},
			recordJournalEntry(entityId, cmd) {
				return unwrap(
					client.POST('/api/v1/entities/{id}/journal-entries', {
						params: { path: { id: entityId } },
						body: { ...cmd, legal_entity_id: entityId }
					})
				);
			},
			upsertBalanceLines(entityId, fiscalYear, lines) {
				return unwrapVoid(
					client.PUT('/api/v1/entities/{id}/statements/lines', {
						params: { path: { id: entityId } },
						body: {
							fiscal_year: fiscalYear,
							lines
						}
					})
				);
			},
			previewStatements(entityId, fiscalYear, disabledLineIds) {
				return unwrap(
					client.POST('/api/v1/entities/{id}/statements/preview', {
						params: { path: { id: entityId } },
						body: {
							fiscal_year: fiscalYear,
							disabled_line_ids: disabledLineIds
						}
					})
				);
			},
			generateAnnualAccountsPdf(entityId, fiscalYear, disabledLineIds) {
				return unwrap(
					client.POST('/api/v1/entities/{id}/statements/pdf', {
						params: { path: { id: entityId } },
						body: {
							fiscal_year: fiscalYear,
							disabled_line_ids: disabledLineIds
						}
					})
				);
			},
			exportFecTsv(entityId, fiscalYear) {
				return unwrap(
					client.GET('/api/v1/entities/{id}/fec', {
						params: {
							path: { id: entityId },
							query: { fiscal_year: String(fiscalYear) }
						},
						parseAs: 'text'
					})
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
	let refreshInFlight: Promise<string | null> | null = null;

	const refreshSession = async (): Promise<string | null> => {
		if (!currentRefreshToken) {
			return null;
		}
		if (refreshInFlight) {
			return refreshInFlight;
		}
		refreshInFlight = (async () => {
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
			} finally {
				refreshInFlight = null;
			}
		})();
		return refreshInFlight;
	};

	const clearSession = () => {
		currentAccessToken = null;
		currentRefreshToken = null;
		event.locals.user = null;
		clearSessionCookies(event.cookies);
	};

	const requestClones = new WeakMap<Request, Request>();

	const client = createClient<paths>({
		baseUrl: baseUrl.replace(/\/$/, ''),
		fetch: event.fetch
	});

	client.use({
		async onRequest({ request }) {
			for (const [key, value] of Object.entries(defaultHeaders)) {
				if (!request.headers.has(key)) {
					request.headers.set(key, value);
				}
			}
			const isPublicAuthRoute =
				request.url.endsWith('/api/v1/auth/login') ||
				request.url.endsWith('/api/v1/auth/signup') ||
				request.url.endsWith('/api/v1/auth/refresh');
			if (!isPublicAuthRoute && currentAccessToken && !request.headers.has('Authorization')) {
				request.headers.set('Authorization', `Bearer ${currentAccessToken}`);
			}
			requestClones.set(request, request.clone() as unknown as Request);
			return request;
		},
		async onResponse({ request, response }) {
			const isPublicAuthRoute =
				request.url.endsWith('/api/v1/auth/login') ||
				request.url.endsWith('/api/v1/auth/signup') ||
				request.url.endsWith('/api/v1/auth/refresh');
			if (response.status !== 401 || isPublicAuthRoute) {
				return response;
			}

			const newToken = await refreshSession();
			if (!newToken) {
				clearSession();
				return response;
			}

			const retrySource = requestClones.get(request) ?? request;
			const retryHeaders = new Headers(retrySource.headers);
			retryHeaders.set('Authorization', `Bearer ${newToken}`);
			const retryRequest = new Request(retrySource, { headers: retryHeaders });
			const retryResponse = await event.fetch(retryRequest);
			if (retryResponse.status === 401) {
				clearSession();
			}
			return retryResponse;
		}
	});

	return createDomainServices(client);
}
