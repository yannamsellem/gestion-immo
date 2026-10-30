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
	import type { RentCallDTO } from '#lib/modules/lease/types';
	import type { NegativePositionDTO } from '#lib/modules/payment/types';
	import type { PageProps } from './$types';
	import ImportPaymentsCsvDialog from './(components)/ImportPaymentsCsvDialog.svelte';
	import RentActionSheet from './(components)/RentActionSheet.svelte';
	import RentCallsTableCard, {
		type RentStatusFilter,
		type UnifiedRentRow
	} from './(components)/RentCallsTableCard.svelte';
	import RentsMetricsGrid from './(components)/RentsMetricsGrid.svelte';

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

	const unifiedRows = $derived.by((): UnifiedRentRow[] => {
		const entityMap = new Map(data.entities.map((entity) => [entity.id, entity]));
		const tenantMap = new Map(data.tenants.map((tenant) => [tenant.id, tenant]));
		const lotLabelMap = new Map<string, string>(
			Object.values(data.propertiesByEntity).flatMap((propList) =>
				propList.flatMap((prop) =>
					(prop.lots ?? []).map((lot) => [lot.id, `${prop.label} — ${lot.label}`] as const)
				)
			)
		);
		const rentCallByLeaseId = new Map<string, RentCallDTO>(
			Object.values(data.rentCallsByEntity).flatMap((callList) =>
				callList
					.filter((rc) => rc.period_month === workspaceStore.selectedPeriodMonth)
					.map((rc) => [rc.lease_id, rc] as const)
			)
		);
		const negativeByLeaseId = new Map<string, NegativePositionDTO>(
			Object.values(data.negativePositionsByEntity).flatMap((negList) =>
				negList.map((neg) => [neg.lease_id, neg] as const)
			)
		);

		return data.leases
			.filter(
				(lease) =>
					workspaceStore.selectedEntityId === 'ALL' ||
					lease.legal_entity_id === workspaceStore.selectedEntityId
			)
			.map((lease) => {
				const entity = entityMap.get(lease.legal_entity_id) ?? data.entities[0];
				const rentCall = rentCallByLeaseId.get(lease.id) ?? null;
				const negativePosition = negativeByLeaseId.get(lease.id) ?? null;
				const vatCents = lease.subject_to_vat
					? Math.trunc((lease.rent_net_cents * lease.vat_rate_basis_points) / 10000)
					: 0;
				const expectedGrossCents = rentCall
					? rentCall.total_due_cents
					: lease.rent_net_cents + vatCents + lease.charges_cents;
				const remainingDueCents = rentCall ? rentCall.remaining_due_cents : expectedGrossCents;

				let statusCategory: 'PAID' | 'OVERDUE' | 'TO_COLLECT' = 'TO_COLLECT';
				let statusBadgeLabel = 'Appel à générer';
				let statusBadgeVariant: 'default' | 'secondary' | 'destructive' | 'outline' = 'outline';

				if (negativePosition && negativePosition.total_unpaid_cents > 0) {
					statusCategory = 'OVERDUE';
					statusBadgeLabel = `Impayé (${negativePosition.unpaid_months.length} m.)`;
					statusBadgeVariant = 'destructive';
				} else if (
					rentCall?.status === 'PAID' ||
					(rentCall && rentCall.remaining_due_cents === 0)
				) {
					statusCategory = 'PAID';
					statusBadgeLabel = 'Soldé';
					statusBadgeVariant = 'secondary';
				} else if (rentCall?.status === 'PARTIAL') {
					statusBadgeLabel = 'Partiel';
					statusBadgeVariant = 'default';
				} else if (rentCall) {
					statusBadgeLabel = 'En attente';
				}

				return {
					lease,
					entity,
					propertyAndLotLabel: lotLabelMap.get(lease.lot_id) ?? `Lot ${lease.lot_id.slice(0, 8)}`,
					tenantName: tenantMap.get(lease.tenant_id)?.full_name ?? 'Locataire',
					rentCall,
					negativePosition,
					expectedGrossCents,
					remainingDueCents,
					statusCategory,
					statusBadgeLabel,
					statusBadgeVariant
				};
			});
	});

	const expectedTotalCents = $derived(
		unifiedRows.reduce((sum, row) => sum + row.expectedGrossCents, 0)
	);
	const remainingTotalCents = $derived(
		unifiedRows.reduce((sum, row) => sum + row.remainingDueCents, 0)
	);
	const collectedTotalCents = $derived(Math.max(0, expectedTotalCents - remainingTotalCents));
	const overdueCount = $derived(
		unifiedRows.filter((row) => row.statusCategory === 'OVERDUE').length
	);
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

<RentsMetricsGrid {expectedTotalCents} {collectedTotalCents} {remainingTotalCents} {overdueCount} />

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
