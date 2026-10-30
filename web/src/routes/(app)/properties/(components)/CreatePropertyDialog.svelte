<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import { toast } from 'svelte-sonner';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import {
		ACCOUNTING_CATEGORY_OPTIONS,
		formatAccountingCategoryLabel,
		formatPropertyTypeLabel,
		PROPERTY_TYPE_OPTIONS
	} from '#lib/modules/portfolio/labels';
	import type {
		AccountingCategory,
		LegalEntityDTO,
		PropertyDTO
	} from '#lib/modules/portfolio/types';

	let {
		open = $bindable(false),
		entities,
		defaultEntityId = '',
		onCreated
	}: {
		open?: boolean;
		entities: LegalEntityDTO[];
		defaultEntityId?: string;
		onCreated?: (property: PropertyDTO) => void;
	} = $props();

	let entityId = $state('');
	let label = $state('');
	let propertyType = $state('BUILDING');
	let accountingCategory = $state<AccountingCategory>('FIXED_ASSET');
	let address = $state('');
	let acquisitionEuro = $state('250000,00');
	let workStockEuro = $state('0,00');
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);

	$effect(() => {
		if (open) {
			const preferred =
				defaultEntityId && entities.some((entity) => entity.id === defaultEntityId)
					? defaultEntityId
					: (entities[0]?.id ?? '');
			if (preferred) {
				entityId = preferred;
			}
		}
	});

	function resetForm(): void {
		label = '';
		propertyType = 'BUILDING';
		accountingCategory = 'FIXED_ASSET';
		address = '';
		acquisitionEuro = '250000,00';
		workStockEuro = '0,00';
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
			<Dialog.Title>Nouveau bien immobilier</Dialog.Title>
			<Dialog.Description>
				Renseignez l’entité propriétaire, la qualification comptable et le prix de revient.
			</Dialog.Description>
		</Dialog.Header>

		{#if errorMessage}
			<Alert.Root variant="destructive">
				<IconAlertTriangle />
				<Alert.Title>Création impossible</Alert.Title>
				<Alert.Description>{errorMessage}</Alert.Description>
			</Alert.Root>
		{/if}

		<form
			method="POST"
			action="?/createProperty"
			use:enhance={() => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					isSubmitting = false;
					if (result.type === 'success') {
						const payload = result.data as
							{ createdProperty?: PropertyDTO; successMessage?: string } | undefined;
						toast.success(payload?.successMessage ?? 'Bien immobilier enregistré.');
						resetForm();
						open = false;
						await invalidateAll();
						if (payload?.createdProperty) {
							onCreated?.(payload.createdProperty);
						}
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Échec de la création du bien.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message =
							result.error?.message ?? 'Erreur serveur lors de la création du bien immobilier.';
						errorMessage = message;
						toast.error(message);
					}
				};
			}}
			class="flex flex-col gap-4"
		>
			<input type="hidden" name="legal_entity_id" value={entityId} />
			<input type="hidden" name="property_type" value={propertyType} />
			<input type="hidden" name="accounting_category" value={accountingCategory} />

			<Field.Group>
				<Field.Field>
					<Field.Label>Entité juridique propriétaire</Field.Label>
					<Select.Root
						type="single"
						value={entityId}
						onValueChange={(next) => {
							if (next) entityId = next;
						}}
					>
						<Select.Trigger class="w-full">
							{entities.find((entity) => entity.id === entityId)?.name ?? 'Sélectionner une entité'}
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
					<Field.Label for="create-property-label">Désignation du bien</Field.Label>
					<Input
						id="create-property-label"
						name="label"
						placeholder="Immeuble 12 rue de Rivoli"
						bind:value={label}
						required
					/>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label>Typologie</Field.Label>
						<Select.Root
							type="single"
							value={propertyType}
							onValueChange={(next) => {
								if (next) propertyType = next;
							}}
						>
							<Select.Trigger class="w-full">
								{formatPropertyTypeLabel(propertyType)}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each PROPERTY_TYPE_OPTIONS as option (option.value)}
										<Select.Item value={option.value} label={option.label}>
											{option.label}
										</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>

					<Field.Field>
						<Field.Label>Qualification PCG</Field.Label>
						<Select.Root
							type="single"
							value={accountingCategory}
							onValueChange={(next) => {
								if (next) accountingCategory = next;
							}}
						>
							<Select.Trigger class="w-full">
								{formatAccountingCategoryLabel(accountingCategory)}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each ACCOUNTING_CATEGORY_OPTIONS as option (option.value)}
										<Select.Item value={option.value} label={option.label}>
											{option.label}
										</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="create-property-address">Adresse postale complète</Field.Label>
					<Input
						id="create-property-address"
						name="address"
						placeholder="12 rue de Rivoli, 75004 Paris"
						bind:value={address}
						required
					/>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="create-property-acquisition">Valeur d’acquisition (€)</Field.Label>
						<Input
							id="create-property-acquisition"
							name="acquisition_euro"
							type="text"
							inputmode="decimal"
							bind:value={acquisitionEuro}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="create-property-work">Stock travaux (€)</Field.Label>
						<Input
							id="create-property-work"
							name="work_stock_euro"
							type="text"
							inputmode="decimal"
							bind:value={workStockEuro}
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
				<Button type="submit" disabled={isSubmitting}>
					{#if isSubmitting}
						<Spinner data-icon="inline-start" />
					{/if}
					Créer le bien immobilier
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
