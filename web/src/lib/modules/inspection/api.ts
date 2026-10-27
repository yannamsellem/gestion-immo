import type { HttpClient } from '#lib/core/api/http-client';
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
} from './types';

export class InspectionApiService {
	constructor(private readonly http: HttpClient) {}

	listTemplates(): Promise<InspectionTemplateDTO[]> {
		return this.http.requestJson<InspectionTemplateDTO[]>('/api/v1/inspection-templates');
	}

	createTemplate(cmd: CreateInspectionTemplateCmd): Promise<InspectionTemplateDTO> {
		return this.http.requestJson<InspectionTemplateDTO>('/api/v1/inspection-templates', {
			method: 'POST',
			body: JSON.stringify(cmd)
		});
	}

	getTemplate(templateId: string): Promise<InspectionTemplateDTO> {
		return this.http.requestJson<InspectionTemplateDTO>(
			`/api/v1/inspection-templates/${encodeURIComponent(templateId)}`
		);
	}

	createInspection(cmd: CreateInspectionCmd): Promise<InspectionDTO> {
		return this.http.requestJson<InspectionDTO>('/api/v1/inspections', {
			method: 'POST',
			body: JSON.stringify(cmd)
		});
	}

	getInspection(inspectionId: string): Promise<InspectionDTO> {
		return this.http.requestJson<InspectionDTO>(
			`/api/v1/inspections/${encodeURIComponent(inspectionId)}`
		);
	}

	upsertInspectionItems(
		inspectionId: string,
		items: InspectionItemInput[]
	): Promise<InspectionDTO> {
		return this.http.requestJson<InspectionDTO>(
			`/api/v1/inspections/${encodeURIComponent(inspectionId)}/items`,
			{
				method: 'PUT',
				body: JSON.stringify({ items })
			}
		);
	}

	generateInspectionPdf(inspectionId: string): Promise<InspectionReportDTO> {
		return this.http.requestJson<InspectionReportDTO>(
			`/api/v1/inspections/${encodeURIComponent(inspectionId)}/pdf`,
			{
				method: 'POST'
			}
		);
	}

	listIncidents(filters?: {
		propertyId?: string;
		lotId?: string;
		leaseId?: string;
		status?: string;
	}): Promise<IncidentDTO[]> {
		const params = new URLSearchParams();
		if (filters?.propertyId) params.set('property_id', filters.propertyId);
		if (filters?.lotId) params.set('lot_id', filters.lotId);
		if (filters?.leaseId) params.set('lease_id', filters.leaseId);
		if (filters?.status) params.set('status', filters.status);
		const query = params.toString();
		return this.http.requestJson<IncidentDTO[]>(`/api/v1/incidents${query ? `?${query}` : ''}`);
	}

	createIncident(cmd: CreateIncidentCmd): Promise<IncidentDTO> {
		return this.http.requestJson<IncidentDTO>('/api/v1/incidents', {
			method: 'POST',
			body: JSON.stringify(cmd)
		});
	}

	getIncident(incidentId: string): Promise<IncidentDTO> {
		return this.http.requestJson<IncidentDTO>(
			`/api/v1/incidents/${encodeURIComponent(incidentId)}`
		);
	}

	updateIncident(incidentId: string, cmd: UpdateIncidentCmd): Promise<IncidentDTO> {
		return this.http.requestJson<IncidentDTO>(
			`/api/v1/incidents/${encodeURIComponent(incidentId)}`,
			{
				method: 'PATCH',
				body: JSON.stringify(cmd)
			}
		);
	}
}
