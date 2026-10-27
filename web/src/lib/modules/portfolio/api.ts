import type { HttpClient } from '#lib/core/api/http-client';
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
} from './types';

export class PortfolioApiService {
	constructor(private readonly http: HttpClient) {}

	listEntities(): Promise<LegalEntityDTO[]> {
		return this.http.requestJson<LegalEntityDTO[]>('/api/v1/entities');
	}

	createEntity(cmd: CreateLegalEntityCmd): Promise<LegalEntityDTO> {
		return this.http.requestJson<LegalEntityDTO>('/api/v1/entities', {
			method: 'POST',
			body: JSON.stringify(cmd)
		});
	}

	getEntitySummary(entityId: string): Promise<EntityPortfolioSummaryDTO> {
		return this.http.requestJson<EntityPortfolioSummaryDTO>(
			`/api/v1/entities/${encodeURIComponent(entityId)}`
		);
	}

	upsertShareholder(entityId: string, cmd: UpsertShareholderCmd): Promise<ShareholderDTO> {
		return this.http.requestJson<ShareholderDTO>(
			`/api/v1/entities/${encodeURIComponent(entityId)}/shareholders`,
			{
				method: 'POST',
				body: JSON.stringify({ ...cmd, legal_entity_id: entityId })
			}
		);
	}

	listProperties(entityId: string): Promise<PropertyDTO[]> {
		return this.http.requestJson<PropertyDTO[]>(
			`/api/v1/entities/${encodeURIComponent(entityId)}/properties`
		);
	}

	createProperty(entityId: string, cmd: CreatePropertyCmd): Promise<PropertyDTO> {
		return this.http.requestJson<PropertyDTO>(
			`/api/v1/entities/${encodeURIComponent(entityId)}/properties`,
			{
				method: 'POST',
				body: JSON.stringify({ ...cmd, legal_entity_id: entityId })
			}
		);
	}

	createLot(propertyId: string, cmd: CreateLotCmd): Promise<LotDTO> {
		return this.http.requestJson<LotDTO>(
			`/api/v1/properties/${encodeURIComponent(propertyId)}/lots`,
			{
				method: 'POST',
				body: JSON.stringify({ ...cmd, property_id: propertyId })
			}
		);
	}
}
