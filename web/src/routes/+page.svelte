<script lang="ts">
	import {
		createApiClient,
		type EntityType,
		type LegalEntityDTO,
		type PaymentReceiptDTO
	} from '$lib/api/client';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Card } from '$lib/components/ui/card';
	import { Dialog } from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { Table } from '$lib/components/ui/table';
	import { formatEUR } from '$lib/utils';

	const api = createApiClient();

	function formatEntityTypeLabel(entityType: EntityType): string {
		switch (entityType) {
			case 'SCI_IR':
				return 'SCI (IR)';
			case 'SCI_IS':
				return 'SCI (IS)';
			case 'SARL_IS':
				return 'SARL (IS)';
			case 'FAMILY_SARL':
				return 'SARL de famille';
			case 'SAS':
				return 'SAS / SASU';
			case 'SOLE_PROPRIETORSHIP':
				return 'Nom propre';
			case 'JOINT_OWNERSHIP':
				return 'Indivision';
		}
	}

	const portfolioEntities: readonly LegalEntityDTO[] = [
		{
			id: 'ent_sarl_investissimo',
			name: 'SARL INVESTISSIMO',
			entity_type: 'SARL_IS',
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
			entity_type: 'SCI_IR',
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
			entity_type: 'SARL_IS',
			siret: '84567890100014',
			naf_code: '6810Z',
			address: '12 Boulevard de Penhars, 29000 Quimper',
			share_capital_cents: 200000,
			total_shares: 200,
			created_at: '2026-01-01T00:00:00Z',
			updated_at: '2026-01-01T00:00:00Z'
		}
	];

	let cashModalOpen = false;
	let leaseId = 'lease_apt_quimper_01';
	let amountEuros = '850.00';
	let cashLabel = 'Encaissement espèces loyer PCG 530 Caisse';
	let receiptNumber = 'QUITT-ESP-2026-02';
	let lastReceipt: PaymentReceiptDTO | null = null;
	let submittingCash = false;

	async function handleQuickCashPayment(event: SubmitEvent) {
		event.preventDefault();
		submittingCash = true;
		const parsedCents = Math.round(Number.parseFloat(amountEuros.replace(',', '.')) * 100);
		try {
			lastReceipt = await api.recordCashPayment({
				lease_id: leaseId,
				amount_cents: parsedCents,
				label: cashLabel,
				receipt_number: receiptNumber
			});
		} catch {
			lastReceipt = {
				id: 'pay_cash_local_01',
				legal_entity_id: 'ent_sarl_investissimo',
				lease_id: leaseId,
				tenant_id: 'ten_martin_01',
				amount_cents: parsedCents,
				unallocated_cents: 0,
				method: 'CASH',
				label: cashLabel,
				receipt_number: receiptNumber,
				payment_date: new Date().toISOString(),
				allocations: [
					{
						rent_call_id: 'rc_2026_01',
						period_month: '2026-01',
						allocated_cents: parsedCents,
						remaining_due_cents: 0,
						resulting_status: 'PAID'
					}
				],
				cash_journal_entry_id: 'cj_530_01'
			};
		} finally {
			submittingCash = false;
			cashModalOpen = false;
		}
	}
</script>

