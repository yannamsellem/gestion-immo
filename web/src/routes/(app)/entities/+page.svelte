<script lang="ts">
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconBuildingBank from '@tabler/icons-svelte/icons/building-bank';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import IconUsers from '@tabler/icons-svelte/icons/users';
	import { toast } from 'svelte-sonner';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { ApiError } from '#lib/core/api/api-error';
	import {
		formatBasisPointsToPercent,
		formatCentsToEuro,
		parseEuroToCents
	} from '#lib/core/formatters/money';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import type {
		EntityPortfolioSummaryDTO,
		EntityType,
		LegalEntityDTO
	} from '#lib/modules/portfolio/types';
	import { apiRegistry } from '#lib/modules/registry';

	const ENTITY_TYPES: Array<{ value: EntityType; label: string }> = [
		{ value: 'SCI_IS', label: 'SCI à l’IS' },
		{ value: 'SCI_IR', label: 'SCI à l’IR' },
		{ value: 'SARL_FAMILLE', label: 'SARL de famille' },
		{ value: 'LMNP', label: 'LMNP' },
		{ value: 'NOM_PROPRE', label: 'Détention en nom propre' }
	];

	let isLoading = $state(true);
	let errorMessage = $state<string | null>(null);
	let reloadCounter = $state(0);

	let entities = $state<LegalEntityDTO[]>([]);
	let summariesByEntity = $state<Record<string, EntityPortfolioSummaryDTO>>({});
	let focusedEntityId = $state<string>('');

	// Create Entity Sheet
	let createEntityOpen = $state(false);
	let newName = $state('');
	let newEntityType = $state<EntityType>('SCI_IS');
	let newSiret = $state('');
	let newNafCode = $state('6820B');
	let newAddress = $state('');
	let newShareCapitalEuro = $state('1000,00');
	let newTotalShares = $state('100');
	let isCreatingEntity = $state(false);
	let autoOpenedCreate = $state(false);

	// Upsert Shareholder Sheet
	let shareholderSheetOpen = $state(false);
	let shEntityId = $state('');
	let shFullName = $state('');
	let shSharesCount = $state('50');
	let shCurrentAccountEuro = $state('0,00');
	let shRemunerationEuro = $state('0,00');
	let isSavingShareholder = $state(false);

	$effect(() => {
		if (!autoOpenedCreate && page.url.searchParams.get('action') === 'create') {
			createEntityOpen = true;
			autoOpenedCreate = true;
		}
	});

	$effect(() => {
		const selectedEntityId = workspaceStore.selectedEntityId;
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
						: allEntities.filter((e) => e.id === selectedEntityId);

				if (targetEntities.length > 0) {
					if (!focusedEntityId || !targetEntities.some((e) => e.id === focusedEntityId)) {
						focusedEntityId = targetEntities[0].id;
					}
					if (!shEntityId) {
						shEntityId = targetEntities[0].id;
					}
				} else {
					focusedEntityId = '';
				}

				const nextSummaries: Record<string, EntityPortfolioSummaryDTO> = {};
				await Promise.all(
					targetEntities.map(async (entity) => {
						const summary = await apiRegistry.portfolio.getEntitySummary(entity.id);
						nextSummaries[entity.id] = summary;
					})
				);
				if (cancelled) return;
				summariesByEntity = nextSummaries;
			} catch (err) {
				if (cancelled) return;
				errorMessage =
					err instanceof ApiError
						? err.message
						: 'Impossible de charger les entités juridiques et associés.';
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

	const visibleSummaries = $derived(Object.values(summariesByEntity));
	const focusedSummary = $derived(
		summariesByEntity[focusedEntityId] ?? visibleSummaries[0] ?? null
	);

	const totalAcquisitionCents = $derived(
		visibleSummaries.reduce((sum, s) => sum + s.total_acquisition_cents, 0)
	);
	const totalWorkStockCents = $derived(
		visibleSummaries.reduce((sum, s) => sum + s.total_work_stock_cents, 0)
	);
	const totalBookValueCents = $derived(
		visibleSummaries.reduce((sum, s) => sum + s.total_book_value_cents, 0)
	);
	const totalLotsCount = $derived(visibleSummaries.reduce((sum, s) => sum + s.total_lots_count, 0));

	function formatEntityTypeLabel(entityType: string): string {
		return ENTITY_TYPES.find((t) => t.value === entityType)?.label ?? entityType;
	}

	function openShareholderSheet(entityId: string) {
		shEntityId = entityId;
		shFullName = '';
		shSharesCount = '50';
		shCurrentAccountEuro = '0,00';
		shRemunerationEuro = '0,00';
		shareholderSheetOpen = true;
	}

	async function handleCreateEntity(event: SubmitEvent) {
		event.preventDefault();
		if (!newName.trim()) return;

		let shareCapitalCents = 0;
		try {
			shareCapitalCents = parseEuroToCents(newShareCapitalEuro || '0');
		} catch {
			toast.error('Montant du capital social invalide.');
			return;
		}

		const totalShares = Number.parseInt(newTotalShares, 10) || 100;
		isCreatingEntity = true;
		try {
			const created = await apiRegistry.portfolio.createEntity({
				name: newName.trim(),
				entity_type: newEntityType,
				siret: newSiret.trim(),
				naf_code: newNafCode.trim() || '6820B',
				address: newAddress.trim(),
				share_capital_cents: shareCapitalCents,
				total_shares: totalShares
			});
			toast.success(`Entité « ${created.name} » créée.`);
			createEntityOpen = false;
			newName = '';
			newSiret = '';
			newAddress = '';
			focusedEntityId = created.id;
			reloadCounter += 1;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Échec de la création de l’entité.');
		} finally {
			isCreatingEntity = false;
		}
	}

	async function handleUpsertShareholder(event: SubmitEvent) {
		event.preventDefault();
		if (!shEntityId || !shFullName.trim()) return;

		let currentAccountCents = 0;
		let remunerationPaidCents = 0;
		try {
			currentAccountCents = parseEuroToCents(shCurrentAccountEuro || '0');
			remunerationPaidCents = parseEuroToCents(shRemunerationEuro || '0');
		} catch {
			toast.error('Montant compte courant ou rémunération invalide.');
			return;
		}

		isSavingShareholder = true;
		try {
			await apiRegistry.portfolio.upsertShareholder(shEntityId, {
				full_name: shFullName.trim(),
				shares_count: Number.parseInt(shSharesCount, 10) || 0,
				current_account_cents: currentAccountCents,
				remuneration_paid_cents: remunerationPaidCents
			});
			toast.success('Associé enregistré (quote-part recalculée en points de base).');
			shareholderSheetOpen = false;
			reloadCounter += 1;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Impossible d’enregistrer l’associé.');
		} finally {
			isSavingShareholder = false;
		}
	}
</script>

<PageHeader
	title="Entités Juridiques & Associés (CERFA 2059-F)"
	subtitle="Structures sociales (SCI IS/IR, SARL, LMNP), valeur comptable du patrimoine et répartition du capital"
>
	{#snippet actions()}
		{#if focusedSummary}
			<Button
				size="sm"
				variant="outline"
				onclick={() => openShareholderSheet(focusedSummary.entity.id)}
			>
				<IconUsers data-icon="inline-start" />
				Ajouter un associé
			</Button>
		{/if}
		<Button size="sm" onclick={() => (createEntityOpen = true)}>
			<IconPlus data-icon="inline-start" />
			Nouvelle entité
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

<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Valeur d'acquisition brute</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{formatCentsToEuro(totalAcquisitionCents)}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-xs text-muted-foreground">Actifs immobilisés & stocks</p>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>En-cours & stocks travaux</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{formatCentsToEuro(totalWorkStockCents)}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-xs text-muted-foreground">Travaux capitalisés rattachés</p>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Valeur nette comptable (VNC)</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{formatCentsToEuro(totalBookValueCents)}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-xs text-muted-foreground">Acquisition + Stock travaux</p>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Lots en portefeuille</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">{totalLotsCount}</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-xs text-muted-foreground">Sur {visibleSummaries.length} entité(s)</p>
		</Card.Content>
	</Card.Root>
</div>

{#if isLoading}
	<Skeleton class="h-56 w-full" />
{:else if visibleSummaries.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconBuildingBank class="size-5" />
					</Empty.Media>
					<Empty.Title>Aucune entité juridique</Empty.Title>
					<Empty.Description>
						Créez une entité pour gérer son capital social, ses associés et ses biens immobiliers.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button onclick={() => (createEntityOpen = true)}>
						<IconPlus data-icon="inline-start" />
						Créer une entité juridique
					</Button>
				</Empty.Content>
			</Empty.Root>
		</Card.Content>
	</Card.Root>
{:else}
	<div class="grid gap-6 lg:grid-cols-12">
		<!-- Entities Table -->
		<Card.Root class="lg:col-span-7">
			<Card.Header class="border-b pb-3">
				<Card.Title class="text-sm font-semibold">Structures juridiques</Card.Title>
				<Card.Description>
					Cliquez sur une entité pour afficher la composition de son capital (CERFA 2059-F)
				</Card.Description>
			</Card.Header>
			<Card.Content class="p-0">
				<Table.Root>
					<Table.Header>
						<Table.Row>
							<Table.Head class="pl-4">Raison sociale</Table.Head>
							<Table.Head>Forme</Table.Head>
							<Table.Head class="text-right">Capital</Table.Head>
							<Table.Head class="text-right">VNC Patrimoine</Table.Head>
							<Table.Head class="pr-4 text-right">Associés</Table.Head>
						</Table.Row>
					</Table.Header>
					<Table.Body>
						{#each visibleSummaries as summary (summary.entity.id)}
							<Table.Row
								class="cursor-pointer"
								onclick={() => (focusedEntityId = summary.entity.id)}
							>
								<Table.Cell class="pl-4">
									<div class="flex flex-col">
										<span class="font-medium">{summary.entity.name}</span>
										<span class="font-mono text-[11px] text-muted-foreground">
											SIRET {summary.entity.siret || '—'} · NAF {summary.entity.naf_code || '6820B'}
										</span>
									</div>
								</Table.Cell>
								<Table.Cell>
									<Badge variant="outline">
										{formatEntityTypeLabel(summary.entity.entity_type)}
									</Badge>
								</Table.Cell>
								<Table.Cell class="text-right font-mono tabular-nums">
									{formatCentsToEuro(summary.entity.share_capital_cents)}
									<span class="block text-[10px] text-muted-foreground">
										{summary.entity.total_shares} parts
									</span>
								</Table.Cell>
								<Table.Cell class="text-right font-mono tabular-nums">
									{formatCentsToEuro(summary.total_book_value_cents)}
								</Table.Cell>
								<Table.Cell class="pr-4 text-right">
									<Button
										size="sm"
										variant={focusedSummary?.entity.id === summary.entity.id
											? 'default'
											: 'outline'}
										onclick={(e: MouseEvent) => {
											e.stopPropagation();
											focusedEntityId = summary.entity.id;
										}}
									>
										{summary.shareholders.length} associé(s)
									</Button>
								</Table.Cell>
							</Table.Row>
						{/each}
					</Table.Body>
				</Table.Root>
			</Card.Content>
		</Card.Root>

		<!-- CERFA 2059-F Shareholders Table -->
		<Card.Root class="lg:col-span-5">
			<Card.Header class="flex flex-row items-center justify-between border-b pb-3">
				<div>
					<Card.Title class="text-sm font-semibold">
						Associés & Comptes Courants — {focusedSummary?.entity.name ?? ''}
					</Card.Title>
					<Card.Description>
						Répartition en points de base (10000 = 100,00 %) pour le feuillet CERFA 2059-F
					</Card.Description>
				</div>
				{#if focusedSummary}
					<Button
						size="sm"
						variant="secondary"
						onclick={() => openShareholderSheet(focusedSummary.entity.id)}
					>
						<IconPlus data-icon="inline-start" />
						Associé
					</Button>
				{/if}
			</Card.Header>
			<Card.Content class="p-0">
				{#if !focusedSummary || focusedSummary.shareholders.length === 0}
					<p class="px-4 py-6 text-xs text-muted-foreground">
						Aucun associé renseigné pour cette structure.
					</p>
				{:else}
					<Table.Root>
						<Table.Header>
							<Table.Row>
								<Table.Head class="pl-4">Associé</Table.Head>
								<Table.Head class="text-right">Quote-part</Table.Head>
								<Table.Head class="pr-4 text-right">C/C Associé (455)</Table.Head>
							</Table.Row>
						</Table.Header>
						<Table.Body>
							{#each focusedSummary.shareholders as sh (sh.id)}
								<Table.Row>
									<Table.Cell class="pl-4">
										<div class="flex flex-col">
											<span class="font-medium">{sh.full_name}</span>
											<span class="text-[11px] text-muted-foreground">
												{sh.shares_count} parts
											</span>
										</div>
									</Table.Cell>
									<Table.Cell class="text-right font-mono tabular-nums">
										<Badge variant="secondary">
											{formatBasisPointsToPercent(sh.detention_basis_points)}
										</Badge>
									</Table.Cell>
									<Table.Cell class="pr-4 text-right font-mono tabular-nums">
										{formatCentsToEuro(sh.current_account_cents)}
									</Table.Cell>
								</Table.Row>
							{/each}
						</Table.Body>
					</Table.Root>
				{/if}
			</Card.Content>
		</Card.Root>
	</div>
{/if}

<!-- Create Legal Entity Sheet -->
<Sheet.Root bind:open={createEntityOpen}>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-md">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>Nouvelle entité juridique</Sheet.Title>
			<Sheet.Description>
				Configurez les informations légales et le capital social initial de la société ou détention.
			</Sheet.Description>
		</Sheet.Header>

		<form onsubmit={handleCreateEntity} class="flex flex-col gap-4 p-4">
			<Field.FieldGroup>
				<Field.Field>
					<Field.Label for="ent-name">Dénomination sociale / Nom</Field.Label>
					<Input
						id="ent-name"
						placeholder="SCI Patrimoine Haussmann"
						bind:value={newName}
						required
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label>Forme juridique & régime fiscal</Field.Label>
					<Select.Root
						type="single"
						value={newEntityType}
						onValueChange={(v) => {
							if (v) newEntityType = v;
						}}
					>
						<Select.Trigger class="w-full">
							{formatEntityTypeLabel(newEntityType)}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each ENTITY_TYPES as item (item.value)}
									<Select.Item value={item.value} label={item.label}>{item.label}</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="ent-siret">N° SIRET (14 chiffres)</Field.Label>
						<Input id="ent-siret" placeholder="89234567800014" bind:value={newSiret} />
					</Field.Field>

					<Field.Field>
						<Field.Label for="ent-naf">Code APE / NAF</Field.Label>
						<Input id="ent-naf" placeholder="6820B" bind:value={newNafCode} />
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="ent-addr">Adresse du siège social</Field.Label>
					<Input
						id="ent-addr"
						placeholder="14 boulevard Haussmann, 75009 Paris"
						bind:value={newAddress}
						required
					/>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="ent-cap">Capital social (€)</Field.Label>
						<Input
							id="ent-cap"
							type="text"
							inputmode="decimal"
							placeholder="1000,00"
							bind:value={newShareCapitalEuro}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="ent-shares">Nombre total de parts</Field.Label>
						<Input id="ent-shares" type="number" min="1" bind:value={newTotalShares} required />
					</Field.Field>
				</div>
			</Field.FieldGroup>

			<Button type="submit" disabled={isCreatingEntity}>
				{#if isCreatingEntity}
					<Spinner data-icon="inline-start" />
				{/if}
				Créer l'entité juridique
			</Button>
		</form>
	</Sheet.Content>
</Sheet.Root>

<!-- Upsert Shareholder Sheet -->
<Sheet.Root bind:open={shareholderSheetOpen}>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-md">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>Ajouter / modifier un associé</Sheet.Title>
			<Sheet.Description>
				La quote-part en points de base est calculée automatiquement selon le nombre total de parts
				de l'entité.
			</Sheet.Description>
		</Sheet.Header>

		<form onsubmit={handleUpsertShareholder} class="flex flex-col gap-4 p-4">
			<Field.FieldGroup>
				<Field.Field>
					<Field.Label>Entité juridique</Field.Label>
					<Select.Root
						type="single"
						value={shEntityId}
						onValueChange={(v) => {
							if (v) shEntityId = v;
						}}
					>
						<Select.Trigger class="w-full">
							{entities.find((e) => e.id === shEntityId)?.name ?? 'Sélectionner'}
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
					<Field.Label for="sh-name">Nom complet ou raison sociale de l'associé</Field.Label>
					<Input id="sh-name" placeholder="Jean Dupont" bind:value={shFullName} required />
				</Field.Field>

				<Field.Field>
					<Field.Label for="sh-shares">Nombre de parts détenues</Field.Label>
					<Input id="sh-shares" type="number" min="1" bind:value={shSharesCount} required />
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="sh-cc">Solde compte courant 455 (€)</Field.Label>
						<Input id="sh-cc" type="text" inputmode="decimal" bind:value={shCurrentAccountEuro} />
					</Field.Field>

					<Field.Field>
						<Field.Label for="sh-rem">Rémunération versée (€)</Field.Label>
						<Input id="sh-rem" type="text" inputmode="decimal" bind:value={shRemunerationEuro} />
					</Field.Field>
				</div>
			</Field.FieldGroup>

			<Button type="submit" disabled={isSavingShareholder}>
				{#if isSavingShareholder}
					<Spinner data-icon="inline-start" />
				{/if}
				Enregistrer l'associé
			</Button>
		</form>
	</Sheet.Content>
</Sheet.Root>
