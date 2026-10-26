<script lang="ts">
	import { page } from '$app/stores';
	import type {
		AccountingCategory,
		LeaseDTO,
		LeaseType,
		PropertyDTO,
		ShareholderDTO
	} from '$lib/api/client';
	import { Badge } from '$lib/components/ui/badge';
	import { Card } from '$lib/components/ui/card';
	import { Table } from '$lib/components/ui/table';
	import { Tabs } from '$lib/components/ui/tabs';
	import { formatBasisPointsPct, formatEUR } from '$lib/utils';

	$: entityId = $page.params.id || 'ent_sarl_investissimo';
	let activeTab = 'shareholders';

	function formatAccountingCategoryLabel(category: AccountingCategory): string {
		switch (category) {
			case 'FIXED_ASSET':
				return 'Immobilisation';
			case 'MERCHANDISE':
				return 'Marchandise';
			case 'FINISHED_GOOD':
				return 'Produit fini';
		}
	}

	function formatLeaseTypeLabel(leaseType: LeaseType): string {
		switch (leaseType) {
			case 'RESIDENTIAL_UNFURNISHED':
				return 'Habitation nue';
			case 'RESIDENTIAL_FURNISHED':
				return 'Habitation meublée';
			case 'COMMERCIAL_3_6_9':
				return 'Commercial 3-6-9';
			case 'PROFESSIONAL':
				return 'Professionnel';
			case 'PARKING':
				return 'Parking';
			case 'SHORT_TERM':
				return 'Courte durée';
		}
	}

	const shareholders: readonly ShareholderDTO[] = [
		{
			id: 'sh_yann',
			legal_entity_id: 'ent_sarl_investissimo',
			full_name: 'Yann Amsellem',
			shares_count: 400,
			detention_basis_points: 5000,
			current_account_cents: 10203200,
			remuneration_paid_cents: 0
		},
		{
			id: 'sh_david',
			legal_entity_id: 'ent_sarl_investissimo',
			full_name: 'David Amsellem',
			shares_count: 400,
			detention_basis_points: 5000,
			current_account_cents: 10203100,
			remuneration_paid_cents: 0
		}
	];

	const properties: readonly PropertyDTO[] = [
		{
			id: 'prop_quimper_immeuble',
			legal_entity_id: 'ent_sarl_investissimo',
			label: 'Immeuble Locatif & Commercial — Quimper Centre',
			property_type: 'MIXED_USE_BUILDING',
			accounting_category: 'FIXED_ASSET',
			address: '14 Rue Kéréon, 29000 Quimper',
			acquisition_cents: 112640200,
			work_stock_cents: 6609400,
			total_book_value_cents: 119249600
		},
		{
			id: 'prop_penhars_mdb',
			legal_entity_id: 'ent_eurl_penhars_invest',
			label: 'Plateau à découper — Opération Marchand de Biens Penhars',
			property_type: 'DEALER_LOT',
			accounting_category: 'MERCHANDISE',
			address: '12 Boulevard de Penhars, 29000 Quimper',
			acquisition_cents: 18500000,
			work_stock_cents: 4250000,
			total_book_value_cents: 22750000
		}
	];

	const leases: readonly LeaseDTO[] = [
		{
			id: 'lease_hab_01',
			legal_entity_id: 'ent_sarl_investissimo',
			lot_id: 'lot_t2_1er',
			tenant_id: 'ten_dupont',
			lease_type: 'RESIDENTIAL_UNFURNISHED',
			rent_net_cents: 65000,
			charges_cents: 6000,
			subject_to_vat: false,
			vat_rate_basis_points: 0,
			security_deposit_cents: 65000,
			due_day_of_month: 5,
			gli_policy_number: 'GLI-AXA-2026-8891',
			guarantor_name: 'Garantie Loyers Impayés AXA',
			start_date: '2025-01-01T00:00:00Z',
			active: true
		},
		{
			id: 'lease_comm_01',
			legal_entity_id: 'ent_sarl_investissimo',
			lot_id: 'lot_rdc_comm',
			tenant_id: 'ten_boutique_optique',
			lease_type: 'COMMERCIAL_3_6_9',
			rent_net_cents: 180000,
			charges_cents: 15000,
			subject_to_vat: true,
			vat_rate_basis_points: 2000,
			security_deposit_cents: 540000,
			due_day_of_month: 1,
			gli_policy_number: '',
			guarantor_name: 'Caution Bancaire Solidaire',
			start_date: '2024-06-01T00:00:00Z',
			active: true
		}
	];
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-slate-900">
				SARL INVESTISSIMO — Associés (2059-F), Patrimoine & Baux
			</h1>
			<p class="text-sm text-slate-600">
				SIRET 519 465 876 00028 • NAF 6820B • Capital 8 000,00 € (800 parts sociales)
			</p>
		</div>
		<a
			href="/accounting/{entityId}/statements"
			class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
		>
			Ouvrir les Comptes Annuels & Liasse 2033/2065
		</a>
	</div>

	<Tabs
		bind:activeTab
		tabs={[
			{ id: 'shareholders', label: 'Associés & Comptes Courants (2059-F)', badge: shareholders.length },
			{ id: 'properties', label: 'Biens & Stock Marchand de Biens', badge: properties.length },
			{ id: 'leases', label: 'Baux Actifs & Quittancement', badge: leases.length }
		]}
	>
		{#if activeTab === 'shareholders'}
			<Card
				title="Répartition du capital social & Comptes courants d'associés (PCG 455)"
				description="Données synchronisées avec le feuillet CERFA 2059-F (Composition du capital social)"
			>
				<Table>
					<thead>
						<tr class="border-b border-slate-200 text-left text-xs font-semibold uppercase text-slate-500">
							<th class="py-3 pr-4">Associé</th>
							<th class="py-3 pr-4">Nombre de parts</th>
							<th class="py-3 pr-4">% Détention</th>
							<th class="py-3 pr-4">Compte courant (PCG 455)</th>
							<th class="py-3 text-right">Rémunération N</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100">
						{#each shareholders as sh (sh.id)}
							<tr>
								<td class="py-3 pr-4 font-semibold text-slate-900">{sh.full_name}</td>
								<td class="py-3 pr-4">{sh.shares_count} parts</td>
								<td class="py-3 pr-4"><Badge variant="default">{formatBasisPointsPct(sh.detention_basis_points)}</Badge></td>
								<td class="py-3 pr-4 font-mono text-slate-800">{formatEUR(sh.current_account_cents)}</td>
								<td class="py-3 text-right font-mono text-slate-600">
									{formatEUR(sh.remuneration_paid_cents)}
								</td>
							</tr>
						{/each}
					</tbody>
				</Table>
			</Card>
		{:else if activeTab === 'properties'}
			<Card
				title="Actifs Immobiliers (Immobilisations PCG 211/213 vs Marchandises PCG 355/335)"
				description="Distinction comptable stricte entre patrimoine locatif immobilisé et stock Marchand de Biens"
			>
				<Table>
					<thead>
						<tr class="border-b border-slate-200 text-left text-xs font-semibold uppercase text-slate-500">
							<th class="py-3 pr-4">Désignation</th>
							<th class="py-3 pr-4">Qualification PCG</th>
							<th class="py-3 pr-4">Prix d'acquisition</th>
							<th class="py-3 pr-4">Stock travaux / Agencements</th>
							<th class="py-3 text-right">Valeur totale inscrite</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100">
						{#each properties as prop (prop.id)}
							<tr>
								<td class="py-3 pr-4">
									<div class="font-semibold text-slate-900">{prop.label}</div>
									<div class="text-xs text-slate-500">{prop.address}</div>
								</td>
								<td class="py-3 pr-4">
									<Badge
										variant={prop.accounting_category === 'FIXED_ASSET' ? 'default' : 'warning'}
									>
										{formatAccountingCategoryLabel(prop.accounting_category)}
									</Badge>
								</td>
								<td class="py-3 pr-4 font-mono">{formatEUR(prop.acquisition_cents)}</td>
								<td class="py-3 pr-4 font-mono">{formatEUR(prop.work_stock_cents)}</td>
								<td class="py-3 text-right font-mono font-semibold text-slate-900">
									{formatEUR(prop.total_book_value_cents)}
								</td>
							</tr>
						{/each}
					</tbody>
				</Table>
			</Card>
		{:else}
			<Card
				title="Baux d'habitation (exonérés TVA) & Baux commerciaux (TVA 20 %)"
				description="Suivi des loyers, provisions sur charges, dépôts de garantie (PCG 165) et polices GLI"
			>
				<Table>
					<thead>
						<tr class="border-b border-slate-200 text-left text-xs font-semibold uppercase text-slate-500">
							<th class="py-3 pr-4">Bail / Lot</th>
							<th class="py-3 pr-4">Nature</th>
							<th class="py-3 pr-4">Loyer HC</th>
							<th class="py-3 pr-4">Charges</th>
							<th class="py-3 pr-4">TVA</th>
							<th class="py-3 text-right">Garantie / Police GLI</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100">
						{#each leases as lease (lease.id)}
							<tr>
								<td class="py-3 pr-4 font-mono text-xs font-semibold text-slate-900">
									{lease.id} ({lease.lot_id})
								</td>
								<td class="py-3 pr-4">
									<Badge variant="secondary">{formatLeaseTypeLabel(lease.lease_type)}</Badge>
								</td>
								<td class="py-3 pr-4 font-mono">{formatEUR(lease.rent_net_cents)}</td>
								<td class="py-3 pr-4 font-mono">{formatEUR(lease.charges_cents)}</td>
								<td class="py-3 pr-4">
									{#if lease.subject_to_vat}
										<Badge variant="warning">TVA 20,00 %</Badge>
									{:else}
										<Badge variant="outline">Exonéré</Badge>
									{/if}
								</td>
								<td class="py-3 text-right text-xs text-slate-700">
									{lease.gli_policy_number || lease.guarantor_name}
								</td>
							</tr>
						{/each}
					</tbody>
				</Table>
			</Card>
		{/if}
	</Tabs>
</div>
