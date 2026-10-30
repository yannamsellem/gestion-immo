<script lang="ts">
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconBuildingSkyscraper from '@tabler/icons-svelte/icons/building-skyscraper';
	import IconDoor from '@tabler/icons-svelte/icons/door';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import CreateLotDialog from './(components)/CreateLotDialog.svelte';
	import CreatePropertyDialog from './(components)/CreatePropertyDialog.svelte';
	import PropertyLotsCard from './(components)/PropertyLotsCard.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	let createPropertyOpen = $state(false);
	let createLotOpen = $state(false);
	let preselectedEntityId = $state('');
	let preselectedPropertyId = $state('');
	let autoHandledUrl = $state(false);

	const defaultEntityId = $derived(
		preselectedEntityId ||
			(workspaceStore.selectedEntityId !== 'ALL'
				? workspaceStore.selectedEntityId
				: (data.entities[0]?.id ?? ''))
	);

	const entityMap = $derived(new Map(data.entities.map((entity) => [entity.id, entity])));
	const activeLeaseByLotId = $derived(new Map(data.leases.map((lease) => [lease.lot_id, lease])));

	$effect(() => {
		if (!autoHandledUrl) {
			const action = page.url.searchParams.get('action');
			const urlEntityId = page.url.searchParams.get('entityId');
			const urlPropertyId = page.url.searchParams.get('propertyId');
			if (urlEntityId) {
				preselectedEntityId = urlEntityId;
			}
			if (action === 'create') {
				createPropertyOpen = true;
				autoHandledUrl = true;
			} else if (
				urlPropertyId &&
				data.properties.some((property) => property.id === urlPropertyId)
			) {
				preselectedPropertyId = urlPropertyId;
				createLotOpen = true;
				autoHandledUrl = true;
			}
		}
	});

	function handleOpenCreateLotForProperty(propertyId: string): void {
		preselectedPropertyId = propertyId;
		createLotOpen = true;
	}
</script>

<PageHeader
	title="Biens & Lots Immobiliers"
	subtitle="Actifs immobilisés (PCG 213), stocks marchand de biens (PCG 355) et découpage par lots locatifs"
>
	{#snippet actions()}
		{#if data.properties.length > 0}
			<Button
				type="button"
				size="sm"
				variant="outline"
				onclick={() => {
					preselectedPropertyId = preselectedPropertyId || data.properties[0].id;
					createLotOpen = true;
				}}
			>
				<IconDoor data-icon="inline-start" />
				Ajouter un lot
			</Button>
		{/if}
		<Button
			type="button"
			size="sm"
			onclick={() => {
				createPropertyOpen = true;
			}}
			disabled={data.entities.length === 0}
		>
			<IconPlus data-icon="inline-start" />
			Nouveau bien
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

{#if data.properties.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconBuildingSkyscraper />
					</Empty.Media>
					<Empty.Title>Aucun bien immobilier enregistré</Empty.Title>
					<Empty.Description>
						Ajoutez vos immeubles, appartements ou locaux commerciaux puis créez leurs lots pour y
						rattacher des baux.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button
						type="button"
						onclick={() => {
							createPropertyOpen = true;
						}}
					>
						<IconPlus data-icon="inline-start" />
						Ajouter un premier bien
					</Button>
				</Empty.Content>
			</Empty.Root>
		</Card.Content>
	</Card.Root>
{:else}
	<div class="flex flex-col gap-5">
		{#each data.properties as property (property.id)}
			<PropertyLotsCard
				{property}
				entityName={entityMap.get(property.legal_entity_id)?.name ?? 'Entité'}
				{activeLeaseByLotId}
				onAddLot={handleOpenCreateLotForProperty}
			/>
		{/each}
	</div>
{/if}

<CreatePropertyDialog
	bind:open={createPropertyOpen}
	entities={data.entities}
	{defaultEntityId}
	onCreated={(created) => {
		preselectedPropertyId = created.id;
	}}
/>

<CreateLotDialog
	bind:open={createLotOpen}
	properties={data.properties}
	defaultPropertyId={preselectedPropertyId || data.properties[0]?.id || ''}
/>