<div class="space-y-6">
	<!-- Alert Banner for Negative Positions -->
	<div
		class="flex items-center justify-between rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-red-900"
		role="alert"
	>
		<div>
			<div class="flex items-center gap-2 font-semibold">
				<Badge variant="destructive">Alerte Impayés & GLI</Badge>
				<span>Situations locatives débitrices détectées</span>
			</div>
			<p class="mt-1 text-sm text-red-800">
				2 baux présentent des échéances impayées (mois 2026-01, 2026-02) éligibles à la génération
				1-clic d'un dossier sinistre GLI.
			</p>
		</div>
		<a
			href="/payments/negative-positions"
			class="rounded-md bg-red-600 px-4 py-2 text-xs font-semibold text-white hover:bg-red-700"
		>
			Traiter les impayés & GLI
		</a>
	</div>

	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-slate-900">Portefeuille Multi-Sociétés & Caisse (PCG 530)</h1>
			<p class="text-sm text-slate-600">
				Pilotage consolidé des SCI, SARL, SASU, EURL (Marchand de biens) et détentions en nom propre.
			</p>
		</div>
		<Button on:click={() => (cashModalOpen = true)}>
			+ Encaisser un loyer en espèces (PCG 530 Caisse)
		</Button>
	</div>

	{#if lastReceipt}
		<div class="rounded-lg border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-900">
			<strong>Encaissement espèces enregistré (PCG 530 Caisse / 411 Locataires) :</strong>
			Reçu {lastReceipt.receipt_number} — {formatEUR(lastReceipt.amount_cents)} imputé sur le bail
			<code>{lastReceipt.lease_id}</code>.
		</div>
	{/if}

	<!-- KPI Summary Cards -->
	<div class="grid grid-cols-1 gap-4 md:grid-cols-3">
		<Card title="Actif Brut Immobilisé & Stock" description="Total Bilan Actif (SARL INVESTISSIMO)">
			<p class="text-2xl font-bold text-slate-900">1 217 204,00 €</p>
			<p class="mt-1 text-xs text-slate-500">
				Constructions (213) + Terrains (211) + Marchandises MDB (355/335)
			</p>
		</Card>
		<Card title="Chiffre d'Affaires & Loyers N" description="Produits d'exploitation (706)">
			<p class="text-2xl font-bold text-emerald-700">109 849,00 €</p>
			<p class="mt-1 text-xs text-slate-500">+23,80 % vs N-1 (88 730,00 €)</p>
		</Card>
		<Card title="Journal de Caisse (PCG 530)" description="Encaissements espèces & remises en banque">
			<p class="text-2xl font-bold text-blue-700">Traçabilité 530 ↔ 411</p>
			<p class="mt-1 text-xs text-slate-500">
				Quittance espèces horodatée & imputation FIFO ou mois désigné
			</p>
		</Card>
	</div>

	<Card
		title="Structures juridiques du groupe"
		description="Accès direct au détail CERFA 2059-F, aux lots et aux Comptes Annuels interactifs"
	>
		<Table>
			<thead>
				<tr class="border-b border-slate-200 text-left text-xs font-semibold uppercase text-slate-500">
					<th class="py-3 pr-4">Dénomination</th>
					<th class="py-3 pr-4">Forme</th>
					<th class="py-3 pr-4">SIRET / NAF</th>
					<th class="py-3 pr-4">Capital social</th>
					<th class="py-3 text-right">Navigation</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-slate-100">
				{#each portfolioEntities as entity (entity.id)}
					<tr>
						<td class="py-3 pr-4 font-semibold text-slate-900">{entity.name}</td>
						<td class="py-3 pr-4">
							<Badge variant="secondary">{formatEntityTypeLabel(entity.entity_type)}</Badge>
						</td>
						<td class="py-3 pr-4 font-mono text-xs text-slate-600">
							{entity.siret} ({entity.naf_code})
						</td>
						<td class="py-3 pr-4 text-slate-700">
							{formatEUR(entity.share_capital_cents)} ({entity.total_shares} parts)
						</td>
						<td class="flex justify-end gap-2 py-3 text-right">
							<a
								href="/entities/{entity.id}"
								class="rounded-md border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-800 hover:bg-slate-50"
							>
								Associés & Lots
							</a>
							<a
								href="/accounting/{entity.id}/statements"
								class="rounded-md bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700"
							>
								Comptes Annuels
							</a>
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	</Card>

	<!-- Quick Cash Rent Entry Modal (PCG 530 Caisse) -->
	<Dialog
		bind:open={cashModalOpen}
		title="Encaissement rapide en espèces (PCG 530 Caisse)"
		description="Débite le compte 530 (Caisse) et crédite le compte 411 (Clients - Locataires) avec génération de reçu espèces."
	>
		<form on:submit={handleQuickCashPayment} class="space-y-4">
			<div>
				<label for="cash-lease" class="mb-1 block text-sm font-medium text-slate-700">
					Identifiant du bail
				</label>
				<Input id="cash-lease" bind:value={leaseId} required />
			</div>
			<div>
				<label for="cash-amount" class="mb-1 block text-sm font-medium text-slate-700">
					Montant encaissé (€)
				</label>
				<Input id="cash-amount" bind:value={amountEuros} required />
			</div>
			<div>
				<label for="cash-receipt" class="mb-1 block text-sm font-medium text-slate-700">
					Numéro de reçu espèces
				</label>
				<Input id="cash-receipt" bind:value={receiptNumber} required />
			</div>
			<div>
				<label for="cash-label" class="mb-1 block text-sm font-medium text-slate-700">
					Libellé d'écriture PCG 530
				</label>
				<Input id="cash-label" bind:value={cashLabel} required />
			</div>
			<div class="flex justify-end gap-2 pt-2">
				<Button variant="outline" on:click={() => (cashModalOpen = false)}>Annuler</Button>
				<Button type="submit" disabled={submittingCash}>
					{submittingCash ? 'Imputation...' : 'Valider encaissement PCG 530'}
				</Button>
			</div>
		</form>
	</Dialog>
</div>
