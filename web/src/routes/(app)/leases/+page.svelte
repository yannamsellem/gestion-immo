<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import IconUsers from '@tabler/icons-svelte/icons/users';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import type { LeaseDTO } from '#lib/modules/lease/types';
	import type { LotDTO, PropertyDTO } from '#lib/modules/portfolio/types';
	import type { PageProps } from './$types';
	import CreateLeaseDialog from './(components)/CreateLeaseDialog.svelte';
	import LeaseDocumentsSheet from './(components)/LeaseDocumentsSheet.svelte';
	import LeasesTableCard from './(components)/LeasesTableCard.svelte';

	let { data }: PageProps = $props();

	let createLeaseOpen = $state(false);
	let preselectedLotId = $state('');
	let preselectedEntityId = $state('');
	let leaseDocsSheetOpen = $state(false);
	let selectedLease = $state<LeaseDTO | null>(null);
	let autoHandledUrl = $state(false);

	const entityMap = $derived(new Map(data.entities.map((entity) => [entity.id, entity])));
	const tenantMap = $derived(new Map(data.tenants.map((tenant) => [tenant.id, tenant])));
	const lotInfoMap = $derived(
		new Map<string, { lot: LotDTO; property: PropertyDTO }>(
			data.properties.flatMap((property) =>
				(property.lots ?? []).map((lot) => [lot.id, { lot, property }] as const)
			)
		)
	);

	const defaultEntityId = $derived(
		preselectedEntityId ||
			(workspaceStore.selectedEntityId !== 'ALL' ? workspaceStore.selectedEntityId : '')
	);

	$effect(() => {
		if (!autoHandledUrl) {
			const action = page.url.searchParams.get('action');
			const urlLotId = page.url.searchParams.get('lotId') ?? '';
			const urlEntityId = page.url.searchParams.get('entityId') ?? '';
			if (urlLotId || action === 'create') {
				preselectedLotId = urlLotId;
				preselectedEntityId = urlEntityId;
				createLeaseOpen = true;
				autoHandledUrl = true;
			}
		}
	});

	function openCreateLease(): void {
		preselectedLotId = '';
		preselectedEntityId = '';
		createLeaseOpen = true;
	}

	function openLeaseDocuments(lease: LeaseDTO): void {
		selectedLease = lease;
		leaseDocsSheetOpen = true;
	}
</script>

<PageHeader
	title="Baux actifs & Contrats"
	subtitle="Contrats de location (Loi 89, Meublé, Mobilité, Commercial 3/6/9) et coffre-fort documentaire R2"
>
	{#snippet actions()}
		<Button size="sm" variant="outline" href={resolve('/(app)/leases/tenants')}>
			<IconUsers data-icon="inline-start" />
			Locataires ({data.tenants.length})
		</Button>
		<Button size="sm" onclick={openCreateLease}>
			<IconPlus data-icon="inline-start" />
			Nouveau bail
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

<LeasesTableCard
	leases={data.leases}
	entities={data.entities}
	properties={data.properties}
	tenants={data.tenants}
	onCreateLease={openCreateLease}
	onOpenDocuments={openLeaseDocuments}
/>

<CreateLeaseDialog
	bind:open={createLeaseOpen}
	entities={data.entities}
	properties={data.properties}
	tenants={data.tenants}
	preselectedEntityId={defaultEntityId}
	{preselectedLotId}
/>

<LeaseDocumentsSheet
	bind:open={leaseDocsSheetOpen}
	{selectedLease}
	entityName={selectedLease
		? (entityMap.get(selectedLease.legal_entity_id)?.name ?? 'Bailleur')
		: ''}
	tenantName={selectedLease
		? (tenantMap.get(selectedLease.tenant_id)?.full_name ?? 'Locataire')
		: ''}
	propertyLotLabel={selectedLease
		? lotInfoMap.get(selectedLease.lot_id)
			? `${lotInfoMap.get(selectedLease.lot_id)?.property.label} — ${lotInfoMap.get(selectedLease.lot_id)?.lot.label}`
			: selectedLease.lot_id
		: ''}
	templates={data.templates}
	documents={selectedLease ? (data.documentsByLeaseId[selectedLease.id] ?? []) : []}
/>
