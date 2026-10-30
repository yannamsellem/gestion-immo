import { describe, expect, it } from 'vitest';
import {
	buildUnifiedRentRows,
	computeExpectedLeaseGrossCents,
	summarizeRentRows
} from './rent-rows';

describe('buildUnifiedRentRows & summarizeRentRows', () => {
	it('includes commercial VAT in computeExpectedLeaseGrossCents when rentCall is absent', () => {
		expect(
			computeExpectedLeaseGrossCents({
				rent_net_cents: 100000,
				subject_to_vat: true,
				vat_rate_basis_points: 2000,
				charges_cents: 15000
			})
		).toBe(135000);
		expect(
			computeExpectedLeaseGrossCents({
				rent_net_cents: 80000,
				subject_to_vat: false,
				vat_rate_basis_points: 2000,
				charges_cents: 5000
			})
		).toBe(85000);
	});

	it('computes integer VAT/gross rent, categorizes PAID/OVERDUE/TO_COLLECT, and summarizes totals', () => {
		const entityId = '0192dce0-7c4a-7001-8000-000000000001';
		const propId = '0192dce0-7c4a-7001-8000-000000000002';
		const lot1Id = '0192dce0-7c4a-7001-8000-111122223333';
		const lot2Id = '0192dce0-7c4a-7001-8000-9999a2b3c4d5';
		const lease1Id = '0192dce0-7c4a-7001-8000-000000000011';
		const lease2Id = '0192dce0-7c4a-7001-8000-000000000012';
		const tenant1Id = '0192dce0-7c4a-7001-8000-000000000021';
		const tenant2Id = '0192dce0-7c4a-7001-8000-000000000022';
		const rentCall2Id = '0192dce0-7c4a-7001-8000-000000000032';

		const rows = buildUnifiedRentRows({
			entities: [
				{
					id: entityId,
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
					id: lease1Id,
					legal_entity_id: entityId,
					lot_id: lot1Id,
					tenant_id: tenant1Id,
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
					id: lease2Id,
					legal_entity_id: entityId,
					lot_id: lot2Id,
					tenant_id: tenant2Id,
					lease_type: 'RESIDENTIAL_UNFURNISHED',
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
					id: tenant1Id,
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
				[entityId]: [
					{
						id: propId,
						legal_entity_id: entityId,
						label: 'Immeuble Rivoli',
						property_type: 'BUILDING',
						accounting_category: 'FIXED_ASSET',
						address: '12 rue de Rivoli',
						acquisition_cents: 50000000,
						work_stock_cents: 0,
						total_book_value_cents: 50000000,
						lots: [
							{
								id: lot1Id,
								property_id: propId,
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
				[entityId]: [
					{
						id: rentCall2Id,
						lease_id: lease2Id,
						legal_entity_id: entityId,
						tenant_id: tenant2Id,
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
				[entityId]: [
					{
						lease_id: lease1Id,
						legal_entity_id: entityId,
						lot_id: lot1Id,
						tenant_id: tenant1Id,
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
		expect(rows[1].propertyAndLotLabel).toBe('Lot a2b3c4d5');

		expect(summarizeRentRows(rows)).toEqual({
			expectedTotalCents: 220000,
			collectedTotalCents: 85000,
			remainingTotalCents: 135000,
			overdueCount: 1
		});
	});
});
