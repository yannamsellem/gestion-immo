import { describe, expect, it } from 'vitest';
import { buildUnifiedRentRows, summarizeRentRows } from './rent-rows';

describe('buildUnifiedRentRows & summarizeRentRows', () => {
	it('computes integer VAT/gross rent, categorizes PAID/OVERDUE/TO_COLLECT, and summarizes totals', () => {
		const rows = buildUnifiedRentRows({
			entities: [
				{
					id: 'ent_1',
					name: 'SCI Rivoli',
					entity_type: 'SCI_IS',
					siret: '12345678900011',
					naf_code: '6820B',
					address: '12 rue de Rivoli',
					share_capital_cents: 100000,
					total_shares: 100,
					created_at: '2026-01-01T00:00:00Z',
					updated_at: '2026-01-01T00:00:00Z'
				}
			],
			leases: [
				{
					id: 'lease_1',
					legal_entity_id: 'ent_1',
					lot_id: 'lot_1',
					tenant_id: 'ten_1',
					lease_type: 'COMMERCIAL_3_6_9',
					rent_net_cents: 100000,
					charges_cents: 15000,
					subject_to_vat: true,
					vat_rate_basis_points: 2000,
					security_deposit_cents: 200000,
					due_day_of_month: 5,
					gli_policy_number: 'GLI-01',
					guarantor_name: '',
					start_date: '2026-01-01T00:00:00Z',
					active: true,
					created_at: '2026-01-01T00:00:00Z',
					updated_at: '2026-01-01T00:00:00Z'
				},
				{
					id: 'lease_2',
					legal_entity_id: 'ent_1',
					lot_id: 'lot_2',
					tenant_id: 'ten_2',
					lease_type: 'RESIDENTIAL_EMPTY',
					rent_net_cents: 80000,
					charges_cents: 5000,
					subject_to_vat: false,
					vat_rate_basis_points: 0,
					security_deposit_cents: 80000,
					due_day_of_month: 5,
					gli_policy_number: '',
					guarantor_name: '',
					start_date: '2026-01-01T00:00:00Z',
					active: true,
					created_at: '2026-01-01T00:00:00Z',
					updated_at: '2026-01-01T00:00:00Z'
				}
			],
			tenants: [
				{
					id: 'ten_1',
					full_name: 'Boutique SAS',
					email: 'contact@boutique.fr',
					phone: '0102030405',
					address: 'Paris',
					archived: false,
					created_at: '2026-01-01T00:00:00Z',
					updated_at: '2026-01-01T00:00:00Z'
				}
			],
			propertiesByEntity: {
				ent_1: [
					{
						id: 'prop_1',
						legal_entity_id: 'ent_1',
						label: 'Immeuble Rivoli',
						property_type: 'BUILDING',
						accounting_category: 'FIXED_ASSET',
						address: '12 rue de Rivoli',
						acquisition_cents: 50000000,
						work_stock_cents: 0,
						total_book_value_cents: 50000000,
						lots: [
							{
								id: 'lot_1',
								property_id: 'prop_1',
								label: 'RDC Commercial',
								lot_type: 'COMMERCIAL',
								surface_m2: 65,
								ownership_share: 250,
								floor: '0',
								created_at: '2026-01-01T00:00:00Z',
								updated_at: '2026-01-01T00:00:00Z'
							}
						],
						created_at: '2026-01-01T00:00:00Z',
						updated_at: '2026-01-01T00:00:00Z'
					}
				]
			},
			rentCallsByEntity: {
				ent_1: [
					{
						id: 'rc_2',
						lease_id: 'lease_2',
						legal_entity_id: 'ent_1',
						tenant_id: 'ten_2',
						period_month: '2026-10',
						due_date: '2026-10-05T00:00:00Z',
						rent_net_cents: 80000,
						vat_cents: 0,
						charges_cents: 5000,
						total_due_cents: 85000,
						remaining_due_cents: 0,
						status: 'PAID',
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
						tenant_name: 'Boutique SAS',
						total_unpaid_cents: 135000,
						unpaid_months: ['2026-10'],
						oldest_due_date: '2026-10-05T00:00:00Z',
						gli_policy_number: 'GLI-01',
						gli_eligible: true
					}
				]
			},
			selectedEntityId: 'ALL',
			selectedPeriodMonth: '2026-10'
		});

		expect(rows).toHaveLength(2);
		expect(rows[0].expectedGrossCents).toBe(135000);
		expect(rows[0].statusCategory).toBe('OVERDUE');
		expect(rows[0].propertyAndLotLabel).toBe('Immeuble Rivoli — RDC Commercial');
		expect(rows[1].statusCategory).toBe('PAID');
		expect(rows[1].remainingDueCents).toBe(0);

		expect(summarizeRentRows(rows)).toEqual({
			expectedTotalCents: 220000,
			collectedTotalCents: 85000,
			remainingTotalCents: 135000,
			overdueCount: 1
		});
	});
});
