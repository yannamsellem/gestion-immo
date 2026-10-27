import { describe, expect, it } from 'vitest';
import { buildCockpitEntityGroups } from './cockpit-builder';

describe('buildCockpitEntityGroups', () => {
	it('groups properties and lots by legal entity and derives prioritized CTAs per row', () => {
		const groups = buildCockpitEntityGroups({
			periodMonth: '2026-10',
			entities: [
				{
					id: 'ent_1',
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
				ent_1: [
					{
						id: 'prop_1',
						legal_entity_id: 'ent_1',
						label: 'Immeuble République',
						property_type: 'APARTMENT_BUILDING',
						accounting_category: 'FIXED_ASSET',
						address: '10 rue République, Lyon',
						acquisition_cents: 25000000,
						work_stock_cents: 0,
						total_book_value_cents: 25000000,
						lots: [
							{
								id: 'lot_1',
								property_id: 'prop_1',
								label: 'Appt T3 République',
								lot_type: 'APARTMENT',
								surface_m2: 65,
								ownership_share: 500,
								floor: '2',
								created_at: '2026-01-01T00:00:00Z',
								updated_at: '2026-01-01T00:00:00Z'
							},
							{
								id: 'lot_2',
								property_id: 'prop_1',
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
					id: 'lease_1',
					legal_entity_id: 'ent_1',
					lot_id: 'lot_1',
					tenant_id: 'ten_1',
					lease_type: 'RESIDENTIAL_EMPTY',
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
					id: 'ten_1',
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
				ent_1: [
					{
						id: 'rc_1',
						lease_id: 'lease_1',
						legal_entity_id: 'ent_1',
						tenant_id: 'ten_1',
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
				ent_1: [
					{
						lease_id: 'lease_1',
						legal_entity_id: 'ent_1',
						lot_id: 'lot_1',
						tenant_id: 'ten_1',
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
					id: 'inc_1',
					property_id: 'prop_1',
					lot_id: 'lot_1',
					lease_id: 'lease_1',
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
		expect(leasedRow.actions[0].href).toContain('/rents?entityId=ent_1&leaseId=lease_1');

		const vacantRow = groups[0].rows[1];
		expect(vacantRow.isVacant).toBe(true);
		expect(vacantRow.actions[0].kind).toBe('create-lease');
	});

	it('marks a paid lease as isUpToDate even when a zero-balance negativePosition record exists', () => {
		const groups = buildCockpitEntityGroups({
			periodMonth: '2026-10',
			entities: [
				{
					id: 'ent_1',
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
				ent_1: [
					{
						id: 'prop_1',
						legal_entity_id: 'ent_1',
						label: 'Immeuble République',
						property_type: 'APARTMENT_BUILDING',
						accounting_category: 'FIXED_ASSET',
						address: '10 rue République, Lyon',
						acquisition_cents: 25000000,
						work_stock_cents: 0,
						total_book_value_cents: 25000000,
						lots: [
							{
								id: 'lot_1',
								property_id: 'prop_1',
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
					id: 'lease_1',
					legal_entity_id: 'ent_1',
					lot_id: 'lot_1',
					tenant_id: 'ten_1',
					lease_type: 'RESIDENTIAL_EMPTY',
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
					id: 'ten_1',
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
				ent_1: [
					{
						id: 'rc_1',
						lease_id: 'lease_1',
						legal_entity_id: 'ent_1',
						tenant_id: 'ten_1',
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
				ent_1: [
					{
						lease_id: 'lease_1',
						legal_entity_id: 'ent_1',
						lot_id: 'lot_1',
						tenant_id: 'ten_1',
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
});
