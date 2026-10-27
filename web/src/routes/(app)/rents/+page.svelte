<script lang="ts">
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconBuildingBank from '@tabler/icons-svelte/icons/building-bank';
	import IconCalendarPlus from '@tabler/icons-svelte/icons/calendar-plus';
	import IconCash from '@tabler/icons-svelte/icons/cash';
	import IconFileUpload from '@tabler/icons-svelte/icons/file-upload';
	import IconReceiptEuro from '@tabler/icons-svelte/icons/receipt-euro';
	import { toast } from 'svelte-sonner';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import RentActionSheet from '#lib/components/rents/RentActionSheet.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import * as ToggleGroup from '#lib/components/ui/toggle-group/index.js';
	import { ApiError } from '#lib/core/api/api-error';
	import { formatIsoDateFr, formatPeriodMonthFr } from '#lib/core/formatters/date';
	import { formatCentsToEuro, parseEuroToCents } from '#lib/core/formatters/money';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import type { LeaseDTO, RentCallDTO, TenantDTO } from '#lib/modules/lease/types';
	import type { CashJournalEntryDTO, NegativePositionDTO } from '#lib/modules/payment/types';
	import type { LegalEntityDTO, PropertyDTO } from '#lib/modules/portfolio/types';
	import { apiRegistry } from '#lib/modules/registry';

	type RentStatusFilter = 'ALL' | 'TO_COLLECT' | 'OVERDUE' | 'PAID';

	interface UnifiedRentRow {
		lease: LeaseDTO;
		entity: LegalEntityDTO;
		propertyAndLotLabel: string;
		tenantName: string;
		rentCall: RentCallDTO | null;
		negativePosition: NegativePositionDTO | null;
		expectedGrossCents: number;
		remainingDueCents: number;
		statusCategory: 'PAID' | 'OVERDUE' | 'TO_COLLECT';
		statusBadgeLabel: string;
		statusBadgeVariant: 'default' | 'secondary' | 'destructive' | 'outline';
	}

	let statusFilter = $state<RentStatusFilter>('ALL');
	let isLoading = $state(true);
	let errorMessage = $state<string | null>(null);
	let reloadCounter = $state(0);

	let entities = $state<LegalEntityDTO[]>([]);
	let propertiesByEntity = $state<Record<string, PropertyDTO[]>>({});
	let leases = $state<LeaseDTO[]>([]);
	let tenants = $state<TenantDTO[]>([]);
	let rentCallsByEntity = $state<Record<string, RentCallDTO[]>>({});
	let negativePositionsByEntity = $state<Record<string, NegativePositionDTO[]>>({});
	let cashJournalByEntity = $state<Record<string, CashJournalEntryDTO[]>>({});

	let isGeneratingCalls = $state(false);

	// Sheet state
	let sheetOpen = $state(false);
	let selectedLeaseId = $state<string | null>(null);
	let sheetInitialAction = $state<'pay' | 'remind' | 'generate-calls' | null>(null);
	let autoOpenedQueryKey = $state<string | null>(null);

	// CSV Import Dialog state
	let csvDialogOpen = $state(false);
	let csvTargetEntityId = $state('');
	let csvRawContent = $state('');
	let isImportingCsv = $state(false);

	// Cash Deposit Dialog state
	let depositDialogOpen = $state(false);
	let depositEntityId = $state('');
	let depositAmountEuro = $state('');
	let depositLabel = $state('Remise espèces en banque (530 → 512)');
	let depositDate = $state(new Date().toISOString().slice(0, 10));
	let isSubmittingDeposit = $state(false);

	// Sync URL entityId parameter into workspaceStore
	$effect(() => {
		const urlEntityId = page.url.searchParams.get('entityId');
		if (urlEntityId && urlEntityId !== workspaceStore.selectedEntityId) {
			workspaceStore.setEntityId(urlEntityId);
		}
	});

	// Load data for the selected workspace & period
	$effect(() => {
		const selectedEntityId = workspaceStore.selectedEntityId;
		const periodMonth = workspaceStore.selectedPeriodMonth;
		const trigger = reloadCounter;
		void trigger;

		let cancelled = false;
		isLoading = true;
		errorMessage = null;

		void (async () => {
			try {
				const allEntities = await apiRegistry.portfolio.listEntities();
				if (cancelled) return;
				entities = allEntities;
				workspaceStore.setEntities(allEntities);

				const targetEntities =
					selectedEntityId === 'ALL'
						? allEntities
						: allEntities.filter((item) => item.id === selectedEntityId);

				if (targetEntities.length > 0) {
					if (!csvTargetEntityId) csvTargetEntityId = targetEntities[0].id;
					if (!depositEntityId) depositEntityId = targetEntities[0].id;
				}

				const [loadedLeases, loadedTenants] = await Promise.all([
					apiRegistry.lease.listLeases(selectedEntityId),
					apiRegistry.lease.listTenants(true)
				]);
				if (cancelled) return;

				const nextProps: Record<string, PropertyDTO[]> = {};
				const nextCalls: Record<string, RentCallDTO[]> = {};
				const nextNegatives: Record<string, NegativePositionDTO[]> = {};
				const nextCash: Record<string, CashJournalEntryDTO[]> = {};

				await Promise.all(
					targetEntities.map(async (entity) => {
						const [props, calls, negatives, cashEntries] = await Promise.all([
							apiRegistry.portfolio.listProperties(entity.id),
							apiRegistry.lease.listEntityRentCalls(entity.id, periodMonth),
							apiRegistry.payment.listNegativePositions(entity.id).catch(() => []),
							apiRegistry.payment.listCashJournal(entity.id).catch(() => [])
						]);
						nextProps[entity.id] = props;
						nextCalls[entity.id] = calls;
						nextNegatives[entity.id] = negatives;
						nextCash[entity.id] = cashEntries;
					})
				);
				if (cancelled) return;

				leases = loadedLeases.filter((lease) => lease.active);
				tenants = loadedTenants;
				propertiesByEntity = nextProps;
				rentCallsByEntity = nextCalls;
				negativePositionsByEntity = nextNegatives;
				cashJournalByEntity = nextCash;
			} catch (err) {
				if (cancelled) return;
				errorMessage =
					err instanceof ApiError
						? err.message
						: 'Impossible de charger les appels de loyers et encaissements.';
			} finally {
				if (!cancelled) {
					isLoading = false;
				}
			}
		})();

		return () => {
			cancelled = true;
		};
	});

	const unifiedRows = $derived.by((): UnifiedRentRow[] => {
		const entityMap = new Map(entities.map((entity) => [entity.id, entity]));
		const tenantMap = new Map(tenants.map((tenant) => [tenant.id, tenant]));
		const lotLabelMap = new Map<string, string>();

		for (const propList of Object.values(propertiesByEntity)) {
			for (const prop of propList) {
				for (const lot of prop.lots ?? []) {
					lotLabelMap.set(lot.id, `${prop.label} — ${lot.label}`);
				}
			}
		}

		const rentCallByLeaseId = new Map<string, RentCallDTO>();
		for (const callList of Object.values(rentCallsByEntity)) {
			for (const rc of callList) {
				if (rc.period_month === workspaceStore.selectedPeriodMonth) {
					rentCallByLeaseId.set(rc.lease_id, rc);
				}
			}
		}

		const negativeByLeaseId = new Map<string, NegativePositionDTO>();
		for (const negList of Object.values(negativePositionsByEntity)) {
			for (const neg of negList) {
				negativeByLeaseId.set(neg.lease_id, neg);
			}
		}

		return leases
			.filter(
				(lease) =>
					workspaceStore.selectedEntityId === 'ALL' ||
					lease.legal_entity_id === workspaceStore.selectedEntityId
			)
			.map((lease) => {
				const entity = entityMap.get(lease.legal_entity_id) ?? {
					id: lease.legal_entity_id,
					name: 'Entité',
					entity_type: 'SCI_IR',
					siret: '',
					naf_code: '',
					address: '',
					share_capital_cents: 0,
					total_shares: 0,
					shareholders: [],
					created_at: '',
					updated_at: ''
				};
				const tenant = tenantMap.get(lease.tenant_id);
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
					statusCategory = 'TO_COLLECT';
					statusBadgeLabel = 'Partiel';
					statusBadgeVariant = 'default';
				} else if (rentCall) {
					statusCategory = 'TO_COLLECT';
					statusBadgeLabel = 'En attente';
					statusBadgeVariant = 'outline';
				}

				return {
					lease,
					entity,
					propertyAndLotLabel: lotLabelMap.get(lease.lot_id) ?? `Lot ${lease.lot_id.slice(0, 8)}`,
					tenantName: tenant?.full_name ?? 'Locataire',
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

	const filteredRows = $derived(
		unifiedRows.filter((row) => {
			if (statusFilter === 'ALL') return true;
			return row.statusCategory === statusFilter;
		})
	);

	const selectedRow = $derived(unifiedRows.find((row) => row.lease.id === selectedLeaseId) ?? null);

	const allCashEntries = $derived(
		Object.values(cashJournalByEntity)
			.flat()
			.sort((a, b) => b.entry_date.localeCompare(a.entry_date))
	);

	// Auto-open contextual Sheet when arriving from Cockpit CTA query params
	$effect(() => {
		if (isLoading || unifiedRows.length === 0) return;

		const urlLeaseId = page.url.searchParams.get('leaseId');
		const urlRentCallId = page.url.searchParams.get('rentCallId');
		const urlAction = page.url.searchParams.get('action') as
			'pay' | 'remind' | 'generate-calls' | null;

		const queryKey = `${urlLeaseId ?? ''}|${urlRentCallId ?? ''}|${urlAction ?? ''}`;
		if (!urlLeaseId && !urlRentCallId) return;
		if (autoOpenedQueryKey === queryKey) return;

		const matchingRow = unifiedRows.find(
			(row) =>
				(urlLeaseId && row.lease.id === urlLeaseId) ||
				(urlRentCallId && row.rentCall?.id === urlRentCallId)
		);

		if (matchingRow) {
			selectedLeaseId = matchingRow.lease.id;
			sheetInitialAction = urlAction;
			sheetOpen = true;
			autoOpenedQueryKey = queryKey;
		}
	});

	function openRowSheet(row: UnifiedRentRow, action: 'pay' | 'remind' | null = null) {
		selectedLeaseId = row.lease.id;
		sheetInitialAction = action;
		sheetOpen = true;
	}

	async function handleGenerateMonthlyCalls() {
		const targetEntities =
			workspaceStore.selectedEntityId === 'ALL'
				? entities
				: entities.filter((item) => item.id === workspaceStore.selectedEntityId);

		if (targetEntities.length === 0) {
			toast.error('Aucune entité sélectionnée.');
			return;
		}

		isGeneratingCalls = true;
		try {
			let generatedCount = 0;
			for (const entity of targetEntities) {
				const calls = await apiRegistry.lease.generateRentCalls(
					entity.id,
					workspaceStore.selectedPeriodMonth
				);
				generatedCount += calls.length;
			}
			toast.success(
				`${generatedCount} appel(s) de loyer généré(s) pour ${formatPeriodMonthFr(workspaceStore.selectedPeriodMonth)}.`
			);
			reloadCounter += 1;
		} catch (err) {
			toast.error(
				err instanceof ApiError ? err.message : 'Erreur lors de la génération des appels de loyers.'
			);
		} finally {
			isGeneratingCalls = false;
		}
	}

	async function handleCsvFileChange(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		csvRawContent = await file.text();
	}

	async function handleImportCsv(event: SubmitEvent) {
		event.preventDefault();
		if (!csvTargetEntityId || !csvRawContent.trim()) {
			toast.error('Veuillez choisir une entité et fournir un contenu CSV.');
			return;
		}

		isImportingCsv = true;
		try {
			const result = await apiRegistry.payment.importPaymentsCsv(csvTargetEntityId, csvRawContent);
			toast.success(
				`${result.imported_count} virement(s) importé(s) (${formatCentsToEuro(result.total_imported_cents)}).`
			);
			csvDialogOpen = false;
			csvRawContent = '';
			reloadCounter += 1;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Échec de l’import bancaire CSV.');
		} finally {
			isImportingCsv = false;
		}
	}

	async function handleCashDeposit(event: SubmitEvent) {
		event.preventDefault();
		if (!depositEntityId) return;

		let amountCents = 0;
		try {
			amountCents = parseEuroToCents(depositAmountEuro);
		} catch {
			toast.error('Montant de remise en banque invalide.');
			return;
		}

		if (amountCents <= 0) {
			toast.error('Le montant déposé doit être supérieur à 0 €.');
			return;
		}

		isSubmittingDeposit = true;
		try {
			await apiRegistry.payment.recordCashDeposit({
				legal_entity_id: depositEntityId,
				amount_cents: amountCents,
				label: depositLabel.trim() || 'Remise espèces en banque (530 → 512)',
				entry_date: depositDate ? new Date(depositDate).toISOString() : undefined
			});
			toast.success('Remise d’espèces en banque enregistrée (écriture 530 → 512).');
			depositDialogOpen = false;
			depositAmountEuro = '';
			reloadCounter += 1;
		} catch (err) {
			toast.error(
				err instanceof ApiError ? err.message : 'Impossible d’enregistrer la remise d’espèces.'
			);
		} finally {
			isSubmittingDeposit = false;
		}
	}
</script>

<PageHeader
	title="Loyers, Appels & Relances"
	subtitle="Tableau mensuel unifié des appels d'échéance, encaissements FIFO, livre de caisse PCG 530 et dossiers GLI"
>
	{#snippet actions()}
		<Input
			type="month"
			value={workspaceStore.selectedPeriodMonth}
			onchange={(e) => {
				const val = (e.currentTarget as HTMLInputElement).value;
				if (val) workspaceStore.setPeriodMonth(val);
			}}
			class="h-8 w-40"
			aria-label="Mois d'échéance"
		/>

		<Button
			variant="secondary"
			size="sm"
			onclick={handleGenerateMonthlyCalls}
			disabled={isGeneratingCalls || entities.length === 0}
		>
			{#if isGeneratingCalls}
				<Spinner data-icon="inline-start" />
			{:else}
				<IconCalendarPlus data-icon="inline-start" />
			{/if}
			Générer appels du mois
		</Button>

		<Button variant="outline" size="sm" onclick={() => (csvDialogOpen = true)}>
			<IconFileUpload data-icon="inline-start" />
			Import CSV bancaire
		</Button>

		<Button variant="outline" size="sm" onclick={() => (depositDialogOpen = true)}>
			<IconBuildingBank data-icon="inline-start" />
			Remise espèces (530 → 512)
		</Button>
	{/snippet}
</PageHeader>

{#if errorMessage}
	<Alert.Root variant="destructive">
		<IconAlertTriangle class="size-4" />
		<Alert.Title>Erreur</Alert.Title>
		<Alert.Description>{errorMessage}</Alert.Description>
	</Alert.Root>
{/if}

<div class="flex flex-wrap items-center justify-between gap-3">
	<ToggleGroup.Root
		type="single"
		variant="outline"
		size="sm"
		value={statusFilter}
		onValueChange={(val) => {
			if (val) statusFilter = val as RentStatusFilter;
		}}
	>
		<ToggleGroup.Item value="ALL">Tous ({unifiedRows.length})</ToggleGroup.Item>
		<ToggleGroup.Item value="TO_COLLECT">
			À encaisser ({unifiedRows.filter((r) => r.statusCategory === 'TO_COLLECT').length})
		</ToggleGroup.Item>
		<ToggleGroup.Item value="OVERDUE">
			Impayés ({unifiedRows.filter((r) => r.statusCategory === 'OVERDUE').length})
		</ToggleGroup.Item>
		<ToggleGroup.Item value="PAID">
			Payés ({unifiedRows.filter((r) => r.statusCategory === 'PAID').length})
		</ToggleGroup.Item>
	</ToggleGroup.Root>
</div>

{#if isLoading}
	<div class="flex flex-col gap-3">
		<Skeleton class="h-12 w-full" />
		<Skeleton class="h-48 w-full" />
	</div>
{:else if filteredRows.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconReceiptEuro class="size-5" />
					</Empty.Media>
					<Empty.Title>Aucun bail actif pour ce filtre</Empty.Title>
					<Empty.Description>
						Créez un bail dans la section Baux & Locataires ou modifiez le filtre de statut.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button href="/leases?action=create" variant="outline">Aller aux baux →</Button>
				</Empty.Content>
			</Empty.Root>
		</Card.Content>
	</Card.Root>
{:else}
	<Card.Root>
		<Card.Content class="p-0">
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head class="pl-4">Entité & Lot</Table.Head>
						<Table.Head>Locataire</Table.Head>
						<Table.Head class="text-right">Appelé TTC / CC</Table.Head>
						<Table.Head class="text-right">Reste à payer</Table.Head>
						<Table.Head>Statut</Table.Head>
						<Table.Head class="pr-4 text-right">Actions</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each filteredRows as row (row.lease.id)}
						<Table.Row class="cursor-pointer" onclick={() => openRowSheet(row)}>
							<Table.Cell class="pl-4">
								<div class="flex flex-col gap-0.5">
									<span class="font-medium">{row.propertyAndLotLabel}</span>
									<span class="text-[11px] text-muted-foreground">{row.entity.name}</span>
								</div>
							</Table.Cell>

							<Table.Cell>
								<span class="font-medium">{row.tenantName}</span>
							</Table.Cell>

							<Table.Cell class="text-right font-mono tabular-nums">
								{formatCentsToEuro(row.expectedGrossCents)}
							</Table.Cell>

							<Table.Cell class="text-right font-mono tabular-nums">
								{#if row.remainingDueCents > 0}
									<span class="font-semibold text-destructive">
										{formatCentsToEuro(row.remainingDueCents)}
									</span>
								{:else}
									<span class="text-muted-foreground">0,00 €</span>
								{/if}
							</Table.Cell>

							<Table.Cell>
								<Badge variant={row.statusBadgeVariant}>{row.statusBadgeLabel}</Badge>
							</Table.Cell>

							<Table.Cell class="pr-4 text-right">
								<div class="flex items-center justify-end gap-1.5">
									{#if row.negativePosition && row.negativePosition.total_unpaid_cents > 0}
										<Button
											size="sm"
											variant="destructive"
											onclick={(e: MouseEvent) => {
												e.stopPropagation();
												openRowSheet(row, 'remind');
											}}
										>
											Relancer / GLI →
										</Button>
									{/if}
									<Button
										size="sm"
										variant={row.remainingDueCents > 0 ? 'default' : 'outline'}
										onclick={(e: MouseEvent) => {
											e.stopPropagation();
											openRowSheet(row, 'pay');
										}}
									>
										{row.remainingDueCents > 0 ? 'Saisir loyer →' : 'Quittance / Détail →'}
									</Button>
								</div>
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</Card.Content>
	</Card.Root>
{/if}

<!-- PCG 530 Cash Journal Card -->
<Card.Root>
	<Card.Header
		class="flex flex-col justify-between gap-2 border-b pb-3 sm:flex-row sm:items-center"
	>
		<div>
			<Card.Title class="flex items-center gap-1.5 text-sm font-semibold">
				<IconCash class="size-4" />
				Livre de caisse espèces (Compte PCG 530)
			</Card.Title>
			<Card.Description>
				Traçabilité chronologique des encaissements en espèces (411 → 530) et remises en banque (530
				→ 512)
			</Card.Description>
		</div>
		<Badge variant="outline">{allCashEntries.length} écriture(s)</Badge>
	</Card.Header>
	<Card.Content class="p-0">
		{#if allCashEntries.length === 0}
			<p class="px-4 py-6 text-xs text-muted-foreground">
				Aucune opération en espèces enregistrée sur ce périmètre.
			</p>
		{:else}
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head class="pl-4">Date</Table.Head>
						<Table.Head>Type</Table.Head>
						<Table.Head>Libellé & Reçu</Table.Head>
						<Table.Head>Comptes PCG</Table.Head>
						<Table.Head class="text-right">Débit 530</Table.Head>
						<Table.Head class="pr-4 text-right">Crédit 530</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each allCashEntries as entry (entry.id)}
						<Table.Row>
							<Table.Cell class="pl-4 text-xs">{formatIsoDateFr(entry.entry_date)}</Table.Cell>
							<Table.Cell>
								<Badge variant={entry.entry_type === 'CASH_RECEIPT' ? 'secondary' : 'outline'}>
									{entry.entry_type === 'CASH_RECEIPT' ? 'Reçu espèces' : 'Remise banque'}
								</Badge>
							</Table.Cell>
							<Table.Cell class="text-xs">
								<span class="font-medium">{entry.label}</span>
								{#if entry.receipt_number}
									<span class="ml-1 font-mono text-muted-foreground">
										({entry.receipt_number})
									</span>
								{/if}
							</Table.Cell>
							<Table.Cell class="font-mono text-xs">
								{entry.account_code} / {entry.counterpart_account_code}
							</Table.Cell>
							<Table.Cell class="text-right font-mono tabular-nums">
								{entry.debit_cents > 0 ? formatCentsToEuro(entry.debit_cents) : '—'}
							</Table.Cell>
							<Table.Cell class="pr-4 text-right font-mono tabular-nums">
								{entry.credit_cents > 0 ? formatCentsToEuro(entry.credit_cents) : '—'}
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		{/if}
	</Card.Content>
</Card.Root>

<!-- Contextual Action Sheet -->
<RentActionSheet
	bind:open={sheetOpen}
	lease={selectedRow?.lease ?? null}
	rentCall={selectedRow?.rentCall ?? null}
	negativePosition={selectedRow?.negativePosition ?? null}
	tenantName={selectedRow?.tenantName ?? ''}
	entityName={selectedRow?.entity.name ?? ''}
	propertyAndLotLabel={selectedRow?.propertyAndLotLabel ?? ''}
	periodMonth={workspaceStore.selectedPeriodMonth}
	initialAction={sheetInitialAction}
	onChanged={() => {
		reloadCounter += 1;
	}}
/>

<!-- CSV Import Dialog -->
<Dialog.Root bind:open={csvDialogOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Importer un relevé bancaire CSV</Dialog.Title>
			<Dialog.Description>
				Importe et impute automatiquement les virements de loyers (colonnes :
				<code>lease_id,amount_cents,payment_date,label,external_ref</code>).
			</Dialog.Description>
		</Dialog.Header>

		<form onsubmit={handleImportCsv} class="flex flex-col gap-4">
			<Field.FieldGroup>
				<Field.Field>
					<Field.Label>Entité juridique</Field.Label>
					<Select.Root
						type="single"
						value={csvTargetEntityId}
						onValueChange={(val) => {
							if (val) csvTargetEntityId = val;
						}}
					>
						<Select.Trigger class="w-full">
							{entities.find((e) => e.id === csvTargetEntityId)?.name ?? 'Sélectionner une entité'}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each entities as entity (entity.id)}
									<Select.Item value={entity.id} label={entity.name}>{entity.name}</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<Field.Field>
					<Field.Label for="csv-file">Fichier CSV</Field.Label>
					<Input id="csv-file" type="file" accept=".csv,text/csv" onchange={handleCsvFileChange} />
				</Field.Field>

				<Field.Field>
					<Field.Label for="csv-raw">Contenu CSV brut</Field.Label>
					<textarea
						id="csv-raw"
						rows="4"
						class="w-full rounded-md border border-input bg-background p-2 font-mono text-xs placeholder:text-muted-foreground"
						placeholder="lease_id,amount_cents,payment_date,label,external_ref"
						bind:value={csvRawContent}
						required></textarea>
				</Field.Field>
			</Field.FieldGroup>

			<Dialog.Footer>
				<Button variant="outline" type="button" onclick={() => (csvDialogOpen = false)}>
					Annuler
				</Button>
				<Button type="submit" disabled={isImportingCsv}>
					{#if isImportingCsv}
						<Spinner data-icon="inline-start" />
					{/if}
					Importer et imputer
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>

<!-- Cash Bank Deposit Dialog (530 -> 512) -->
<Dialog.Root bind:open={depositDialogOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Remise d'espèces en banque (PCG 530 → 512)</Dialog.Title>
			<Dialog.Description>
				Crédite le compte Caisse (530) et débite le compte Banque (512) lors du dépôt des espèces.
			</Dialog.Description>
		</Dialog.Header>

		<form onsubmit={handleCashDeposit} class="flex flex-col gap-4">
			<Field.FieldGroup>
				<Field.Field>
					<Field.Label>Entité juridique</Field.Label>
					<Select.Root
						type="single"
						value={depositEntityId}
						onValueChange={(val) => {
							if (val) depositEntityId = val;
						}}
					>
						<Select.Trigger class="w-full">
							{entities.find((e) => e.id === depositEntityId)?.name ?? 'Sélectionner une entité'}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each entities as entity (entity.id)}
									<Select.Item value={entity.id} label={entity.name}>{entity.name}</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="deposit-amount">Montant déposé (€)</Field.Label>
						<Input
							id="deposit-amount"
							type="text"
							inputmode="decimal"
							placeholder="500,00"
							bind:value={depositAmountEuro}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="deposit-date">Date du dépôt</Field.Label>
						<Input id="deposit-date" type="date" bind:value={depositDate} required />
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="deposit-label">Libellé du bordereau</Field.Label>
					<Input id="deposit-label" type="text" bind:value={depositLabel} required />
				</Field.Field>
			</Field.FieldGroup>

			<Dialog.Footer>
				<Button variant="outline" type="button" onclick={() => (depositDialogOpen = false)}>
					Annuler
				</Button>
				<Button type="submit" disabled={isSubmittingDeposit}>
					{#if isSubmittingDeposit}
						<Spinner data-icon="inline-start" />
					{/if}
					Enregistrer la remise
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
