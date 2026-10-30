import { describe, expect, it } from 'vitest';
import { buildCockpitEntityGroups } from './cockpit-builder';

const ENT_ID = '0192dce0-7c4a-7002-8000-000000000001';
const PROP_ID = '0192dce0-7c4a-7002-8000-000000000002';
const LOT_1_ID = '0192dce0-7c4a-7002-8000-000000000003';
const LOT_2_ID = '0192dce0-7c4a-7002-8000-000000000004';
const LEASE_1_ID = '0192dce0-7c4a-7002-8000-000000000005';
const TEN_1_ID = '0192dce0-7c4a-7002-8000-000000000006';
const RC_1_ID = '0192dce0-7c4a-7002-8000-000000000007';
const INC_1_ID = '0192dce0-7c4a-7002-8000-000000000008';
const INC_CLAIM_ID = '0192dce0-7c4a-7002-8000-000000000009';
const INC_WIP_ID = '0192dce0-7c4a-7002-8000-00000000000a';
const INC_CLOSED_ID = '0192dce0-7c4a-7002-8000-00000000000b';

describe('buildCockpitEntityGroups', () => {
	it('groups properties and lots by legal entity and derives prioritized CTAs per row', () => {
		const groups = buildCockpitEntityGroups({
			periodMonth: '2026-10',
			entities: [
				{
					id: ENT_ID,
					name: 'SCI Les Oliviers',
					entity_type: 'SCI_IS',
					siret: '12345678900010',
					naf_code: '6820B',
					address: '10 rue République, Lyon',
					share_capital_cents: 100000,
					total_shares: 100,
					created_at: '2026-01-01T00:00:00Z',
					updated_at: '2026-01-01T00:00:00Z'
				}
			],
			propertiesByEntity: {
				[ENT_ID]: [
					{
						id: PROP_ID,
						legal_entity_id: ENT_ID,
						label: 'Immeuble République',
						property_type: 'APARTMENT_BUILDING',
						accounting_category: 'FIXED_ASSET',
						address: '10 rue République, Lyon',
						acquisition_cents: 25000000,
						work_stock_cents: 0,
						total_book_value_cents: 25000000,
						lots: [
							{
								id: LOT_1_ID,
								property_id: PROP_ID,
								label: 'Appt T3 République',
								lot_type: 'APARTMENT',
								surface_m2: 65,
								ownership_share: 500,
								floor: '2',
								created_at: '2026-01-01T00:00:00Z',
								updated_at: '2026-01-01T00:00:00Z'
							},
							{
								id: LOT_2_ID,
								property_id: PROP_ID,
								label: 'Studio Vacant',
								lot_type: 'STUDIO',
								surface_m2: 24,
								ownership_share: 200,
								floor: '1',
								created_at: '2026-01-01T00:00:00Z',
								updated_at: '2026-01-01T00:00:00Z'
							}
						],
						created_at: '2026-01-01T00:00:00Z',
						updated_at: '2026-01-01T00:00:00Z'
					}
				]
			},
			leases: [
				{
					id: LEASE_1_ID,
					legal_entity_id: ENT_ID,
					lot_id: LOT_1_ID,
					tenant_id: TEN_1_ID,
					lease_type: 'RESIDENTIAL_UNFURNISHED',
					rent_net_cents: 85000,
					charges_cents: 7000,
					subject_to_vat: false,
					vat_rate_basis_points: 0,
					security_deposit_cents: 85000,
					due_day_of_month: 5,
					gli_policy_number: 'GLI-001',
					guarantor_name: '',
					start_date: '2025-01-01T00:00:00Z',
					active: true,
					created_at: '2025-01-01T00:00:00Z',
					updated_at: '2025-01-01T00:00:00Z'
				}
			],
			tenants: [
				{
					id: TEN_1_ID,
					full_name: 'Marc Martin',
					email: 'marc@example.com',
					phone: '0600000000',
					address: 'Lyon',
					archived: false,
					created_at: '2025-01-01T00:00:00Z',
					updated_at: '2025-01-01T00:00:00Z'
				}
			],
			rentCallsByEntity: {
				[ENT_ID]: [
					{
						id: RC_1_ID,
						lease_id: LEASE_1_ID,
						legal_entity_id: ENT_ID,
						tenant_id: TEN_1_ID,
						period_month: '2026-10',
						rent_net_cents: 85000,
						vat_cents: 0,
						charges_cents: 7000,
						total_due_cents: 92000,
						remaining_due_cents: 92000,
						status: 'PENDING',
						due_date: '2026-10-05T00:00:00Z',
						created_at: '2026-10-01T00:00:00Z',
						updated_at: '2026-10-01T00:00:00Z'
					}
				]
			},
			negativePositionsByEntity: {
				[ENT_ID]: [
					{
						lease_id: LEASE_1_ID,
						legal_entity_id: ENT_ID,
						lot_id: LOT_1_ID,
						tenant_id: TEN_1_ID,
						tenant_name: 'Marc Martin',
						total_unpaid_cents: 92000,
						unpaid_months: ['2026-10'],
						oldest_due_date: '2026-10-05T00:00:00Z',
						gli_policy_number: 'GLI-001',
						gli_eligible: true
					}
				]
			},
			incidents: [
				{
					id: INC_1_ID,
					property_id: PROP_ID,
					lot_id: LOT_1_ID,
					lease_id: LEASE_1_ID,
					title: 'Fuite ballon ECS',
					status: 'OPEN',
					occurred_at: '2026-10-10',
					estimated_cost_cents: 35000,
					observations: [],
					photo_r2_keys: [],
					created_at: '2026-10-10T00:00:00Z',
					updated_at: '2026-10-10T00:00:00Z'
				}
			]
		});

		expect(groups).toHaveLength(1);
		expect(groups[0].expectedRentCents).toBe(92000);
		expect(groups[0].collectedRentCents).toBe(0);
		expect(groups[0].rows).toHaveLength(2);

		const leasedRow = groups[0].rows[0];
		expect(leasedRow.actions.map((a) => a.kind)).toEqual(['remind', 'pay', 'incident']);
		expect(leasedRow.actions[0].href).toContain(`/rents?entityId=${ENT_ID}&leaseId=${LEASE_1_ID}`);

		const vacantRow = groups[0].rows[1];
		expect(vacantRow.isVacant).toBe(true);
		expect(vacantRow.actions[0].kind).toBe('create-lease');
	});

	it('marks a paid lease as isUpToDate even when a zero-balance negativePosition record exists', () => {
		const groups = buildCockpitEntityGroups({
			periodMonth: '2026-10',
			entities: [
				{
					id: ENT_ID,
					name: 'SCI Les Oliviers',
					entity_type: 'SCI_IS',
					siret: '12345678900010',
					naf_code: '6820B',
					address: '10 rue République, Lyon',
					share_capital_cents: 100000,
					total_shares: 100,
					created_at: '2026-01-01T00:00:00Z',
					updated_at: '2026-01-01T00:00:00Z'
				}
			],
			propertiesByEntity: {
				[ENT_ID]: [
					{
						id: PROP_ID,
						legal_entity_id: ENT_ID,
						label: 'Immeuble République',
						property_type: 'APARTMENT_BUILDING',
						accounting_category: 'FIXED_ASSET',
						address: '10 rue République, Lyon',
						acquisition_cents: 25000000,
						work_stock_cents: 0,
						total_book_value_cents: 25000000,
						lots: [
							{
								id: LOT_1_ID,
								property_id: PROP_ID,
								label: 'Appt T3 République',
								lot_type: 'APARTMENT',
								surface_m2: 65,
								ownership_share: 500,
								floor: '2',
								created_at: '2026-01-01T00:00:00Z',
								updated_at: '2026-01-01T00:00:00Z'
							}
						],
						created_at: '2026-01-01T00:00:00Z',
						updated_at: '2026-01-01T00:00:00Z'
					}
				]
			},
			leases: [
				{
					id: LEASE_1_ID,
					legal_entity_id: ENT_ID,
					lot_id: LOT_1_ID,
					tenant_id: TEN_1_ID,
					lease_type: 'RESIDENTIAL_UNFURNISHED',
					rent_net_cents: 85000,
					charges_cents: 7000,
					subject_to_vat: false,
					vat_rate_basis_points: 0,
					security_deposit_cents: 85000,
					due_day_of_month: 5,
					gli_policy_number: 'GLI-001',
					guarantor_name: '',
					start_date: '2025-01-01T00:00:00Z',
					active: true,
					created_at: '2025-01-01T00:00:00Z',
					updated_at: '2025-01-01T00:00:00Z'
				}
			],
			tenants: [
				{
					id: TEN_1_ID,
					full_name: 'Marc Martin',
					email: 'marc@example.com',
					phone: '0600000000',
					address: 'Lyon',
					archived: false,
					created_at: '2025-01-01T00:00:00Z',
					updated_at: '2025-01-01T00:00:00Z'
				}
			],
			rentCallsByEntity: {
				[ENT_ID]: [
					{
						id: RC_1_ID,
						lease_id: LEASE_1_ID,
						legal_entity_id: ENT_ID,
						tenant_id: TEN_1_ID,
						period_month: '2026-10',
						rent_net_cents: 85000,
						vat_cents: 0,
						charges_cents: 7000,
						total_due_cents: 92000,
						remaining_due_cents: 0,
						status: 'PAID',
						due_date: '2026-10-05T00:00:00Z',
						created_at: '2026-10-01T00:00:00Z',
						updated_at: '2026-10-01T00:00:00Z'
					}
				]
			},
			negativePositionsByEntity: {
				[ENT_ID]: [
					{
						lease_id: LEASE_1_ID,
						legal_entity_id: ENT_ID,
						lot_id: LOT_1_ID,
						tenant_id: TEN_1_ID,
						tenant_name: 'Marc Martin',
						total_unpaid_cents: 0,
						unpaid_months: [],
						oldest_due_date: '2026-10-05T00:00:00Z',
						gli_policy_number: 'GLI-001',
						gli_eligible: false
					}
				]
			},
			incidents: []
		});

		expect(groups[0].rows[0].isUpToDate).toBe(true);
		expect(groups[0].rows[0].actions).toEqual([]);
	});

	it('counts INSURANCE_CLAIM and WORK_IN_PROGRESS incidents as open while excluding CLOSED and links to /inspections/incidents without tab query params', () => {
		const groups = buildCockpitEntityGroups({
			periodMonth: '2026-10',
			entities: [
				{
					id: ENT_ID,
					name: 'SCI Les Oliviers',
					entity_type: 'SCI_IS',
					siret: '12345678900010',
					naf_code: '6820B',
					address: '10 rue République, Lyon',
					share_capital_cents: 100000,
					total_shares: 100,
					created_at: '2026-01-01T00:00:00Z',
					updated_at: '2026-01-01T00:00:00Z'
				}
			],
			propertiesByEntity: {
				[ENT_ID]: [
					{
						id: PROP_ID,
						legal_entity_id: ENT_ID,
						label: 'Immeuble République',
						property_type: 'BUILDING',
						accounting_category: 'FIXED_ASSET',
						address: '10 rue République, Lyon',
						acquisition_cents: 25000000,
						work_stock_cents: 0,
						total_book_value_cents: 25000000,
						lots: [
							{
								id: LOT_1_ID,
								property_id: PROP_ID,
								label: 'Appt T3 République',
								lot_type: 'APARTMENT',
								surface_m2: 65,
								ownership_share: 500,
								floor: '2',
								created_at: '2026-01-01T00:00:00Z',
								updated_at: '2026-01-01T00:00:00Z'
							}
						],
						created_at: '2026-01-01T00:00:00Z',
						updated_at: '2026-01-01T00:00:00Z'
					}
				]
			},
			leases: [],
			tenants: [],
			rentCallsByEntity: {},
			negativePositionsByEntity: {},
			incidents: [
				{
					id: INC_CLAIM_ID,
					property_id: PROP_ID,
					lot_id: LOT_1_ID,
					title: 'Dégât des eaux toiture',
					status: 'INSURANCE_CLAIM',
					occurred_at: '2026-10-12',
					estimated_cost_cents: 120000,
					observations: [],
					photo_r2_keys: [],
					created_at: '2026-10-12T00:00:00Z',
					updated_at: '2026-10-12T00:00:00Z'
				},
				{
					id: INC_WIP_ID,
					property_id: PROP_ID,
					lot_id: LOT_1_ID,
					title: 'Réparation plomberie',
					status: 'WORK_IN_PROGRESS',
					occurred_at: '2026-10-14',
					estimated_cost_cents: 45000,
					observations: [],
					photo_r2_keys: [],
					created_at: '2026-10-14T00:00:00Z',
					updated_at: '2026-10-14T00:00:00Z'
				},
				{
					id: INC_CLOSED_ID,
					property_id: PROP_ID,
					lot_id: LOT_1_ID,
					title: 'Ampoule hall remplacée',
					status: 'CLOSED',
					occurred_at: '2026-10-01',
					estimated_cost_cents: 2000,
					observations: [],
					photo_r2_keys: [],
					created_at: '2026-10-01T00:00:00Z',
					updated_at: '2026-10-02T00:00:00Z'
				}
			]
		});

		const row = groups[0].rows[0];
		expect(row.openIncidentsCount).toBe(2);
		const incidentAction = row.actions.find((action) => action.kind === 'incident');
		expect(incidentAction).toBeDefined();
		expect(incidentAction?.href).toBe(
			`/inspections/incidents?entityId=${ENT_ID}&propertyId=${PROP_ID}&incidentId=${INC_CLAIM_ID}`
		);
		expect(incidentAction?.href).not.toContain('tab=');
	});
});
