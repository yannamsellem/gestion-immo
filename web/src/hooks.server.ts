import type { RequestEvent } from '@sveltejs/kit';
import { createBackendApi } from '#lib/server/backend-api';
import {
	getSessionFromCookies,
	getWorkspaceFromCookies,
	setWorkspaceCookies
} from '#lib/server/cookies';

export async function handle({
	event,
	resolve
}: {
	event: RequestEvent;
	resolve: (event: RequestEvent) => Promise<Response>;
}): Promise<Response> {
	const isSecure = event.url.protocol === 'https:';
	const session = getSessionFromCookies(event.cookies);
	const workspace = getWorkspaceFromCookies(event.cookies);

	const queryEntityId = event.url.searchParams.get('entityId');
	const queryPeriodMonth = event.url.searchParams.get('periodMonth');

	const selectedEntityId =
		queryEntityId && queryEntityId.trim() !== ''
			? queryEntityId.trim()
			: workspace.selectedEntityId;
	const selectedPeriodMonth =
		queryPeriodMonth && /^\d{4}-\d{2}$/.test(queryPeriodMonth)
			? queryPeriodMonth
			: workspace.selectedPeriodMonth;

	if (queryEntityId || queryPeriodMonth) {
		setWorkspaceCookies(event.cookies, { selectedEntityId, selectedPeriodMonth }, isSecure);
	}

	event.locals.user = session.user;
	event.locals.selectedEntityId = selectedEntityId;
	event.locals.selectedPeriodMonth = selectedPeriodMonth;
	event.locals.api = createBackendApi(event);

	return resolve(event);
}
