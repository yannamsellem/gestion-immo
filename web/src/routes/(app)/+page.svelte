<script lang="ts">
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconBuildingBank from '@tabler/icons-svelte/icons/building-bank';
	import IconCalendarEvent from '@tabler/icons-svelte/icons/calendar-event';
	import IconCheck from '@tabler/icons-svelte/icons/check';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import IconRefresh from '@tabler/icons-svelte/icons/refresh';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { ApiError } from '#lib/core/api/api-error';
	import { formatPeriodMonthFr } from '#lib/core/formatters/date';
	import { formatCentsToEuro } from '#lib/core/formatters/money';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import { buildCockpitEntityGroups } from '#lib/modules/cockpit/cockpit-builder';
	import type { EntityCockpitGroup } from '#lib/modules/cockpit/types';
	import type { RentCallDTO } from '#lib/modules/lease/types';
	import type { NegativePositionDTO } from '#lib/modules/payment/types';
	import type { PropertyDTO } from '#lib/modules/portfolio/types';
	import { apiRegistry } from '#lib/modules/registry';

	let isLoading = $state(true);
	let errorMessage = $state<string | null>(null);
	let groups = $state<EntityCockpitGroup[]>([]);
	let totalOverdueLeasesCount = $state(0);
	let totalOpenIncidentsCount = $state(0);
	let reloadTrigger = $state(0);

	const ENTITY_TYPE_LABELS: Record<string, string> = {
		SCI_IR: 'SCI à l’IR',
		SCI_IS: 'SCI à l’IS',
		SARL_FAMILLE: 'SARL de famille',
		LMNP: 'LMNP',
		LMP: 'LMP',
		NOM_PROPRE: 'Nom propre'
	};

	const LOT_TYPE_LABELS: Record<string, string> = {
		APARTMENT: 'Appartement',
		HOUSE: 'Maison',
		COMMERCIAL: 'Local commercial',
		PARKING: 'Parking',
		CELLAR: 'Cave',
		BUILDING: 'Immeuble'
	};

	function formatEntityTypeLabel(entityType: string): string {
		return ENTITY_TYPE_LABELS[entityType] ?? entityType;
	}

	function formatLotTypeLabel(lotType: string): string {
		return LOT_TYPE_LABELS[lotType] ?? lotType;
	}

	const totalCollectedCents = $derived(
		groups.reduce((sum, group) => sum + group.collectedRentCents, 0)
	);
	const totalExpectedCents = $derived(
		groups.reduce((sum, group) => sum + group.expectedRentCents, 0)
	);
	const totalRemainingDueCents = $derived(
		groups.reduce((sum, group) => sum + group.remainingDueCents, 0)
	);

	$effect(() => {
		const selectedEntityId = workspaceStore.selectedEntityId;
		const periodMonth = workspaceStore.selectedPeriodMonth;
		const trigger = reloadTrigger;
		void trigger;

		let cancelled = false;
		isLoading = true;
		errorMessage = null;

		void (async () => {
			try {
				const allEntities = await apiRegistry.portfolio.listEntities();
				if (cancelled) return;
				workspaceStore.setEntities(allEntities);

				const targetEntities =
					selectedEntityId === 'ALL'
						? allEntities
						: allEntities.filter((entity) => entity.id === selectedEntityId);

				if (targetEntities.length === 0) {
					groups = [];
					totalOverdueLeasesCount = 0;
					totalOpenIncidentsCount = 0;
					return;
				}

				const [leases, tenants, incidents] = await Promise.all([
					apiRegistry.lease.listLeases(selectedEntityId),
					apiRegistry.lease.listTenants(false),
					apiRegistry.inspection.listIncidents().catch(() => [])
				]);
				if (cancelled) return;

				const propertiesByEntity: Record<string, PropertyDTO[]> = {};
				const rentCallsByEntity: Record<string, RentCallDTO[]> = {};
				const negativePositionsByEntity: Record<string, NegativePositionDTO[]> = {};

				await Promise.all(
					targetEntities.map(async (entity) => {
						const [properties, rentCalls, negativePositions] = await Promise.all([
							apiRegistry.portfolio.listProperties(entity.id),
							apiRegistry.lease.listEntityRentCalls(entity.id, periodMonth),
							apiRegistry.payment.listNegativePositions(entity.id).catch(() => [])
						]);
						propertiesByEntity[entity.id] = properties;
						rentCallsByEntity[entity.id] = rentCalls;
						negativePositionsByEntity[entity.id] = negativePositions;
					})
				);
				if (cancelled) return;

				const computedGroups = buildCockpitEntityGroups({
					periodMonth,
					entities: targetEntities,
					propertiesByEntity,
					leases,
					tenants,
					rentCallsByEntity,
					negativePositionsByEntity,
					incidents
				});

				groups = computedGroups;
				totalOverdueLeasesCount = Object.values(negativePositionsByEntity).reduce(
					(acc, list) => acc + list.filter((item) => item.total_unpaid_cents > 0).length,
					0
				);
				totalOpenIncidentsCount = computedGroups.reduce(
					(acc, group) =>
						acc + group.rows.reduce((rowAcc, row) => rowAcc + row.openIncidentsCount, 0),
					0
				);
			} catch (err) {
				if (cancelled) return;
				errorMessage =
					err instanceof ApiError
						? err.message
						: 'Impossible de charger les données du cockpit opérationnel.';
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

	function handlePeriodChange(event: Event) {
		const target = event.currentTarget as HTMLInputElement;
		if (target.value) {
			workspaceStore.setPeriodMonth(target.value);
		}
	}
</script>

<PageHeader
	title="Cockpit Opérationnel"
	subtitle="Pilotage par entité juridique et par bien — loyers, appels d'échéance, relances et incidents"
>
	{#snippet actions()}
		<div class="flex items-center gap-2">
			<IconCalendarEvent class="text-muted-foreground size-4" />
			<Input
				type="month"
				value={workspaceStore.selectedPeriodMonth}
				onchange={handlePeriodChange}
				class="h-8 w-40"
				aria-label="Mois de gestion"
			/>
		</div>
		<Button variant="outline" size="sm" onclick={() => (reloadTrigger += 1)}>
			<IconRefresh data-icon="inline-start" />
			Actualiser
		</Button>
	{/snippet}
</PageHeader>

{#if errorMessage}
	<Alert.Root variant="destructive">
		<IconAlertTriangle class="size-4" />
		<Alert.Title>Erreur de chargement</Alert.Title>
		<Alert.Description>{errorMessage}</Alert.Description>
	</Alert.Root>
{/if}

<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>
				Encaissé ({formatPeriodMonthFr(workspaceStore.selectedPeriodMonth)})
			</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{formatCentsToEuro(totalCollectedCents)}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-muted-foreground text-xs tabular-nums">
				Sur {formatCentsToEuro(totalExpectedCents)} attendus
			</p>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Reste à encaisser</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{formatCentsToEuro(totalRemainingDueCents)}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-muted-foreground text-xs">Échéances en attente de règlement</p>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Impayés à relancer</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{totalOverdueLeasesCount}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-muted-foreground text-xs">Baux en position débitrice</p>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Incidents & travaux ouverts</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{totalOpenIncidentsCount}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-muted-foreground text-xs">Sinistres ou interventions en cours</p>
		</Card.Content>
	</Card.Root>
</div>

{#if isLoading}
	<div class="flex flex-col gap-4">
		<Skeleton class="h-44 w-full" />
		<Skeleton class="h-44 w-full" />
	</div>
{:else if groups.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconBuildingBank class="size-5" />
					</Empty.Media>
					<Empty.Title>Aucune entité juridique configurée</Empty.Title>
					<Empty.Description>
						Créez votre première SCI, SARL de famille ou détention en nom propre pour structurer vos
						biens, baux et encaissements.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button href="/entities?action=create">
						<IconPlus data-icon="inline-start" />
						Créer une entité juridique
					</Button>
				</Empty.Content>
			</Empty.Root>
		</Card.Content>
	</Card.Root>
{:else}
	<div class="flex flex-col gap-5">
		{#each groups as group (group.entity.id)}
			<Card.Root>
				<Card.Header class="flex flex-col justify-between gap-3 border-b pb-3 sm:flex-row sm:items-center">
					<div class="flex flex-wrap items-center gap-2">
						<Card.Title class="text-sm font-semibold">{group.entity.name}</Card.Title>
						<Badge variant="outline">{formatEntityTypeLabel(group.entity.entity_type)}</Badge>
						{#if group.entity.siret}
							<span class="text-muted-foreground font-mono text-xs">SIRET {group.entity.siret}</span>
						{/if}
					</div>
					<div class="flex flex-wrap items-center gap-3">
						<span class="text-muted-foreground text-xs tabular-nums">
							Encaissé :
							<strong class="text-foreground font-semibold">
								{formatCentsToEuro(group.collectedRentCents)}
							</strong>
							/ {formatCentsToEuro(group.expectedRentCents)}
						</span>
						{#if group.generateCallsHref}
							<Button size="sm" variant="secondary" href={group.generateCallsHref}>
								Générer appels du mois ({group.uncalledLeasesCount}) →
							</Button>
						{/if}
					</div>
				</Card.Header>

				<Card.Content class="p-0">
					{#if group.rows.length === 0}
						<div class="flex items-center justify-between px-4 py-6">
							<p class="text-muted-foreground text-xs">
								Aucun bien immobilier rattaché à cette entité.
							</p>
							<Button
								size="sm"
								variant="outline"
								href={`/properties?entityId=${encodeURIComponent(group.entity.id)}&action=create`}
							>
								<IconPlus data-icon="inline-start" />
								Ajouter un bien →
							</Button>
						</div>
					{:else}
						<Table.Root>
							<Table.Header>
								<Table.Row>
									<Table.Head class="pl-4">Bien & Lot</Table.Head>
									<Table.Head>Locataire / Statut</Table.Head>
									<Table.Head class="text-right">Loyer CC</Table.Head>
									<Table.Head class="text-right">Reste dû</Table.Head>
									<Table.Head class="pr-4 text-right">Actions à traiter</Table.Head>
								</Table.Row>
							</Table.Header>
							<Table.Body>
								{#each group.rows as row (`${row.propertyId}-${row.lotId}`)}
									<Table.Row>
										<Table.Cell class="pl-4">
											<div class="flex flex-col gap-0.5">
												<span class="font-medium">
													{row.propertyLabel}
													{#if row.lotLabel !== row.propertyLabel}
														— {row.lotLabel}
													{/if}
												</span>
												<span class="text-muted-foreground text-[11px]">
													{formatLotTypeLabel(row.lotType)}
												</span>
											</div>
										</Table.Cell>

										<Table.Cell>
											{#if row.isVacant}
												<Badge variant="outline">Lot vacant</Badge>
											{:else}
												<span class="font-medium">{row.tenantName}</span>
											{/if}
										</Table.Cell>

										<Table.Cell class="text-right font-mono tabular-nums">
											{#if row.isVacant}
												—
											{:else}
												{formatCentsToEuro(row.monthlyRentIncludingChargesCents)}
											{/if}
										</Table.Cell>

										<Table.Cell class="text-right font-mono tabular-nums">
											{#if row.isVacant}
												—
											{:else if row.remainingDueCents > 0}
												<span class="text-destructive font-semibold">
													{formatCentsToEuro(row.remainingDueCents)}
												</span>
											{:else}
												<span class="text-muted-foreground">0,00 €</span>
											{/if}
										</Table.Cell>

										<Table.Cell class="pr-4 text-right">
											<div class="flex flex-wrap items-center justify-end gap-1.5">
												{#if row.isUpToDate && row.actions.length === 0}
													<Badge variant="secondary">
														<IconCheck class="size-3" />
														À jour
													</Badge>
												{:else}
													{#each row.actions as action (`${row.lotId}-${action.kind}-${action.href}`)}
														<Button size="sm" variant={action.variant} href={action.href}>
															{action.label}
														</Button>
													{/each}
												{/if}
											</div>
										</Table.Cell>
									</Table.Row>
								{/each}
							</Table.Body>
						</Table.Root>
					{/if}
				</Card.Content>
			</Card.Root>
		{/each}
	</div>
{/if}
