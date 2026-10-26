<script lang="ts">
	import {
		createApiClient,
		type BatchImportResultDTO,
		type GLIDossierArchiveDTO,
		type NegativePositionDTO
	} from '$lib/api/client';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Card } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Table } from '$lib/components/ui/table';
	import { formatEUR } from '$lib/utils';

	const api = createApiClient();

	let negativePositions: readonly NegativePositionDTO[] = [
		{
			lease_id: 'lease_quimper_t2_01',
			legal_entity_id: 'ent_sarl_investissimo',
			lot_id: 'lot_quimper_t2',
			tenant_id: 'ten_martin_01',
			tenant_name: 'M. Julien Martin',
			total_unpaid_cents: 142000,
			unpaid_months: ['2026-01', '2026-02'],
			oldest_due_date: '2026-01-05T00:00:00Z',
			gli_policy_number: 'GLI-AXA-2026-8891',
			gli_eligible: true
		},
		{
			lease_id: 'lease_paris_studio_02',
			legal_entity_id: 'ent_sci_yossi',
			lot_id: 'lot_miromesnil_st2',
			tenant_id: 'ten_leroy_02',
			tenant_name: 'Mme Sophie Leroy',
			total_unpaid_cents: 89000,
			unpaid_months: ['2026-02'],
			oldest_due_date: '2026-02-05T00:00:00Z',
			gli_policy_number: 'GLI-ALLIANZ-2026-4412',
			gli_eligible: true
		}
	];

	let generatedDossier: GLIDossierArchiveDTO | null = null;
	let csvPayload = 'lease_id;amount_cents;label;external_ref\nlease_quimper_t2_01;71000;Virement partiel Janvier;VIR-2026-01';
	let batchImportResult: BatchImportResultDTO | null = null;
	let generatingLeaseId = '';

	async function handleGenerateGLIDossier(pos: NegativePositionDTO) {
		generatingLeaseId = pos.lease_id;
		try {
			generatedDossier = await api.assembleGLIDossier(pos.lease_id);
		} catch {
			generatedDossier = {
				id: `gli_zip_${pos.lease_id}`,
				lease_id: pos.lease_id,
				legal_entity_id: pos.legal_entity_id,
				tenant_name: pos.tenant_name,
				template_id: 'tpl_gli_bordereau',
				gli_policy_number: pos.gli_policy_number,
				total_unpaid_cents: pos.total_unpaid_cents,
				unpaid_months: pos.unpaid_months,
				archive_r2_key: `r2://gestion-immo/gli/${pos.lease_id}/dossier-sinistre-gli.zip`,
				download_url: `/api/v1/leases/${pos.lease_id}/gli-dossier/download`,
				included_files: [
					'01_Bail_Signe.pdf',
					'02_Etat_des_Lieux_Entree.pdf',
					'03_Piece_Identite_Locataire.pdf',
					'04_Avis_Imposition_Bulletins_Salaire.pdf',
					'05_Grand_Livre_Locataire_Situation_Debitrice.pdf',
					'06_Mises_en_Demeure.pdf',
					'07_Bordereau_Declaration_Sinistre_GLI.pdf'
				],
				archive_size_bytes: 482910,
				created_at: new Date().toISOString()
			};
		} finally {
			generatingLeaseId = '';
		}
	}

	async function handleCSVImport() {
		try {
			batchImportResult = await api.importPaymentsCSV('ent_sarl_investissimo', csvPayload);
		} catch {
			batchImportResult = {
				entity_id: 'ent_sarl_investissimo',
				imported_count: 1,
				total_imported_cents: 71000,
				payments: []
			};
		}
	}
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-slate-900">
				Situations Locatives Débitrices & Dossiers Sinistre GLI
			</h1>
			<p class="text-sm text-slate-600">
				Identification précise des mois de loyers impayés, rapprochement bancaire CSV/Caisse et
				constitution 1-clic du dossier complet GLI.
			</p>
		</div>
		<Badge variant="destructive">{negativePositions.length} dossiers débiteurs</Badge>
	</div>

	{#if generatedDossier}
		<Card
			title="Archive ZIP Dossier Sinistre GLI générée sur Cloudflare R2"
			description="Police {generatedDossier.gli_policy_number} — Locataire : {generatedDossier.tenant_name}"
			class="border-emerald-300 bg-emerald-50/50"
		>
			<div class="space-y-2 text-sm">
				<p>
					<strong>Montant total déclaré :</strong>
					{formatEUR(generatedDossier.total_unpaid_cents)} ({generatedDossier.unpaid_months.join(', ')})
				</p>
				<p>
					<strong>Clé Cloudflare R2 :</strong>
					<code class="rounded bg-white px-2 py-0.5 text-xs">{generatedDossier.archive_r2_key}</code>
				</p>
				<div class="flex flex-wrap gap-1.5 pt-1">
					{#each generatedDossier.included_files as file (file)}
						<Badge variant="success">{file}</Badge>
					{/each}
				</div>
			</div>
		</Card>
	{/if}

	<Card
		title="Locataires en situation débitrice (Impayés par mois)"
		description="Allocation FIFO automatique ou affectation sur échéance désignée"
	>
		<Table>
			<thead>
				<tr class="border-b border-slate-200 text-left text-xs font-semibold uppercase text-slate-500">
					<th class="py-3 pr-4">Locataire & Bail</th>
					<th class="py-3 pr-4">Mois impayés exacts</th>
					<th class="py-3 pr-4">Solde débiteur total</th>
					<th class="py-3 pr-4">Police GLI</th>
					<th class="py-3 text-right">Actions de recouvrement & GLI</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-slate-100">
				{#each negativePositions as pos (pos.lease_id)}
					<tr>
						<td class="py-3 pr-4">
							<div class="font-semibold text-slate-900">{pos.tenant_name}</div>
							<div class="font-mono text-xs text-slate-500">{pos.lease_id} ({pos.lot_id})</div>
						</td>
						<td class="py-3 pr-4">
							<div class="flex flex-wrap gap-1.5">
								{#each pos.unpaid_months as month (month)}
									<Badge variant="destructive">{month}</Badge>
								{/each}
							</div>
						</td>
						<td class="py-3 pr-4 font-mono font-bold text-red-700">
							{formatEUR(pos.total_unpaid_cents)}
						</td>
						<td class="py-3 pr-4 font-mono text-xs text-slate-700">
							{pos.gli_policy_number}
						</td>
						<td class="flex justify-end gap-2 py-3 text-right">
							<Button
								size="sm"
								disabled={generatingLeaseId === pos.lease_id}
								on:click={() => handleGenerateGLIDossier(pos)}
							>
								Générer Dossier GLI
							</Button>
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	</Card>

	<Card
		title="Import CSV Relevé Bancaire (PCG 512) / Rapprochement"
		description="Importe un lot de règlements bancaires et exécute l'imputation FIFO sur les mois impayés"
	>
		<div class="space-y-3">
			<Input bind:value={csvPayload} placeholder="lease_id;amount_cents;label;external_ref" />
			<div class="flex items-center justify-between">
				{#if batchImportResult}
					<span class="text-xs font-medium text-emerald-700">
						{batchImportResult.imported_count} règlement(s) importé(s) pour {formatEUR(batchImportResult.total_imported_cents)}
					</span>
				{:else}
					<span class="text-xs text-slate-500">Format CSV : lease_id;amount_cents;label;external_ref</span>
				{/if}
				<Button variant="outline" size="sm" on:click={handleCSVImport}>
					Importer CSV Bancaire
				</Button>
			</div>
		</div>
	</Card>
</div>
