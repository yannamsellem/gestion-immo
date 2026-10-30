<script lang="ts">
	import { resolve } from '$app/paths';
	import IconFileExport from '@tabler/icons-svelte/icons/file-export';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import type { LegalEntityDTO } from '#lib/modules/portfolio/types';

	let {
		activeEntity,
		fiscalYear
	}: {
		activeEntity: LegalEntityDTO | null;
		fiscalYear: number;
	} = $props();

	const siren = $derived((activeEntity?.siret ?? '000000000').slice(0, 9) || '000000000');
	const fecFileName = $derived(`${siren}FEC${fiscalYear}1231.tsv`);
	const fecExportHref = $derived(
		activeEntity
			? `${resolve('/(app)/accounting')}/fec?entityId=${encodeURIComponent(activeEntity.id)}&fiscalYear=${fiscalYear}&siren=${encodeURIComponent(siren)}`
			: ''
	);
</script>

<Card.Root class="flex flex-col justify-between">
	<Card.Header class="border-b pb-3">
		<div class="flex flex-wrap items-center justify-between gap-2">
			<Card.Title class="text-sm font-semibold">
				Fichier des Écritures Comptables (Export FEC DGFiP)
			</Card.Title>
			<Badge variant="secondary" class="font-mono text-xs">{fecFileName}</Badge>
		</div>
		<Card.Description>
			Export normatif TSV 18 colonnes conforme à l’article A.47 A-1 du Livre des Procédures Fiscales
			(séparateur tabulation, virgule décimale française, encodage UTF-8).
		</Card.Description>
	</Card.Header>

	<Card.Content class="flex flex-col gap-3 py-4 text-xs text-muted-foreground">
		<div class="grid gap-3 rounded-lg border bg-muted/30 p-3 sm:grid-cols-2">
			<div class="flex flex-col gap-0.5">
				<span>SIREN retenu (9 chiffres)</span>
				<span class="font-mono text-sm font-semibold text-foreground">{siren}</span>
			</div>
			<div class="flex flex-col gap-0.5">
				<span>Date de clôture normative</span>
				<span class="font-mono text-sm font-semibold text-foreground">31/12/{fiscalYear}</span>
			</div>
		</div>
		<p>
			Toutes les écritures validées dans le livre-journal PCG ainsi que les écritures d’encaissement
			et de dépôt d’espèces (compte 530) de l’exercice {fiscalYear} sont incluses dans le flux.
		</p>
	</Card.Content>

	<Card.Footer class="border-t pt-3">
		{#if fecExportHref}
			<Button variant="outline" class="w-full" href={fecExportHref}>
				<IconFileExport data-icon="inline-start" />
				Télécharger l’export FEC (.tsv)
			</Button>
		{:else}
			<Button variant="outline" class="w-full" disabled>
				<IconFileExport data-icon="inline-start" />
				Télécharger l’export FEC (.tsv)
			</Button>
		{/if}
	</Card.Footer>
</Card.Root>
