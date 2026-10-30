import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import type { TemplateFormat, TemplateType } from '#lib/modules/document/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
	try {
		const templates = await locals.api.document.listTemplates();
		return {
			templates,
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			templates: [],
			errorMessage:
				err instanceof ApiError ? err.message : 'Impossible de charger les modèles de documents.'
		};
	}
};

export const actions: Actions = {
	registerTemplate: async ({ request, locals }) => {
		const formData = await request.formData();
		const name = String(formData.get('name') ?? '').trim();
		const templateType = String(formData.get('template_type') ?? 'LEASE') as TemplateType;
		const format = String(formData.get('format') ?? 'HTML') as TemplateFormat;
		const htmlBody = String(formData.get('html_body') ?? '');
		const docxBase64 = String(formData.get('docx_base64') ?? '');

		if (!name) {
			return fail(400, { actionError: 'Veuillez saisir le nom du modèle.' });
		}
		if (format === 'HTML' && !htmlBody.trim()) {
			return fail(400, { actionError: 'Veuillez renseigner le gabarit HTML.' });
		}
		if (format === 'DOCX' && !docxBase64) {
			return fail(400, { actionError: 'Veuillez sélectionner un fichier .docx source.' });
		}

		try {
			const created = await locals.api.document.registerTemplate({
				name,
				template_type: templateType,
				format,
				html_body: format === 'HTML' ? htmlBody : undefined,
				docx_base64: format === 'DOCX' ? docxBase64 : undefined
			});
			return {
				registeredTemplate: created,
				successMessage: 'Modèle documentaire enregistré.'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Impossible d’enregistrer le modèle.'
			});
		}
	}
};
