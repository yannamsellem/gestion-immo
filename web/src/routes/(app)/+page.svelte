<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconBuildingBank from '@tabler/icons-svelte/icons/building-bank';
	import IconCalendarEvent from '@tabler/icons-svelte/icons/calendar-event';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import IconRefresh from '@tabler/icons-svelte/icons/refresh';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { useEntityDialogContext } from '#lib/core/workspace/entity-dialog-context';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import CockpitMetricsGrid from './(components)/CockpitMetricsGrid.svelte';
	import PortfolioCockpitGroupCard from './(components)/PortfolioCockpitGroupCard.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const entityDialog = useEntityDialogContext();

	const totalCollectedCents = $derived(
		data.groups.reduce((sum, group) => sum + group.collectedRentCents, 0)
	);
	const totalExpectedCents = $derived(
		data.groups.reduce((sum, group) => sum + group.expectedRentCents, 0)
	);
	const totalRemainingDueCents = $derived(
		data.groups.reduce((sum, group) => sum + group.remainingDueCents, 0)
	);

	function handlePeriodChange(event: Event): void {
		const target = event.currentTarget as HTMLInputElement;
		if (target.value) {
			workspaceStore.setPeriodMonth(target.value);
		}
	}
</script>

<PageHeader
	title="Vue d’ensemble"
	subtitle="Pilotage par entité juridique et par bien — loyers, appels d’échéance, relances et incidents"
>
	{#snippet actions()}
		<div class="flex items-center gap-2">
			<IconCalendarEvent class="size-4 text-muted-foreground" />
			<Input
				type="month"
				value={workspaceStore.selectedPeriodMonth}
				onchange={handlePeriodChange}
				class="h-8 w-40"
				aria-label="Mois de gestion"
			/>
		</div>
		<Button variant="outline" size="sm" onclick={() => void invalidateAll()}>
			<IconRefresh data-icon="inline-start" />
			Actualiser
		</Button>
	{/snippet}
</PageHeader>

{#if data.errorMessage}
	<Alert.Root variant="destructive">
		<IconAlertTriangle />
		<Alert.Title>Erreur de chargement</Alert.Title>
		<Alert.Description>{data.errorMessage}</Alert.Description>
	</Alert.Root>
{/if}

<CockpitMetricsGrid
	periodMonth={workspaceStore.selectedPeriodMonth}
	{totalCollectedCents}
	{totalExpectedCents}
	{totalRemainingDueCents}
	totalOverdueLeasesCount={data.totalOverdueLeasesCount}
	totalOpenIncidentsCount={data.totalOpenIncidentsCount}
/>

{#if data.groups.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconBuildingBank />
					</Empty.Media>
					<Empty.Title>Aucune entité juridique sélectionnée</Empty.Title>
					<Empty.Description>
						Créez une SCI, SARL de famille ou détention en nom propre pour structurer vos biens,
						baux et encaissements.
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
	<div class="flex flex-col gap-5">
		{#each data.groups as group (group.entity.id)}
			<PortfolioCockpitGroupCard {group} />
		{/each}
	</div>
{/if}
