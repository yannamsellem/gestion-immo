import { fail } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import type { InspectionTemplateDTO } from '#lib/modules/inspection/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
	try {
		const templates = await locals.api.inspection.listTemplates();
		return {
			templates,
			errorMessage: null as string | null
		};
	} catch (err) {
		return {
			templates: [] as InspectionTemplateDTO[],
			errorMessage:
				err instanceof ApiError
					? err.message
					: 'Impossible de charger les modèles d’états des lieux.'
		};
	}
};

export const actions: Actions = {
	createTemplate: async ({ request, locals }) => {
		const formData = await request.formData();
		const name = String(formData.get('name') ?? '').trim();
		const roomsText = String(formData.get('rooms_text') ?? '');
		const elementsText = String(formData.get('elements_text') ?? '');

		const rooms = roomsText
			.split(',')
			.map((segment) => segment.trim())
			.filter(Boolean);
		const defaultElements = elementsText
			.split(',')
			.map((segment) => segment.trim())
			.filter(Boolean);

		if (!name || rooms.length === 0) {
			return fail(400, {
				actionError: 'Veuillez renseigner un nom et au moins une pièce.'
			});
		}

		try {
			const created = await locals.api.inspection.createTemplate({
				name,
				rooms,
				default_elements: defaultElements
			});
			return {
				createdTemplate: created,
				successMessage: 'Modèle d’état des lieux créé.'
			};
		} catch (err) {
			return fail(err instanceof ApiError ? err.status : 500, {
				actionError: err instanceof ApiError ? err.message : 'Erreur lors de la création du modèle.'
			});
		}
	}
};
