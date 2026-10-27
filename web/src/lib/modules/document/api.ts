import type { HttpClient } from '#lib/core/api/http-client';
import type {
	DocumentTemplateDTO,
	GenerateDocCmd,
	GLIDossierArchiveDTO,
	RegisterTemplateCmd,
	StoredFileDTO
} from './types';

export class DocumentApiService {
	constructor(private readonly http: HttpClient) {}

	listTemplates(): Promise<DocumentTemplateDTO[]> {
		return this.http.requestJson<DocumentTemplateDTO[]>('/api/v1/document-templates');
	}

	registerTemplate(cmd: RegisterTemplateCmd): Promise<DocumentTemplateDTO> {
		return this.http.requestJson<DocumentTemplateDTO>('/api/v1/document-templates', {
			method: 'POST',
			body: JSON.stringify(cmd)
		});
	}

	generateFromTemplate(cmd: GenerateDocCmd): Promise<StoredFileDTO> {
		return this.http.requestJson<StoredFileDTO>('/api/v1/documents/generate', {
			method: 'POST',
			body: JSON.stringify(cmd)
		});
	}

	uploadFile(params: {
		legalEntityId?: string;
		leaseId?: string;
		category: string;
		file: File;
	}): Promise<StoredFileDTO> {
		const formData = new FormData();
		if (params.legalEntityId) {
			formData.set('legal_entity_id', params.legalEntityId);
		}
		if (params.leaseId) {
			formData.set('lease_id', params.leaseId);
		}
		formData.set('category', params.category);
		formData.set('file', params.file);
		return this.http.requestMultipart<StoredFileDTO>('/api/v1/documents/upload', formData);
	}

	assembleGliDossier(leaseId: string, templateId?: string): Promise<GLIDossierArchiveDTO> {
		return this.http.requestJson<GLIDossierArchiveDTO>(
			`/api/v1/leases/${encodeURIComponent(leaseId)}/gli-dossier`,
			{
				method: 'POST',
				body: JSON.stringify({ template_id: templateId ?? '' })
			}
		);
	}

	listLeaseDocuments(leaseId: string): Promise<StoredFileDTO[]> {
		return this.http.requestJson<StoredFileDTO[]>(
			`/api/v1/leases/${encodeURIComponent(leaseId)}/documents`
		);
	}
}
