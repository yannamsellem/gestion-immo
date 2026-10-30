<script lang="ts">
	import { enhance } from '$app/forms';
	import IconDownload from '@tabler/icons-svelte/icons/download';
	import IconFileText from '@tabler/icons-svelte/icons/file-text';
	import { toast } from 'svelte-sonner';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { formatCentsToEuro } from '#lib/core/formatters/money';
	import type {
		AnnualAccountsReportDTO,
		ComputedFinancialStatementsDTO
	} from '#lib/modules/accounting/types';
	import { formatEntityTypeLabel } from '#lib/modules/portfolio/labels';
	import type { LegalEntityDTO } from '#lib/modules/portfolio/types';

	let {
		activeEntity,
		fiscalYear,
		statements,
		disabledLineIds
	}: {
		activeEntity: LegalEntityDTO | null;
		fiscalYear: number;
		statements: ComputedFinancialStatementsDTO | null;
		disabledLineIds: string[];
	} = $props();

	let isGeneratingPdf = $state(false);
	let generatedReport = $state<AnnualAccountsReportDTO | null>(null);

	$effect(() => {
		void activeEntity?.id;
		void fiscalYear;
		generatedReport = null;
	});
</script>

<Card.Root class="flex flex-col justify-between">
	<Card.Header class="border-b pb-3">
		<div class="flex flex-wrap items-center justify-between gap-2">
			<Card.Title class="text-sm font-semibold">
				Plaquette Comptes Annuels & Liasse 2072 / 2033
			</Card.Title>
			{#if activeEntity}
				<Badge variant="outline">{formatEntityTypeLabel(activeEntity.entity_type)}</Badge>
			{/if}
		</div>
		<Card.Description>
			Génère la plaquette PDF complète (Bilan Actif/Passif, Compte de résultat, SIG et
			correspondance CERFA 2072 ou 2033) pour l’exercice {fiscalYear}.
		</Card.Description>
	</Card.Header>

	<Card.Content class="flex flex-col gap-4 py-4">
		{#if statements}
			<div class="grid gap-3 rounded-lg border bg-muted/30 p-3 text-xs sm:grid-cols-2">
				<div class="flex flex-col gap-0.5">
					<span class="text-muted-foreground">Produits d’exploitation ({fiscalYear})</span>
					<span class="font-mono text-sm font-semibold tabular-nums">
						{formatCentsToEuro(statements.total_operating_revenue_n_cents)}
					</span>
				</div>
				<div class="flex flex-col gap-0.5">
					<span class="text-muted-foreground">Résultat Net Comptable ({fiscalYear})</span>
					<span class="font-mono text-sm font-semibold tabular-nums">
						{formatCentsToEuro(statements.net_income_n_cents)}
					</span>
				</div>
				<div class="flex flex-col gap-0.5">
					<span class="text-muted-foreground">Capacité d’Autofinancement (CAF)</span>
					<span class="font-mono font-medium tabular-nums">
						{formatCentsToEuro(statements.self_financing_capacity_n_cents)}
					</span>
				</div>
				<div class="flex flex-col gap-0.5">
					<span class="text-muted-foreground">Postes de balance inclus</span>
					<span class="font-medium">
						{statements.lines.length} poste(s) · {disabledLineIds.length} neutralisé(s)
					</span>
				</div>
			</div>
		{/if}

		{#if generatedReport}
			<Separator />
			<div
				class="flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-muted/40 p-3"
			>
				<div class="flex flex-col gap-0.5 text-xs">
					<span class="font-semibold">{generatedReport.file_name}</span>
					<span class="text-muted-foreground tabular-nums">
						Résultat Net : {formatCentsToEuro(generatedReport.net_income_n_cents)} ·
						{generatedReport.disabled_line_ids.length} ligne(s) neutralisée(s)
					</span>
				</div>
				{#if generatedReport.download_url}
					<Button
						size="sm"
						variant="secondary"
						href={generatedReport.download_url}
						target="_blank"
						rel="noreferrer"
					>
						<IconDownload data-icon="inline-start" />
						Ouvrir le PDF
					</Button>
				{/if}
			</div>
		{/if}
	</Card.Content>

	<Card.Footer class="border-t pt-3">
		<form
			method="POST"
			action="?/generatePdf"
			use:enhance={() => {
				isGeneratingPdf = true;
				return async ({ result }) => {
					isGeneratingPdf = false;
					if (result.type === 'success') {
						const payload = result.data as
							| {
									generatedReport?: AnnualAccountsReportDTO;
									successMessage?: string;
							  }
							| undefined;
						if (payload?.generatedReport) {
							generatedReport = payload.generatedReport;
						}
						toast.success(
							payload?.successMessage ?? `Plaquette des comptes annuels ${fiscalYear} générée.`
						);
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						toast.error(
							payload?.actionError ?? 'Échec de la génération du PDF des comptes annuels.'
						);
						return;
					}
					if (result.type === 'error') {
						toast.error(
							result.error?.message ?? 'Erreur serveur lors de la génération de la plaquette PDF.'
						);
					}
				};
			}}
			class="w-full"
		>
			<input type="hidden" name="entity_id" value={activeEntity?.id ?? ''} />
			<input type="hidden" name="fiscal_year" value={fiscalYear} />
			<input type="hidden" name="disabled_line_ids_json" value={JSON.stringify(disabledLineIds)} />
			<Button type="submit" class="w-full" disabled={isGeneratingPdf || !activeEntity}>
				{#if isGeneratingPdf}
					<Spinner data-icon="inline-start" />
				{:else}
					<IconFileText data-icon="inline-start" />
				{/if}
				Générer la plaquette PDF {fiscalYear}
			</Button>
		</form>
	</Card.Footer>
</Card.Root>
