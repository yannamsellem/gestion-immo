import type { HttpClient } from '#lib/core/api/http-client';
import type { CreateLeaseCmd, CreateTenantCmd, LeaseDTO, RentCallDTO, TenantDTO } from './types';

export class LeaseApiService {
	constructor(private readonly http: HttpClient) {}

	listTenants(includeArchived = false): Promise<TenantDTO[]> {
		const query = includeArchived ? '?include_archived=true' : '';
		return this.http.requestJson<TenantDTO[]>(`/api/v1/tenants${query}`);
	}

	createTenant(cmd: CreateTenantCmd): Promise<TenantDTO> {
		return this.http.requestJson<TenantDTO>('/api/v1/tenants', {
			method: 'POST',
			body: JSON.stringify(cmd)
		});
	}

	archiveTenant(tenantId: string): Promise<TenantDTO> {
		return this.http.requestJson<TenantDTO>(
			`/api/v1/tenants/${encodeURIComponent(tenantId)}/archive`,
			{
				method: 'POST'
			}
		);
	}

	listLeases(entityId?: string): Promise<LeaseDTO[]> {
		const query = entityId && entityId !== 'ALL' ? `?entity_id=${encodeURIComponent(entityId)}` : '';
		return this.http.requestJson<LeaseDTO[]>(`/api/v1/leases${query}`);
	}

	createLease(cmd: CreateLeaseCmd): Promise<LeaseDTO> {
		return this.http.requestJson<LeaseDTO>('/api/v1/leases', {
			method: 'POST',
			body: JSON.stringify(cmd)
		});
	}

	generateRentCalls(entityId: string, periodMonth: string): Promise<RentCallDTO[]> {
		return this.http.requestJson<RentCallDTO[]>(
			`/api/v1/entities/${encodeURIComponent(entityId)}/rent-calls/generate`,
			{
				method: 'POST',
				body: JSON.stringify({ period_month: periodMonth })
			}
		);
	}

	listEntityRentCalls(entityId: string, periodMonth?: string): Promise<RentCallDTO[]> {
		const query = periodMonth ? `?period_month=${encodeURIComponent(periodMonth)}` : '';
		return this.http.requestJson<RentCallDTO[]>(
			`/api/v1/entities/${encodeURIComponent(entityId)}/rent-calls${query}`
		);
	}
}
