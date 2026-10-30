<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconInfoCircle from '@tabler/icons-svelte/icons/info-circle';
	import { toast } from 'svelte-sonner';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Checkbox } from '#lib/components/ui/checkbox/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { formatLeaseTypeLabel, LEASE_TYPE_OPTIONS } from '#lib/modules/lease/labels';
	import type { LeaseDTO, LeaseType, TenantDTO } from '#lib/modules/lease/types';
	import type { LegalEntityDTO, LotDTO, PropertyDTO } from '#lib/modules/portfolio/types';

	let {
		open = $bindable(false),
		entities,
		properties,
		tenants,
		preselectedEntityId = '',
		preselectedLotId = '',
		onCreated
	}: {
		open?: boolean;
		entities: LegalEntityDTO[];
		properties: PropertyDTO[];
		tenants: TenantDTO[];
		preselectedEntityId?: string;
		preselectedLotId?: string;
		onCreated?: (lease: LeaseDTO) => void;
	} = $props();

	let entityId = $state('');
	let lotId = $state('');
	let tenantId = $state('');
	let leaseType = $state<LeaseType>('RESIDENTIAL_UNFURNISHED');
	let rentNetEuro = $state('800,00');
	let chargesEuro = $state('75,00');
	let subjectToVat = $state(false);
	let vatPercent = $state('20,00');
	let depositEuro = $state('800,00');
	let dueDay = $state('5');
	let gliPolicy = $state('');
	let guarantor = $state('');
	let startDate = $state(new Date().toISOString().slice(0, 10));
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);

	const activeTenants = $derived(tenants.filter((tenant) => !tenant.archived));
	const entityMap = $derived(new Map(entities.map((entity) => [entity.id, entity])));
	const tenantMap = $derived(new Map(activeTenants.map((tenant) => [tenant.id, tenant])));
	const lotInfoMap = $derived(
		new Map<string, { lot: LotDTO; property: PropertyDTO }>(
			properties.flatMap((property) =>
				(property.lots ?? []).map((lot) => [lot.id, { lot, property }] as const)
			)
		)
	);

	const availableLotsForEntity = $derived(
		properties
			.filter((property) => !entityId || property.legal_entity_id === entityId)
			.flatMap((property) => (property.lots ?? []).map((lot) => ({ lot, property })))
	);

	$effect(() => {
		if (open) {
			if (preselectedLotId && lotInfoMap.has(preselectedLotId)) {
				const target = lotInfoMap.get(preselectedLotId)!;
				entityId = target.property.legal_entity_id;
				lotId = preselectedLotId;
			} else {
				const preferredEntity =
					preselectedEntityId && entities.some((entity) => entity.id === preselectedEntityId)
						? preselectedEntityId
						: (entities[0]?.id ?? '');
				if (!entityId || !entities.some((entity) => entity.id === entityId)) {
					entityId = preferredEntity;
				}
			}

			if (!tenantId || !activeTenants.some((tenant) => tenant.id === tenantId)) {
				tenantId = activeTenants[0]?.id ?? '';
			}
		}
	});

	$effect(() => {
		if (
			open &&
			availableLotsForEntity.length > 0 &&
			!availableLotsForEntity.some((item) => item.lot.id === lotId)
		) {
			lotId = availableLotsForEntity[0].lot.id;
		}
	});

	function resetForm(): void {
		leaseType = 'RESIDENTIAL_UNFURNISHED';
		rentNetEuro = '800,00';
		chargesEuro = '75,00';
		subjectToVat = false;
		vatPercent = '20,00';
		depositEuro = '800,00';
		dueDay = '5';
		gliPolicy = '';
		guarantor = '';
		startDate = new Date().toISOString().slice(0, 10);
		errorMessage = null;
	}
</script>

<Dialog.Root
	bind:open
	onOpenChange={(next) => {
		if (!next) {
			errorMessage = null;
		}
	}}
