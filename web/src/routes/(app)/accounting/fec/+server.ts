import { error, redirect } from '@sveltejs/kit';
import { ApiError } from '#lib/core/api/api-error';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ url, locals }) => {
	if (!locals.user) {
		throw redirect(303, '/login');
	}

	const entityId = (url.searchParams.get('entityId') ?? '').trim();
	const fiscalYear =
		Number.parseInt(url.searchParams.get('fiscalYear') ?? '', 10) || new Date().getFullYear();
	const siren = (url.searchParams.get('siren') ?? '000000000').slice(0, 9) || '000000000';

	if (!entityId) {
		throw error(400, 'Paramètre entityId manquant.');
	}

	try {
		const tsvContent = await locals.api.accounting.exportFecTsv(entityId, fiscalYear);
		return new Response(tsvContent, {
			headers: {
				'Content-Type': 'text/tab-separated-values; charset=utf-8',
				'Content-Disposition': `attachment; filename="${siren}FEC${fiscalYear}1231.tsv"`
			}
		});
	} catch (err) {
		throw error(
			err instanceof ApiError ? err.status : 500,
			err instanceof ApiError ? err.message : 'Échec de l’export FEC.'
		);
	}
};
