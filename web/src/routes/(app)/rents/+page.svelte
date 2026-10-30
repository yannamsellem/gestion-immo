<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconCalendarPlus from '@tabler/icons-svelte/icons/calendar-plus';
	import IconCash from '@tabler/icons-svelte/icons/cash';
	import IconFileUpload from '@tabler/icons-svelte/icons/file-upload';
	import { toast } from 'svelte-sonner';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import ImportPaymentsCsvDialog from './(components)/ImportPaymentsCsvDialog.svelte';
	import RentActionSheet from './(components)/RentActionSheet.svelte';
	import RentCallsTableCard from './(components)/RentCallsTableCard.svelte';
	import RentsMetricsGrid from './(components)/RentsMetricsGrid.svelte';
	import {
		type RentStatusFilter,
		type UnifiedRentRow,
		buildUnifiedRentRows,
		summarizeRentRows
	} from './(components)/rent-rows';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	let statusFilter = $state<RentStatusFilter>('ALL');
	let isGeneratingCalls = $state(false);
	let csvDialogOpen = $state(false);
	let sheetOpen = $state(false);
	let selectedLeaseId = $state<string | null>(null);
	let sheetInitialAction = $state<'pay' | 'remind' | 'generate-calls' | null>(null);
	let autoOpenedQueryKey = $state<string | null>(null);

	const defaultEntityId = $derived(
		workspaceStore.selectedEntityId !== 'ALL' ? workspaceStore.selectedEntityId : ''
	);

	const unifiedRows = $derived(
		buildUnifiedRentRows({
			entities: data.entities,
			leases: data.leases,
			tenants: data.tenants,
			propertiesByEntity: data.propertiesByEntity,
			rentCallsByEntity: data.rentCallsByEntity,
			negativePositionsByEntity: data.negativePositionsByEntity,
			selectedEntityId: workspaceStore.selectedEntityId,
			selectedPeriodMonth: workspaceStore.selectedPeriodMonth
		})
	);

	const summary = $derived(summarizeRentRows(unifiedRows));
	const selectedRow = $derived(unifiedRows.find((row) => row.lease.id === selectedLeaseId) ?? null);

	$effect(() => {
		if (unifiedRows.length === 0) return;
		const urlLeaseId = page.url.searchParams.get('leaseId');
		const urlRentCallId = page.url.searchParams.get('rentCallId');
		const urlAction = page.url.searchParams.get('action') as
			'pay' | 'remind' | 'generate-calls' | null;

		const queryKey = `${urlLeaseId ?? ''}|${urlRentCallId ?? ''}|${urlAction ?? ''}`;
		if ((!urlLeaseId && !urlRentCallId) || autoOpenedQueryKey === queryKey) return;

		const match = unifiedRows.find(
			(row) =>
				(urlLeaseId && row.lease.id === urlLeaseId) ||
				(urlRentCallId && row.rentCall?.id === urlRentCallId)
		);
		if (match) {
			selectedLeaseId = match.lease.id;
			sheetInitialAction = urlAction;
			sheetOpen = true;
			autoOpenedQueryKey = queryKey;
		}
	});

	function openRowSheet(row: UnifiedRentRow, action: 'pay' | 'remind' | null = null): void {
		selectedLeaseId = row.lease.id;
		sheetInitialAction = action;
		sheetOpen = true;
	}
</script>

<PageHeader
	title="Appels de loyers & Encaissements"
	subtitle="Échéancier mensuel unifié, imputation FIFO des règlements, avis d’échéance, quittances et dossiers sinistre GLI"
>
	{#snippet actions()}
		<Input
			type="month"
			value={workspaceStore.selectedPeriodMonth}
			onchange={(event) => {
				const next = (event.currentTarget as HTMLInputElement).value;
				if (next) workspaceStore.setPeriodMonth(next);
			}}
			class="h-8 w-40"
			aria-label="Mois d’échéance"
		/>

		<form
			method="POST"
			action="?/generateCalls"
			use:enhance={() => {
				isGeneratingCalls = true;
				return async ({ result }) => {
					isGeneratingCalls = false;
					if (result.type === 'success') {
						const payload = result.data as { successMessage?: string } | undefined;
						toast.success(payload?.successMessage ?? 'Appels de loyers générés.');
						await invalidateAll();
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						toast.error(payload?.actionError ?? 'Erreur lors de la génération des appels.');
						return;
					}
					if (result.type === 'error') {
						toast.error(result.error?.message ?? 'Erreur serveur lors de la génération.');
					}
				};
			}}
		>
			<input type="hidden" name="entity_id" value={workspaceStore.selectedEntityId} />
			<input type="hidden" name="period_month" value={workspaceStore.selectedPeriodMonth} />
			<Button
				type="submit"
				variant="secondary"
				size="sm"
				disabled={isGeneratingCalls || data.entities.length === 0}
			>
				{#if isGeneratingCalls}
					<Spinner data-icon="inline-start" />
				{:else}
					<IconCalendarPlus data-icon="inline-start" />
				{/if}
				Générer appels du mois
			</Button>
		</form>

		<Button variant="outline" size="sm" onclick={() => (csvDialogOpen = true)}>
			<IconFileUpload data-icon="inline-start" />
			Import CSV bancaire
		</Button>

		<Button variant="outline" size="sm" href={resolve('/(app)/rents/cash-journal')}>
			<IconCash data-icon="inline-start" />
			Livre de caisse (530)
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

<RentsMetricsGrid
	expectedTotalCents={summary.expectedTotalCents}
	collectedTotalCents={summary.collectedTotalCents}
	remainingTotalCents={summary.remainingTotalCents}
	overdueCount={summary.overdueCount}
/>

<RentCallsTableCard rows={unifiedRows} bind:statusFilter onOpenRow={openRowSheet} />

<RentActionSheet
	bind:open={sheetOpen}
	lease={selectedRow?.lease ?? null}
	rentCall={selectedRow?.rentCall ?? null}
	negativePosition={selectedRow?.negativePosition ?? null}
	tenantName={selectedRow?.tenantName ?? ''}
	entityName={selectedRow?.entity.name ?? ''}
	propertyAndLotLabel={selectedRow?.propertyAndLotLabel ?? ''}
	periodMonth={workspaceStore.selectedPeriodMonth}
	templates={data.templates}
	leaseDocuments={selectedRow ? (data.documentsByLeaseId[selectedRow.lease.id] ?? []) : []}
	initialAction={sheetInitialAction}
/>

<ImportPaymentsCsvDialog bind:open={csvDialogOpen} entities={data.entities} {defaultEntityId} />
