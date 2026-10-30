<script lang="ts">
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconBuildingBank from '@tabler/icons-svelte/icons/building-bank';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import type { PageProps } from './$types';
	import CashDepositDialog from './(components)/CashDepositDialog.svelte';
	import CashJournalTableCard from './(components)/CashJournalTableCard.svelte';

	let { data }: PageProps = $props();

	let depositDialogOpen = $state(false);
	let autoHandledUrl = $state(false);

	const defaultEntityId = $derived(
		workspaceStore.selectedEntityId !== 'ALL' ? workspaceStore.selectedEntityId : ''
	);

	$effect(() => {
		const action = page.url.searchParams.get('action');
		if (!autoHandledUrl && (action === 'create' || action === 'deposit')) {
			depositDialogOpen = true;
			autoHandledUrl = true;
		}
	});
</script>

<PageHeader
	title="Livre de caisse espèces (PCG 530)"
	subtitle="Traçabilité chronologique des encaissements en espèces (411 → 530) et remises en banque (530 → 512)"
>
	{#snippet actions()}
		<Button size="sm" onclick={() => (depositDialogOpen = true)}>
			<IconBuildingBank data-icon="inline-start" />
			Remise espèces (530 → 512)
		</Button>
	{/snippet}
</PageHeader>

{#if data.errorMessage}
	<Alert.Root variant="destructive">
		<IconAlertTriangle class="size-4" />
		<Alert.Title>Erreur</Alert.Title>
		<Alert.Description>{data.errorMessage}</Alert.Description>
	</Alert.Root>
{/if}

<CashJournalTableCard
	cashEntries={data.cashEntries}
	entities={data.entities}
	onRecordDeposit={() => (depositDialogOpen = true)}
/>

<CashDepositDialog bind:open={depositDialogOpen} entities={data.entities} {defaultEntityId} />
