<script lang="ts">
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconBuildingBank from '@tabler/icons-svelte/icons/building-bank';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import IconUsers from '@tabler/icons-svelte/icons/users';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import { useEntityDialogContext } from '#lib/core/workspace/entity-dialog-context';
	import EntitiesMetricsGrid from './(components)/EntitiesMetricsGrid.svelte';
	import EntitiesTableCard from './(components)/EntitiesTableCard.svelte';
	import ShareholdersTableCard from './(components)/ShareholdersTableCard.svelte';
	import UpsertShareholderDialog from './(components)/UpsertShareholderDialog.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const entityDialog = useEntityDialogContext();

	let focusedEntityId = $state('');
	let shareholderDialogOpen = $state(false);
	let shareholderEntityId = $state('');
	let autoOpenedCreate = $state(false);

	const visibleSummaries = $derived(Object.values(data.summariesByEntity));
	const focusedSummary = $derived(
		data.summariesByEntity[focusedEntityId] ?? visibleSummaries[0] ?? null
	);

	$effect(() => {
		if (!autoOpenedCreate && page.url.searchParams.get('action') === 'create') {
			entityDialog.openCreateDialog();
			autoOpenedCreate = true;
		}
	});

	$effect(() => {
		if (visibleSummaries.length > 0 && !data.summariesByEntity[focusedEntityId]) {
			focusedEntityId = visibleSummaries[0].entity.id;
		}
	});

	const totalAcquisitionCents = $derived(
		visibleSummaries.reduce((sum, summary) => sum + summary.total_acquisition_cents, 0)
	);
	const totalWorkStockCents = $derived(
		visibleSummaries.reduce((sum, summary) => sum + summary.total_work_stock_cents, 0)
	);
	const totalBookValueCents = $derived(
		visibleSummaries.reduce((sum, summary) => sum + summary.total_book_value_cents, 0)
	);
	const totalLotsCount = $derived(
		visibleSummaries.reduce((sum, summary) => sum + summary.total_lots_count, 0)
	);

	function handleOpenShareholderDialog(entityId: string): void {
		shareholderEntityId = entityId;
		shareholderDialogOpen = true;
	}
</script>

<PageHeader
	title="Entités Juridiques & Associés (CERFA 2059-F)"
	subtitle="Structures sociales (SCI IS/IR, SARL, SAS, nom propre), valeur comptable du patrimoine et répartition du capital"
>
	{#snippet actions()}
		{#if focusedSummary}
			<Button
				type="button"
				size="sm"
				variant="outline"
				onclick={() => handleOpenShareholderDialog(focusedSummary.entity.id)}
			>
				<IconUsers data-icon="inline-start" />
				Ajouter un associé
			</Button>
		{/if}
		<Button type="button" size="sm" onclick={() => entityDialog.openCreateDialog()}>
			<IconPlus data-icon="inline-start" />
			Nouvelle entité
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

<EntitiesMetricsGrid
	{totalAcquisitionCents}
	{totalWorkStockCents}
	{totalBookValueCents}
	{totalLotsCount}
	entitiesCount={visibleSummaries.length}
/>

{#if visibleSummaries.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconBuildingBank />
					</Empty.Media>
					<Empty.Title>Aucune entité juridique</Empty.Title>
					<Empty.Description>
						Créez une entité pour gérer son capital social, ses associés et ses biens immobiliers.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button type="button" onclick={() => entityDialog.openCreateDialog()}>
						<IconPlus data-icon="inline-start" />
						Créer une entité juridique
					</Button>
				</Empty.Content>
			</Empty.Root>
		</Card.Content>
	</Card.Root>
{:else}
	<div class="grid gap-6 lg:grid-cols-12">
		<EntitiesTableCard
			summaries={visibleSummaries}
			selectedEntityId={focusedSummary?.entity.id ?? ''}
			onSelectEntity={(entityId) => {
				focusedEntityId = entityId;
			}}
		/>
		<ShareholdersTableCard
			summary={focusedSummary}
			onAddShareholder={handleOpenShareholderDialog}
		/>
	</div>
{/if}

<UpsertShareholderDialog
	bind:open={shareholderDialogOpen}
	entities={data.entities}
	defaultEntityId={shareholderEntityId || focusedSummary?.entity.id || ''}
/>
