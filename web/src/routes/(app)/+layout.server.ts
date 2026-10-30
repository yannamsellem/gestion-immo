import { redirect } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import { clearSessionCookies } from '#lib/server/cookies';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ locals, cookies }) => {
	if (!locals.user) {
		throw redirect(303, '/login');
	}

	try {
		const entities = await locals.api.portfolio.listEntities();

		const validEntityId =
			locals.selectedEntityId !== 'ALL' &&
			entities.length > 0 &&
			!entities.some((entity) => entity.id === locals.selectedEntityId)
				? 'ALL'
				: locals.selectedEntityId;

		locals.selectedEntityId = validEntityId;

		return {
			user: locals.user,
			entities,
			entitiesLoadError: null as string | null,
			selectedEntityId: validEntityId,
			selectedPeriodMonth: locals.selectedPeriodMonth
		};
	} catch (err) {
		if (err instanceof ApiError && err.status === 401) {
			clearSessionCookies(cookies);
			locals.user = null;
			throw redirect(303, '/login');
		}
		return {
			user: locals.user,
			entities: [],
			entitiesLoadError:
				err instanceof ApiError
					? err.message
					: 'Impossible de charger la liste des entités juridiques.',
			selectedEntityId: locals.selectedEntityId,
			selectedPeriodMonth: locals.selectedPeriodMonth
		};
	}
};
