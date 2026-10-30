<script lang="ts">
	import { resolve } from '$app/paths';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconCalculator from '@tabler/icons-svelte/icons/calculator';
	import AccountingContextToolbar from '#lib/components/accounting/AccountingContextToolbar.svelte';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import AnnualAccountsPdfCard from './(components)/AnnualAccountsPdfCard.svelte';
	import FecExportCard from './(components)/FecExportCard.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const activeEntity = $derived(
		data.entities.find((entity) => entity.id === data.activeEntityId) ?? null
	);
</script>

<PageHeader
	title="Liasse fiscale & Export FEC"
	subtitle="Édition de la plaquette annuelle CERFA 2072/2033 en PDF et téléchargement du Fichier des Écritures Comptables normatif DGFiP"
>
	{#snippet actions()}
		<AccountingContextToolbar
			entities={data.entities}
			activeEntityId={data.activeEntityId}
			fiscalYear={data.fiscalYear}
		/>

		<Button
			size="sm"
			variant="outline"
			href={`${resolve('/(app)/accounting')}?year=${data.fiscalYear}`}
		>
			<IconCalculator data-icon="inline-start" />
			Bilan & Compte de résultat
		</Button>
	{/snippet}
</PageHeader>

{#if data.errorMessage}
	<Alert.Root variant="destructive">
		<IconAlertTriangle />
		<Alert.Title>Erreur</Alert.Title>
		<Alert.Description>{data.errorMessage}</Alert.Description>
	</Alert.Root>
{/if}

<div class="grid gap-6 lg:grid-cols-2">
	<AnnualAccountsPdfCard
		{activeEntity}
		fiscalYear={data.fiscalYear}
		statements={data.statements}
		disabledLineIds={data.disabledLineIds}
	/>

	<FecExportCard {activeEntity} fiscalYear={data.fiscalYear} />
</div>
