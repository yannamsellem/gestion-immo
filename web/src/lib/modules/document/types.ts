import type { components } from '#lib/core/api/openapi';

export type TemplateType =
	'LEASE' | 'RENT_RECEIPT' | 'FORMAL_NOTICE' | 'GLI_DECLARATION' | 'CUSTOM' | string;
export type TemplateFormat = 'DOCX' | 'HTML' | string;

export type RegisterTemplateCmd = components['schemas']['RegisterTemplateBody'];
export type DocumentTemplateDTO = components['schemas']['DocumentTemplateDTO'];
export type GenerateDocCmd = components['schemas']['GenerateDocCmd'];
export type StoredFileDTO = components['schemas']['StoredFileDTO'];
export type GLIDossierArchiveDTO = components['schemas']['GLIDossierArchiveDTO'];
