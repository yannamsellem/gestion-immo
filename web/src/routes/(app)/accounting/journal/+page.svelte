<script lang="ts">
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconBook from '@tabler/icons-svelte/icons/book';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import AccountingContextToolbar from '#lib/components/accounting/AccountingContextToolbar.svelte';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { AccountBalanceLineDTO } from '#lib/modules/accounting/types';
	import CreateJournalEntryDialog from './(components)/CreateJournalEntryDialog.svelte';
	import JournalEntriesTableCard from './(components)/JournalEntriesTableCard.svelte';
	import TrialBalanceTableCard from './(components)/TrialBalanceTableCard.svelte';
	import UpsertBalanceLineDialog from './(components)/UpsertBalanceLineDialog.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	let upsertBalanceDialogOpen = $state(false);
	let createJournalDialogOpen = $state(false);

	const existingBalanceLinesJson = $derived.by(() => {
		const lines: AccountBalanceLineDTO[] = (data.statements?.lines ?? []).map((line) => ({
			line_id: line.line_id,
			account_code: line.account_code,
			label: line.label,
			section: line.section,
			cerfa_code: line.cerfa_code,
			gross_n_cents: line.gross_n_cents,
			depreciation_n_cents: line.depreciation_n_cents,
			net_n_cents: line.net_n_cents,
			net_n_minus_1_cents: line.net_n_minus_1_cents
		}));
		return JSON.stringify(lines);
	});
</script>

<PageHeader
	title="Journal & Balance PCG"
	subtitle="Saisie des postes de balance générale et écritures comptables en partie double conformes au Plan Comptable Général"
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
			onclick={() => {
				upsertBalanceDialogOpen = true;
			}}
		>
			<IconPlus data-icon="inline-start" />
			Ligne de balance
		</Button>

		<Button
			size="sm"
			onclick={() => {
				createJournalDialogOpen = true;
			}}
		>
			<IconBook data-icon="inline-start" />
			Nouvelle écriture OD/PCG
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

<div class="flex flex-col gap-6">
	<TrialBalanceTableCard
		lines={data.statements?.lines ?? []}
		fiscalYear={data.fiscalYear}
		onCreateLine={() => {
			upsertBalanceDialogOpen = true;
		}}
	/>

	<JournalEntriesTableCard
		journalEntries={data.journalEntries}
		fiscalYear={data.fiscalYear}
		onCreateEntry={() => {
			createJournalDialogOpen = true;
		}}
	/>
</div>

<UpsertBalanceLineDialog
	bind:open={upsertBalanceDialogOpen}
	activeEntityId={data.activeEntityId}
	fiscalYear={data.fiscalYear}
	existingLinesJson={existingBalanceLinesJson}
/>

<CreateJournalEntryDialog
	bind:open={createJournalDialogOpen}
	activeEntityId={data.activeEntityId}
	fiscalYear={data.fiscalYear}
/>