>
	<Dialog.Content class="max-h-[90vh] overflow-y-auto sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>Nouveau contrat de bail</Dialog.Title>
			<Dialog.Description>
				Paramétrez le loyer hors charges, les provisions sur charges, l’option TVA et la police GLI.
			</Dialog.Description>
		</Dialog.Header>

		{#if activeTenants.length === 0}
			<Alert.Root>
				<IconInfoCircle />
				<Alert.Title>Aucun locataire actif disponible</Alert.Title>
				<Alert.Description>
					Créez d’abord une fiche locataire dans
					<a
						href={resolve('/(app)/leases/tenants')}
						class="font-medium underline underline-offset-4"
					>
						Locataires & RGPD
					</a>
					avant d’activer un contrat de bail.
				</Alert.Description>
			</Alert.Root>
		{/if}

		{#if availableLotsForEntity.length === 0}
			<Alert.Root>
				<IconInfoCircle />
				<Alert.Title>Aucun lot immobilier disponible</Alert.Title>
				<Alert.Description>
					Ajoutez un bien et un lot locatif dans
					<a href={resolve('/(app)/properties')} class="font-medium underline underline-offset-4">
						Biens & Lots
					</a>
					pour cette entité juridique.
				</Alert.Description>
			</Alert.Root>
		{/if}

		{#if errorMessage}
			<Alert.Root variant="destructive">
				<IconAlertTriangle />
				<Alert.Title>Création impossible</Alert.Title>
				<Alert.Description>{errorMessage}</Alert.Description>
			</Alert.Root>
		{/if}

		<form
			method="POST"
			action="?/createLease"
			use:enhance={() => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					isSubmitting = false;
					if (result.type === 'success') {
						const payload = result.data as
							{ createdLease?: LeaseDTO; successMessage?: string } | undefined;
						toast.success(payload?.successMessage ?? 'Contrat de bail créé.');
						resetForm();
						open = false;
						await invalidateAll();
						if (payload?.createdLease) {
							onCreated?.(payload.createdLease);
						}
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Impossible de créer le bail.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message =
							result.error?.message ?? 'Erreur serveur lors de la création du contrat de bail.';
						errorMessage = message;
						toast.error(message);
					}
				};
			}}
			class="flex flex-col gap-4"
		>
			<input type="hidden" name="legal_entity_id" value={entityId} />
			<input type="hidden" name="lot_id" value={lotId} />
			<input type="hidden" name="tenant_id" value={tenantId} />
			<input type="hidden" name="lease_type" value={leaseType} />
			<input type="hidden" name="subject_to_vat" value={subjectToVat ? 'true' : 'false'} />

			<Field.Group>
				<Field.Field>
					<Field.Label>Entité bailleresse</Field.Label>
					<Select.Root
						type="single"
						value={entityId}
						onValueChange={(next) => {
							if (next) entityId = next;
						}}
					>
						<Select.Trigger class="w-full">
							{entityMap.get(entityId)?.name ?? 'Sélectionner une entité'}
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
					<Field.Label>Lot immobilier loué</Field.Label>
					<Select.Root
						type="single"
						value={lotId}
						onValueChange={(next) => {
							if (next) lotId = next;
						}}
					>
						<Select.Trigger class="w-full">
							{#if lotInfoMap.get(lotId)}
								{lotInfoMap.get(lotId)?.property.label} — {lotInfoMap.get(lotId)?.lot.label}
							{:else}
								Sélectionner un lot
							{/if}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each availableLotsForEntity as item (item.lot.id)}
									<Select.Item
										value={item.lot.id}
										label={`${item.property.label} — ${item.lot.label}`}
									>
										{item.property.label} — {item.lot.label}
									</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<Field.Field>
					<Field.Label>Locataire titulaire</Field.Label>
					<Select.Root
						type="single"
						value={tenantId}
						onValueChange={(next) => {
							if (next) tenantId = next;
						}}
					>
						<Select.Trigger class="w-full">
							{tenantMap.get(tenantId)?.full_name ?? 'Sélectionner un locataire'}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each activeTenants as tenant (tenant.id)}
									<Select.Item value={tenant.id} label={tenant.full_name}>
										{tenant.full_name}
									</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<Field.Field>
					<Field.Label>Régime juridique du bail</Field.Label>
					<Select.Root
						type="single"
						value={leaseType}
						onValueChange={(next) => {
							if (next) leaseType = next;
						}}
					>
						<Select.Trigger class="w-full">
							{formatLeaseTypeLabel(leaseType)}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each LEASE_TYPE_OPTIONS as option (option.value)}
									<Select.Item value={option.value} label={option.label}>
										{option.label}
									</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-3">
					<Field.Field>
						<Field.Label for="create-lease-rent">Loyer HC (€)</Field.Label>
						<Input
							id="create-lease-rent"
							name="rent_net_euro"
							type="text"
							inputmode="decimal"
							bind:value={rentNetEuro}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="create-lease-charges">Provisions charges (€)</Field.Label>
						<Input
							id="create-lease-charges"
							name="charges_euro"
							type="text"
							inputmode="decimal"
							bind:value={chargesEuro}
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="create-lease-deposit">Dépôt garantie (€)</Field.Label>
						<Input
							id="create-lease-deposit"
							name="deposit_euro"
							type="text"
							inputmode="decimal"
							bind:value={depositEuro}
						/>
					</Field.Field>
				</div>

				<div class="flex items-center justify-between gap-3 rounded-md border p-3">
					<div class="flex items-center gap-2">
						<Checkbox id="create-lease-vat" bind:checked={subjectToVat} />
						<label for="create-lease-vat" class="text-xs font-medium">
							Bail soumis à la TVA (option 20%)
						</label>
					</div>
					{#if subjectToVat}
						<Input
							name="vat_percent"
							class="h-8 w-24 text-right"
							type="text"
							inputmode="decimal"
							bind:value={vatPercent}
							aria-label="Taux de TVA (%)"
						/>
					{/if}
				</div>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="create-lease-start">Date de prise d’effet</Field.Label>
						<Input
							id="create-lease-start"
							name="start_date"
							type="date"
							bind:value={startDate}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="create-lease-day">Jour d’échéance mensuel (1-28)</Field.Label>
						<Input
							id="create-lease-day"
							name="due_day"
							type="number"
							min="1"
							max="28"
							bind:value={dueDay}
							required
						/>
					</Field.Field>
				</div>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="create-lease-gli">N° police assurance GLI</Field.Label>
						<Input
							id="create-lease-gli"
							name="gli_policy_number"
							placeholder="GLI-AXA-2026-99"
							bind:value={gliPolicy}
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="create-lease-guarantor">Garant physique / Visale</Field.Label>
						<Input
							id="create-lease-guarantor"
							name="guarantor_name"
							placeholder="Caution solidaire"
							bind:value={guarantor}
						/>
					</Field.Field>
				</div>
			</Field.Group>

			<Dialog.Footer>
				<Button
					type="button"
					variant="outline"
					disabled={isSubmitting}
					onclick={() => {
						open = false;
					}}
				>
					Annuler
				</Button>
				<Button
					type="submit"
					disabled={isSubmitting ||
						activeTenants.length === 0 ||
						availableLotsForEntity.length === 0 ||
						!entityId ||
						!lotId ||
						!tenantId}
				>
					{#if isSubmitting}
						<Spinner data-icon="inline-start" />
					{/if}
					Activer le contrat de bail
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
