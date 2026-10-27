<script lang="ts">
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconBuildingSkyscraper from '@tabler/icons-svelte/icons/building-skyscraper';
	import IconDoor from '@tabler/icons-svelte/icons/door';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
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
	import { formatCentsToEuro, parseEuroToCents } from '#lib/core/formatters/money';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import type { LeaseDTO } from '#lib/modules/lease/types';
	import type {
		AccountingCategory,
		LegalEntityDTO,
		PropertyDTO
	} from '#lib/modules/portfolio/types';
	import { apiRegistry } from '#lib/modules/registry';

	const PROPERTY_TYPES = [
		{ value: 'BUILDING', label: 'Immeuble de rapport' },
		{ value: 'APARTMENT', label: 'Appartement' },
		{ value: 'HOUSE', label: 'Maison individuelle' },
		{ value: 'COMMERCIAL', label: 'Local commercial / Bureaux' }
	];

	const ACCOUNTING_CATEGORIES: Array<{ value: AccountingCategory; label: string }> = [
		{ value: 'FIXED_ASSET', label: 'Immobilisation corporelle (PCG 213)' },
		{ value: 'MERCHANDISE', label: 'Stock marchand de biens (PCG 355/370)' }
	];

	const LOT_TYPES = [
		{ value: 'APARTMENT', label: 'Appartement' },
		{ value: 'COMMERCIAL', label: 'Local commercial' },
		{ value: 'PARKING', label: 'Parking / Box' },
		{ value: 'CELLAR', label: 'Cave' },
		{ value: 'HOUSE', label: 'Maison' }
	];

	let isLoading = $state(true);
	let errorMessage = $state<string | null>(null);
	let reloadCounter = $state(0);

	let entities = $state<LegalEntityDTO[]>([]);
	let properties = $state<PropertyDTO[]>([]);
	let leases = $state<LeaseDTO[]>([]);

	// Create Property Sheet
	let createPropertyOpen = $state(false);
	let propEntityId = $state('');
	let propLabel = $state('');
	let propType = $state('BUILDING');
	let propCategory = $state<AccountingCategory>('FIXED_ASSET');
	let propAddress = $state('');
	let propAcquisitionEuro = $state('250000,00');
	let propWorkStockEuro = $state('0,00');
	let isCreatingProperty = $state(false);

	// Create Lot Sheet
	let createLotOpen = $state(false);
	let lotPropertyId = $state('');
	let lotLabel = $state('');
	let lotType = $state('APARTMENT');
	let lotSurfaceM2 = $state('45');
	let lotOwnershipShare = $state('250');
	let lotFloor = $state('1er étage');
	let isCreatingLot = $state(false);

	let autoHandledUrl = $state(false);

	$effect(() => {
		const urlEntityId = page.url.searchParams.get('entityId');
		if (urlEntityId && urlEntityId !== workspaceStore.selectedEntityId) {
			workspaceStore.setEntityId(urlEntityId);
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

				if (targetEntities.length > 0 && !propEntityId) {
					propEntityId = targetEntities[0].id;
				}

				const [propLists, loadedLeases] = await Promise.all([
					Promise.all(
						targetEntities.map((entity) => apiRegistry.portfolio.listProperties(entity.id))
					),
					apiRegistry.lease.listLeases(selectedEntityId)
				]);
				if (cancelled) return;

				const flatProps = propLists.flat();
				properties = flatProps;
				leases = loadedLeases.filter((l) => l.active);

				if (flatProps.length > 0 && !lotPropertyId) {
					lotPropertyId = flatProps[0].id;
				}

				if (!autoHandledUrl) {
					const action = page.url.searchParams.get('action');
					const urlPropertyId = page.url.searchParams.get('propertyId');
					if (action === 'create') {
						createPropertyOpen = true;
						autoHandledUrl = true;
					} else if (urlPropertyId && flatProps.some((p) => p.id === urlPropertyId)) {
						lotPropertyId = urlPropertyId;
						createLotOpen = true;
						autoHandledUrl = true;
					}
				}
			} catch (err) {
				if (cancelled) return;
				errorMessage =
					err instanceof ApiError
						? err.message
						: 'Impossible de charger les biens et lots immobiliers.';
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

	const entityMap = $derived(new Map(entities.map((e) => [e.id, e])));
	const activeLeaseByLotId = $derived(new Map(leases.map((l) => [l.lot_id, l])));

	function openCreateLotForProperty(propertyId: string) {
		lotPropertyId = propertyId;
		lotLabel = '';
		createLotOpen = true;
	}

	async function handleCreateProperty(event: SubmitEvent) {
		event.preventDefault();
		if (!propEntityId || !propLabel.trim()) return;

		let acquisitionCents: number;
		let workStockCents: number;
		try {
			acquisitionCents = parseEuroToCents(propAcquisitionEuro || '0');
			workStockCents = parseEuroToCents(propWorkStockEuro || '0');
		} catch {
			toast.error('Montant d’acquisition ou de stock travaux invalide.');
			return;
		}

		isCreatingProperty = true;
		try {
			const created = await apiRegistry.portfolio.createProperty(propEntityId, {
				label: propLabel.trim(),
				property_type: propType,
				accounting_category: propCategory,
				address: propAddress.trim(),
				acquisition_cents: acquisitionCents,
				work_stock_cents: workStockCents
			});
			toast.success(`Bien « ${created.label} » enregistré.`);
			createPropertyOpen = false;
			propLabel = '';
			propAddress = '';
			lotPropertyId = created.id;
			reloadCounter += 1;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Échec de la création du bien.');
		} finally {
			isCreatingProperty = false;
		}
	}

	async function handleCreateLot(event: SubmitEvent) {
		event.preventDefault();
		if (!lotPropertyId || !lotLabel.trim()) return;

		isCreatingLot = true;
		try {
			const created = await apiRegistry.portfolio.createLot(lotPropertyId, {
				label: lotLabel.trim(),
				lot_type: lotType,
				surface_m2: Math.round(Number.parseFloat(lotSurfaceM2.replace(',', '.'))) || 0,
				ownership_share: Number.parseInt(lotOwnershipShare, 10) || 100,
				floor: lotFloor.trim()
			});
			toast.success(`Lot « ${created.label} » ajouté.`);
			createLotOpen = false;
			lotLabel = '';
			reloadCounter += 1;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Échec de la création du lot.');
		} finally {
			isCreatingLot = false;
		}
	}
</script>

<PageHeader
	title="Biens & Lots Immobiliers"
	subtitle="Actifs immobilisés (PCG 213), stocks marchand de biens (PCG 355) et découpage par lots locatifs"
>
	{#snippet actions()}
		{#if properties.length > 0}
			<Button size="sm" variant="outline" onclick={() => (createLotOpen = true)}>
				<IconDoor data-icon="inline-start" />
				Ajouter un lot
			</Button>
		{/if}
		<Button size="sm" onclick={() => (createPropertyOpen = true)} disabled={entities.length === 0}>
			<IconPlus data-icon="inline-start" />
			Nouveau bien
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

{#if isLoading}
	<div class="flex flex-col gap-4">
		<Skeleton class="h-40 w-full" />
		<Skeleton class="h-40 w-full" />
	</div>
{:else if properties.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconBuildingSkyscraper class="size-5" />
					</Empty.Media>
					<Empty.Title>Aucun bien immobilier enregistré</Empty.Title>
					<Empty.Description>
						Ajoutez vos immeubles, appartements ou locaux commerciaux puis créez leurs lots pour y
						rattacher des baux.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button onclick={() => (createPropertyOpen = true)}>
						<IconPlus data-icon="inline-start" />
						Ajouter un premier bien
					</Button>
				</Empty.Content>
			</Empty.Root>
		</Card.Content>
	</Card.Root>
{:else}
	<div class="flex flex-col gap-5">
		{#each properties as prop (prop.id)}
			<Card.Root>
				<Card.Header
					class="flex flex-col justify-between gap-2 border-b pb-3 sm:flex-row sm:items-center"
				>
					<div class="flex flex-col gap-1">
						<div class="flex flex-wrap items-center gap-2">
							<Card.Title class="text-sm font-semibold">{prop.label}</Card.Title>
							<Badge variant="outline">
								{entityMap.get(prop.legal_entity_id)?.name ?? 'Entité'}
							</Badge>
							<Badge variant={prop.accounting_category === 'FIXED_ASSET' ? 'secondary' : 'default'}>
								{prop.accounting_category === 'FIXED_ASSET'
									? 'Immobilisation (213)'
									: 'Stock marchand (355)'}
							</Badge>
						</div>
						<Card.Description>{prop.address}</Card.Description>
					</div>

					<div class="flex flex-wrap items-center gap-3">
						<div class="text-right text-xs">
							<span class="text-muted-foreground">VNC :</span>
							<strong class="font-mono font-semibold">
								{formatCentsToEuro(prop.total_book_value_cents)}
							</strong>
							<span class="block text-[11px] text-muted-foreground">
								Acq. {formatCentsToEuro(prop.acquisition_cents)} · Travaux
								{formatCentsToEuro(prop.work_stock_cents)}
							</span>
						</div>
						<Button size="sm" variant="secondary" onclick={() => openCreateLotForProperty(prop.id)}>
							<IconPlus data-icon="inline-start" />
							Ajouter un lot
						</Button>
					</div>
				</Card.Header>

				<Card.Content class="p-0">
					{#if !prop.lots || prop.lots.length === 0}
						<div class="flex items-center justify-between px-4 py-4">
							<p class="text-xs text-muted-foreground">
								Aucun lot découpé dans ce bien immobilier.
							</p>
							<Button size="sm" variant="outline" onclick={() => openCreateLotForProperty(prop.id)}>
								Créer le premier lot →
							</Button>
						</div>
					{:else}
						<Table.Root>
							<Table.Header>
								<Table.Row>
									<Table.Head class="pl-4">Désignation du lot</Table.Head>
									<Table.Head>Type</Table.Head>
									<Table.Head>Étage</Table.Head>
									<Table.Head class="text-right">Surface</Table.Head>
									<Table.Head class="text-right">Tantièmes</Table.Head>
									<Table.Head>Occupation</Table.Head>
									<Table.Head class="pr-4 text-right">Action</Table.Head>
								</Table.Row>
							</Table.Header>
							<Table.Body>
								{#each prop.lots as lot (lot.id)}
									{@const activeLease = activeLeaseByLotId.get(lot.id)}
									<Table.Row>
										<Table.Cell class="pl-4 font-medium">{lot.label}</Table.Cell>
										<Table.Cell class="text-xs">
											{LOT_TYPES.find((t) => t.value === lot.lot_type)?.label ?? lot.lot_type}
										</Table.Cell>
										<Table.Cell class="text-xs text-muted-foreground">
											{lot.floor || '—'}
										</Table.Cell>
										<Table.Cell class="text-right font-mono text-xs tabular-nums">
											{lot.surface_m2} m²
										</Table.Cell>
										<Table.Cell class="text-right font-mono text-xs tabular-nums">
											{lot.ownership_share} / 1000
										</Table.Cell>
										<Table.Cell>
											{#if activeLease}
												<Badge variant="secondary">Loué</Badge>
											{:else}
												<Badge variant="outline">Vacant</Badge>
											{/if}
										</Table.Cell>
										<Table.Cell class="pr-4 text-right">
											{#if activeLease}
												<Button
													size="sm"
													variant="outline"
													href={`/rents?entityId=${encodeURIComponent(prop.legal_entity_id)}&leaseId=${encodeURIComponent(activeLease.id)}`}
												>
													Voir loyers →
												</Button>
											{:else}
												<Button
													size="sm"
													variant="default"
													href={`/leases?entityId=${encodeURIComponent(prop.legal_entity_id)}&lotId=${encodeURIComponent(lot.id)}&action=create`}
												>
													Créer un bail →
												</Button>
											{/if}
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

<!-- Create Property Sheet -->
<Sheet.Root bind:open={createPropertyOpen}>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-md">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>Nouveau bien immobilier</Sheet.Title>
			<Sheet.Description>
				Renseignez l'entité propriétaire, la qualification comptable et le prix de revient.
			</Sheet.Description>
		</Sheet.Header>

		<form onsubmit={handleCreateProperty} class="flex flex-col gap-4 p-4">
			<Field.FieldGroup>
				<Field.Field>
					<Field.Label>Entité juridique propriétaire</Field.Label>
					<Select.Root
						type="single"
						value={propEntityId}
						onValueChange={(v) => {
							if (v) propEntityId = v;
						}}
					>
						<Select.Trigger class="w-full">
							{entityMap.get(propEntityId)?.name ?? 'Sélectionner une entité'}
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
					<Field.Label for="prop-label">Désignation du bien</Field.Label>
					<Input
						id="prop-label"
						placeholder="Immeuble 12 rue de Rivoli"
						bind:value={propLabel}
						required
					/>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label>Typologie</Field.Label>
						<Select.Root
							type="single"
							value={propType}
							onValueChange={(v) => {
								if (v) propType = v;
							}}
						>
							<Select.Trigger class="w-full">
								{PROPERTY_TYPES.find((t) => t.value === propType)?.label ?? propType}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each PROPERTY_TYPES as item (item.value)}
										<Select.Item value={item.value} label={item.label}>{item.label}</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>

					<Field.Field>
						<Field.Label>Qualification PCG</Field.Label>
						<Select.Root
							type="single"
							value={propCategory}
							onValueChange={(v) => {
								if (v) propCategory = v;
							}}
						>
							<Select.Trigger class="w-full">
								{ACCOUNTING_CATEGORIES.find((c) => c.value === propCategory)?.label ?? propCategory}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each ACCOUNTING_CATEGORIES as cat (cat.value)}
										<Select.Item value={cat.value} label={cat.label}>{cat.label}</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="prop-addr">Adresse postale complète</Field.Label>
					<Input
						id="prop-addr"
						placeholder="12 rue de Rivoli, 75004 Paris"
						bind:value={propAddress}
						required
					/>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="prop-acq">Valeur d'acquisition (€)</Field.Label>
						<Input
							id="prop-acq"
							type="text"
							inputmode="decimal"
							bind:value={propAcquisitionEuro}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="prop-work">Stock travaux (€)</Field.Label>
						<Input id="prop-work" type="text" inputmode="decimal" bind:value={propWorkStockEuro} />
					</Field.Field>
				</div>
			</Field.FieldGroup>

			<Button type="submit" disabled={isCreatingProperty}>
				{#if isCreatingProperty}
					<Spinner data-icon="inline-start" />
				{/if}
				Créer le bien immobilier
			</Button>
		</form>
	</Sheet.Content>
</Sheet.Root>

<!-- Create Lot Sheet -->
<Sheet.Root bind:open={createLotOpen}>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-md">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>Ajouter un lot locatif</Sheet.Title>
			<Sheet.Description>
				Créez une unité locative rattachée à un bien immobilier existant.
			</Sheet.Description>
		</Sheet.Header>

		<form onsubmit={handleCreateLot} class="flex flex-col gap-4 p-4">
			<Field.FieldGroup>
				<Field.Field>
					<Field.Label>Bien immobilier parent</Field.Label>
					<Select.Root
						type="single"
						value={lotPropertyId}
						onValueChange={(v) => {
							if (v) lotPropertyId = v;
						}}
					>
						<Select.Trigger class="w-full">
							{properties.find((p) => p.id === lotPropertyId)?.label ?? 'Sélectionner un bien'}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each properties as prop (prop.id)}
									<Select.Item value={prop.id} label={prop.label}>{prop.label}</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<Field.Field>
					<Field.Label for="lot-label">Référence / Numéro du lot</Field.Label>
					<Input
						id="lot-label"
						placeholder="Lot A2 — T2 Cour intérieure"
						bind:value={lotLabel}
						required
					/>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label>Type de lot</Field.Label>
						<Select.Root
							type="single"
							value={lotType}
							onValueChange={(v) => {
								if (v) lotType = v;
							}}
						>
							<Select.Trigger class="w-full">
								{LOT_TYPES.find((t) => t.value === lotType)?.label ?? lotType}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each LOT_TYPES as item (item.value)}
										<Select.Item value={item.value} label={item.label}>{item.label}</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>

					<Field.Field>
						<Field.Label for="lot-floor">Étage / Accès</Field.Label>
						<Input id="lot-floor" placeholder="2ème étage droite" bind:value={lotFloor} />
					</Field.Field>
				</div>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="lot-surf">Surface Loi Carrez / Habitable (m²)</Field.Label>
						<Input
							id="lot-surf"
							type="number"
							step="1"
							min="0"
							bind:value={lotSurfaceM2}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="lot-share">Quote-part (Tantièmes / 1000)</Field.Label>
						<Input id="lot-share" type="number" min="1" bind:value={lotOwnershipShare} required />
					</Field.Field>
				</div>
			</Field.FieldGroup>

			<Button type="submit" disabled={isCreatingLot}>
				{#if isCreatingLot}
					<Spinner data-icon="inline-start" />
				{/if}
				Enregistrer le lot
			</Button>
		</form>
	</Sheet.Content>
</Sheet.Root>
