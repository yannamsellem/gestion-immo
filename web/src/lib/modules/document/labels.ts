import type { TemplateFormat, TemplateType } from './types';

export const TEMPLATE_TYPE_OPTIONS: ReadonlyArray<{ value: TemplateType; label: string }> = [
	{ value: 'LEASE', label: 'Contrat de bail' },
	{ value: 'RENT_RECEIPT', label: 'Quittance / Avis d’échéance' },
	{ value: 'FORMAL_NOTICE', label: 'Lettre de relance / Mise en demeure' },
	{ value: 'GLI_DECLARATION', label: 'Déclaration sinistre GLI' },
	{ value: 'CUSTOM', label: 'Modèle personnalisé' }
];

const TEMPLATE_TYPE_LABELS = new Map(TEMPLATE_TYPE_OPTIONS.map((opt) => [opt.value, opt.label]));

export function formatTemplateTypeLabel(value: string): string {
	return TEMPLATE_TYPE_LABELS.get(value) ?? value;
}

export const TEMPLATE_FORMAT_OPTIONS: ReadonlyArray<{ value: TemplateFormat; label: string }> = [
	{ value: 'HTML', label: 'HTML / PDF (Chromedp)' },
	{ value: 'DOCX', label: 'Word .docx (Mustache)' }
];
