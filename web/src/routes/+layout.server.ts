import { createApiClient, type LegalEntityDTO, type NegativePositionDTO, type UserDTO } from '$lib/api/client';

const FALLBACK_ENTITIES: readonly LegalEntityDTO[] = [
	{
		id: 'ent_sarl_investissimo',
		name: 'SARL INVESTISSIMO',
		entity_type: 'SARL',
		siret: '51946587600028',
		naf_code: '6820B',
		address: '30 Rue de Miromesnil, 75008 Paris',
		share_capital_cents: 800000,
		total_shares: 800,
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:00Z'
	},
	{
		id: 'ent_sci_yossi',
		name: 'SCI YOSSI',
		entity_type: 'SCI',
		siret: '81234567800019',
		naf_code: '6820A',
		address: '30 Rue de Miromesnil, 75008 Paris',
		share_capital_cents: 100000,
		total_shares: 100,
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:00Z'
	},
	{
		id: 'ent_sasu_yann_invest',
		name: 'SASU YANN INVEST',
		entity_type: 'SAS',
		siret: '89234567800011',
		naf_code: '6810Z',
		address: '30 Rue de Miromesnil, 75008 Paris',
		share_capital_cents: 500000,
		total_shares: 500,
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:00Z'
	},
	{
		id: 'ent_eurl_penhars_invest',
		name: 'EURL PENHARS INVEST',
		entity_type: 'EURL',
		siret: '84567890100014',
		naf_code: '6810Z',
		address: '12 Boulevard de Penhars, 29000 Quimper',
		share_capital_cents: 200000,
		total_shares: 200,
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:00Z'
	}
];

export async function load({
	fetch,
	cookies,
	platform
}: {
	fetch: typeof globalThis.fetch;
	cookies: { get(name: string): string | undefined };
	platform?: App.Platform;
}) {
	const accessToken = cookies.get('gi_access_token');
	const baseUrl = platform?.env?.PUBLIC_API_BASE_URL ?? process.env.PUBLIC_API_BASE_URL ?? 'http://localhost:8080';
	const api = createApiClient({
		baseUrl,
		accessToken,
		fetchFn: fetch
	});

	let entities: readonly LegalEntityDTO[] = FALLBACK_ENTITIES;
	let negativePositions: readonly NegativePositionDTO[] = [];
	const currentUser: UserDTO = {
		id: 'usr_super_admin',
		full_name: 'Yann Amsellem',
		email: 'yann@investissimo.fr',
		role: 'SUPER_ADMIN',
		status: 'APPROVED',
		created_at: '2026-01-01T00:00:00Z'
	};

	try {
		const remoteEntities = await api.listEntities();
		if (remoteEntities.length > 0) {
			entities = remoteEntities;
		}
		negativePositions = await api.listNegativePositions(entities[0].id);
	} catch {
		// Use deterministic SSR fallback portfolio when backend is unreachable
	}

	return {
		user: currentUser,
		entities,
		activeEntityId: entities[0]?.id ?? 'ent_sarl_investissimo',
		negativePositionsCount: negativePositions.length,
		apiBaseUrl: baseUrl
	};
}
